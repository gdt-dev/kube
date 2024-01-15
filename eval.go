// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	gdtcontext "github.com/gdt-dev/gdt/context"
	"github.com/gdt-dev/gdt/debug"
	gdterrors "github.com/gdt-dev/gdt/errors"
	"github.com/gdt-dev/gdt/parse"
	"github.com/gdt-dev/gdt/result"
	gdttypes "github.com/gdt-dev/gdt/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	// defaultGetTimeout is used as a retry max time if the spec's Timeout has
	// not been specified.
	defaultGetTimeout = time.Second * 5
	// fieldManagerName is the identifier for the field manager we specify in
	// Apply requests.
	fieldManagerName = "gdt-kube"
)

// Run executes the test described by the Kubernetes test. A new Kubernetes
// client request is made during this call.
func (s *Spec) Eval(ctx context.Context, t *testing.T) *result.Result {
	c, err := s.connect(ctx)
	if err != nil {
		return result.New(
			result.WithRuntimeError(ConnectError(err)),
		)
	}
	var res *result.Result
	t.Run(s.Title(), func(t *testing.T) {
		if s.Kube.Get != nil {
			res = s.get(ctx, t, c)
		}
		if s.Kube.Create != "" {
			res = s.create(ctx, t, c)
		}
		if s.Kube.Delete != nil {
			res = s.delete(ctx, t, c)
		}
		if s.Kube.Apply != "" {
			res = s.apply(ctx, t, c)
		}
		for _, failure := range res.Failures() {
			if gdtcontext.TimedOut(ctx, failure) {
				to := s.Timeout
				if to != nil && !to.Expected {
					t.Error(gdterrors.TimeoutExceeded(to.After, failure))
				}
			} else {
				t.Error(failure)
			}
		}
	})
	return res
}

// get executes either a List() or a Get() call against the Kubernetes API
// server and evaluates any assertions that have been set for the returned
// results.
func (s *Spec) get(
	ctx context.Context,
	t *testing.T,
	c *connection,
) *result.Result {
	kind, name := s.Kube.Get.KindName()
	gvk := schema.GroupVersionKind{
		Kind: kind,
	}
	res, err := c.gvrFromGVK(gvk)
	a := newAssertions(s.Assert, err, nil)
	if !a.OK() {
		return result.New(result.WithFailures(a.Failures()...))
	}

	// if the Spec has no timeout, default it to a reasonable value
	var cancel context.CancelFunc
	_, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		ctx, cancel = context.WithTimeout(ctx, defaultGetTimeout)
		defer cancel()
	}

	// retry the Get/List and test the assertions until they succeed, there is
	// a terminal failure, or the timeout expires.
	bo := backoff.WithContext(backoff.NewExponentialBackOff(), ctx)
	ticker := backoff.NewTicker(bo)
	attempts := 0
	start := time.Now().UTC()
	for tick := range ticker.C {
		attempts++
		after := tick.Sub(start)

		if name == "" {
			a = s.doList(ctx, t, c, res, s.Namespace())
		} else {
			a = s.doGet(ctx, t, c, res, name, s.Namespace())
		}
		success := a.OK()
		debug.Println(
			ctx, t, "%s (try %d after %s) ok: %v",
			s.Title(), attempts, after, success,
		)
		if success {
			ticker.Stop()
			break
		}
		for _, f := range a.Failures() {
			debug.Println(
				ctx, t, "%s (try %d after %s) failure: %s",
				s.Title(), attempts, after, f,
			)
		}
	}
	return result.New(result.WithFailures(a.Failures()...))
}

// doList performs the List() call and assertion check for a supplied resource
// kind and name
func (s *Spec) doList(
	ctx context.Context,
	t *testing.T,
	c *connection,
	res schema.GroupVersionResource,
	namespace string,
) gdttypes.Assertions {
	opts := metav1.ListOptions{}
	withlabels := s.Kube.Get.Labels()
	if withlabels != nil {
		// We already validated the label selector during parse-time
		opts.LabelSelector = labels.Set(withlabels).String()
	}
	list, err := c.client.Resource(res).Namespace(namespace).List(
		ctx, opts,
	)
	return newAssertions(s.Assert, err, list)
}

// doGet performs the Get() call and assertion check for a supplied resource
// kind and name
func (s *Spec) doGet(
	ctx context.Context,
	t *testing.T,
	c *connection,
	res schema.GroupVersionResource,
	name string,
	namespace string,
) gdttypes.Assertions {
	obj, err := c.client.Resource(res).Namespace(namespace).Get(
		ctx,
		name,
		metav1.GetOptions{},
	)
	return newAssertions(s.Assert, err, obj)
}

