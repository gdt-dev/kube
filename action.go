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

	gdterrors "github.com/gdt-dev/gdt/errors"
	"github.com/gdt-dev/gdt/parse"
	"github.com/gdt-dev/gdt/result"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// Action describes the the Kubernetes-specific action that is performed by the
// test.
type Action struct {
	// Create is a string containing a file path or raw YAML content describing
	// a Kubernetes resource to call `kubectl create` with.
	Create string `yaml:"create,omitempty"`
	// Apply is a string containing a file path or raw YAML content describing
	// a Kubernetes resource to call `kubectl apply` with.
	Apply string `yaml:"apply,omitempty"`
	// Delete is a string or object containing arguments to `kubectl delete`.
	//
	// It must be one of the following:
	//
	// - a file path to a manifest that will be read and the resources
	//   described in the manifest will be deleted
	// - a resource kind or kind alias, e.g. "pods", "po", followed by one of
	//   the following:
	//   * a space or `/` character followed by the resource name to delete
	//     only a resource with that name.
	// - an object with a `type` and optional `labels` field containing a label
	//   selector that should be used to select that `type` of resource.
	Delete *ResourceIdentifierOrFile `yaml:"delete,omitempty"`
	// Get is a string or object containing arguments to `kubectl get`.
	//
	// It must be one of the following:
	//
	// - a string with a resource kind or kind alias, e.g. "pods", "po",
	//   followed by one of the following:
	//   * a space or `/` character followed by the resource name to get only a
	//     resource with that name.
	// - an object with a `type` and optional `labels` field containing a label
	//   selector that should be used to select that `type` of resource.
	Get *ResourceIdentifier `yaml:"get,omitempty"`
}

// Do performs a single Kubernetes API command, returning the corresponding
// exit code and any runtime error. The `outbuf` and `errbuf` buffers will be
// filled with the contents of the command's stdout and stderr pipes
// respectively.
func (a *Action) Do(
	ctx context.Context,
	t *testing.T,
	c *connection,
) error {
	return nil
}

// doList performs a List() call for a resource.
func (a *Action) doList(
	ctx context.Context,
	t *testing.T,
	c *connection,
	res schema.GroupVersionResource,
	namespace string,
) (*unstructured.UnstructuredList, error) {
	opts := metav1.ListOptions{}
	withlabels := a.Get.Labels()
	if withlabels != nil {
		// We already validated the label selector during parse-time
		opts.LabelSelector = labels.Set(withlabels).String()
	}
	return c.client.Resource(res).Namespace(namespace).List(
		ctx, opts,
	)
}

// doGet performs a Get() call for a resource.
func (a *Action) doGet(
	ctx context.Context,
	t *testing.T,
	c *connection,
	res schema.GroupVersionResource,
	name string,
	namespace string,
) (*unstructured.Unstructured, error) {
	return c.client.Resource(res).Namespace(namespace).Get(
		ctx,
		name,
		metav1.GetOptions{},
	)
}

// splitKindName returns the Kind for a supplied `Get` or `Delete` command
// where the user can specify either a resource kind or alias, e.g. "pods" or
// "po", or the resource kind followed by a forward slash and a resource name.
func splitKindName(subject string) (string, string) {
	kind, name, _ := strings.Cut(subject, "/")
	return kind, name
}

// doCreate executes a Create() call against the Kubernetes API server.
func (a *Action) doCreate(
	ctx context.Context,
	t *testing.T,
	c *connection,
) ([]*unstructured.Unstructured, error) {
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
	res := []*unstructured.Unstructured{}
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
		if err != nil {
			return nil, err
		}
		res = append(res, obj)
	}
	return res, nil
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