// splitKindName returns the Kind for a supplied `Get` or `Delete` command
// where the user can specify either a resource kind or alias, e.g. "pods" or
// "po", or the resource kind followed by a forward slash and a resource name.
func splitKindName(subject string) (string, string) {
	kind, name, _ := strings.Cut(subject, "/")
	return kind, name
}

// create executes a Create() call against the Kubernetes API server and
// evaluates any assertions that have been set for the returned results.
func (s *Spec) create(
	ctx context.Context,
	t *testing.T,
	c *connection,
) *result.Result {
	var err error
	var r io.Reader
	if probablyFilePath(s.Kube.Create) {
		path := s.Kube.Create
		f, err := os.Open(path)
		if err != nil {
			// This should never happen because we check during parse time
			// whether the file can be opened.
			rterr := fmt.Errorf("%w: %s", gdterrors.RuntimeError, err)
			return result.New(result.WithRuntimeError(rterr))
		}
		defer f.Close()
		r = f
	} else {
		// Consider the string to be YAML/JSON content and marshal that into an
		// unstructured.Unstructured that we then pass to Create()
		r = strings.NewReader(s.Kube.Create)
	}

	objs, err := unstructuredFromReader(r)
	if err != nil {
		rterr := fmt.Errorf("%w: %s", gdterrors.RuntimeError, err)
		return result.New(result.WithRuntimeError(rterr))
	}
	for _, obj := range objs {
		gvk := obj.GetObjectKind().GroupVersionKind()
		ns := obj.GetNamespace()
		if ns == "" {
			ns = s.Namespace()
		}
		res, err := c.gvrFromGVK(gvk)
		a := newAssertions(s.Assert, err, nil)
		if !a.OK() {
			return result.New(result.WithFailures(a.Failures()...))
		}
		obj, err := c.client.Resource(res).Namespace(ns).Create(
			ctx,
			obj,
			metav1.CreateOptions{},
		)
		// TODO(jaypipes): Clearly this is applying the same assertion to each
		// object that was created, which is wrong. When I add the polymorphism
		// to the Assertions struct, I will modify this block to look for an
		// indexed set of error assertions.
		a = newAssertions(s.Assert, err, obj)
		return result.New(result.WithFailures(a.Failures()...))
	}
	return nil
}

// apply executes an Apply() call against the Kubernetes API server and
// evaluates any assertions that have been set for the returned results.
func (s *Spec) apply(
	ctx context.Context,
	t *testing.T,
	c *connection,
) *result.Result {
	var err error
	var r io.Reader
	if probablyFilePath(s.Kube.Apply) {
		path := s.Kube.Apply
		f, err := os.Open(path)
		if err != nil {
			// This should never happen because we check during parse time
			// whether the file can be opened.
			rterr := fmt.Errorf("%w: %s", gdterrors.RuntimeError, err)
			return result.New(result.WithRuntimeError(rterr))
		}
		defer f.Close()
		r = f
	} else {
		// Consider the string to be YAML/JSON content and marshal that into an
		// unstructured.Unstructured that we then pass to Apply()
		r = strings.NewReader(s.Kube.Apply)
	}

	objs, err := unstructuredFromReader(r)
	if err != nil {
		rterr := fmt.Errorf("%w: %s", gdterrors.RuntimeError, err)
		return result.New(result.WithRuntimeError(rterr))
	}
	for _, obj := range objs {
		gvk := obj.GetObjectKind().GroupVersionKind()
		ns := obj.GetNamespace()
		if ns == "" {
			ns = s.Namespace()
		}
		res, err := c.gvrFromGVK(gvk)
		a := newAssertions(s.Assert, err, nil)
		if !a.OK() {
			return result.New(result.WithFailures(a.Failures()...))
		}
		obj, err := c.client.Resource(res).Namespace(ns).Apply(
			ctx,
			// NOTE(jaypipes): Not sure why a separate name argument is
			// necessary considering `obj` is of type
			// `*unstructured.Unstructured` and therefore has the `GetName()`
			// method...
			obj.GetName(),
			obj,
			// TODO(jaypipes): Not sure if this hard-coded options struct is
			// always going to work. Maybe add ability to control it?
			metav1.ApplyOptions{FieldManager: fieldManagerName, Force: true},
		)
		// TODO(jaypipes): Clearly this is applying the same assertion to each
		// object that was applied, which is wrong. When I add the polymorphism
		// to the Assertions struct, I will modify this block to look for an
		// indexed set of error assertions.
		a = newAssertions(s.Assert, err, obj)
		return result.New(result.WithFailures(a.Failures()...))
	}
	return nil
}

// unstructuredFromReader attempts to read the supplied io.Reader and unmarshal
// the content into zero or more unstructured.Unstructured objects
func unstructuredFromReader(
	r io.Reader,
) ([]*unstructured.Unstructured, error) {
	yr := yaml.NewYAMLReader(bufio.NewReader(r))

	objs := []*unstructured.Unstructured{}
	for {
		raw, err := yr.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		data := parse.ExpandWithFixedDoubleDollar(string(raw))

		obj := &unstructured.Unstructured{}
		decoder := yaml.NewYAMLOrJSONDecoder(
			bytes.NewBuffer([]byte(data)), len(data),
		)
		if err = decoder.Decode(obj); err != nil {
			return nil, err
		}
		if obj.GetObjectKind().GroupVersionKind().Kind != "" {
			objs = append(objs, obj)
		}
	}

	return objs, nil
}

// delete executes either Delete() call against the Kubernetes API server
// and evaluates any assertions that have been set for the returned results.
func (s *Spec) delete(
	ctx context.Context,
	t *testing.T,
	c *connection,
) *result.Result {
	if s.Kube.Delete.FilePath() != "" {
		path := s.Kube.Delete.FilePath()
		f, err := os.Open(path)
		if err != nil {
			// This should never happen because we check during parse time
			// whether the file can be opened.
			rterr := fmt.Errorf("%w: %s", gdterrors.RuntimeError, err)
			return result.New(result.WithRuntimeError(rterr))
		}
		defer f.Close()
		objs, err := unstructuredFromReader(f)
		if err != nil {
			rterr := fmt.Errorf("%w: %s", gdterrors.RuntimeError, err)
			return result.New(result.WithRuntimeError(rterr))
		}
		for _, obj := range objs {
			gvk := obj.GetObjectKind().GroupVersionKind()
			res, err := c.gvrFromGVK(gvk)
			a := newAssertions(s.Assert, err, nil)
			if !a.OK() {
				return result.New(result.WithFailures(a.Failures()...))
			}
			name := obj.GetName()
			ns := obj.GetNamespace()
			if ns == "" {
				ns = s.Namespace()
			}
			// TODO(jaypipes): Clearly this is applying the same assertion to each
			// object that was deleted, which is wrong. When I add the polymorphism
			// to the Assertions struct, I will modify this block to look for an
			// indexed set of error assertions.
			r := s.doDelete(ctx, t, c, res, name, ns)
			if len(r.Failures()) > 0 {
				return r
			}
		}
		return result.New()
	}

	kind, name := s.Kube.Delete.KindName()
	gvk := schema.GroupVersionKind{
		Kind: kind,
	}
	res, err := c.gvrFromGVK(gvk)
	a := newAssertions(s.Assert, err, nil)
	if !a.OK() {
		return result.New(result.WithFailures(a.Failures()...))
	}
	if name == "" {
		return s.doDeleteCollection(ctx, t, c, res, s.Namespace())
	}
	return s.doDelete(ctx, t, c, res, name, s.Namespace())
}

// doDelete performs the Delete() call and assertion check for a supplied
// resource kind and name
func (s *Spec) doDelete(
	ctx context.Context,
	t *testing.T,
	c *connection,
	res schema.GroupVersionResource,
	name string,
	namespace string,
) *result.Result {
	err := c.client.Resource(res).Namespace(namespace).Delete(
		ctx,
		name,
		metav1.DeleteOptions{},
	)
	a := newAssertions(s.Assert, err, nil)
	return result.New(result.WithFailures(a.Failures()...))
}

// doDeleteCollection performs the DeleteCollection() call and assertion check
// for a supplied resource kind
func (s *Spec) doDeleteCollection(
	ctx context.Context,
	t *testing.T,
	c *connection,
	res schema.GroupVersionResource,
	namespace string,
) *result.Result {
	listOpts := metav1.ListOptions{}
	withlabels := s.Kube.Delete.Labels()
	if withlabels != nil {
		// We already validated the label selector during parse-time
		listOpts.LabelSelector = labels.Set(withlabels).String()
	}
	err := c.client.Resource(res).Namespace(namespace).DeleteCollection(
		ctx,
		metav1.DeleteOptions{},
		listOpts,
	)
	a := newAssertions(s.Assert, err, nil)
	return result.New(result.WithFailures(a.Failures()...))
}
