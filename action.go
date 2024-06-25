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

	"github.com/gdt-dev/gdt/api"
	"github.com/gdt-dev/gdt/debug"
	"github.com/gdt-dev/gdt/parse"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	// fieldManagerName is the identifier for the field manager we specify in
	// Apply requests.
	fieldManagerName = "gdt-kube"
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

// getCommand returns a string of the command that the action will end up
// performing.
func (a *Action) getCommand() string {
	if a.Get != nil {
		return "get"
	}
	if a.Create != "" {
		return "create"
	}
	if a.Delete != nil {
		return "delete"
	}
	if a.Apply != "" {
		return "apply"
	}
	return "unknown"
}

// Do performs a single kube command, returning any runtime error.
//
// `kubeErr` will be filled with any error received from the Kubernetes client
// call.
//
// `out` will be filled with the contents of the command's output, if any. When
// the command is a Get, `out` will be a `*unstructured.Unstructured`. When the
// command is a List, `out` will be a `*unstructured.UnstructuredList`.
func (a *Action) Do(
	ctx context.Context,
	c *connection,
	ns string,
	out *interface{},
) error {
	cmd := a.getCommand()

	switch cmd {
	case "get":
		return a.get(ctx, c, ns, out)
	case "create":
		return a.create(ctx, c, ns, out)
	case "delete":
		return a.delete(ctx, c, ns)
	case "apply":
		return a.apply(ctx, c, ns, out)
	default:
		return fmt.Errorf("unknown command")
	}
}

// get executes either a List() or a Get() call against the Kubernetes API
// server, returning any error returned from the client call and populating
// `out` with the response value.
func (a *Action) get(
	ctx context.Context,
	c *connection,
	ns string,
	out *interface{},
) error {
	kind, name := a.Get.KindName()
	gvk := schema.GroupVersionKind{
		Kind: kind,
	}
	res, err := c.gvrFromGVK(gvk)
	if err != nil {
		return err
	}
	if name == "" {
		list, err := a.doList(ctx, c, res, ns)
		if err == nil {
			*out = list
		}
		return err
	} else {
		obj, err := a.doGet(ctx, c, res, ns, name)
		if err == nil {
			*out = obj
		}
		return err
	}
}

// doList performs the List() call for a supplied resource kind
func (a *Action) doList(
	ctx context.Context,
	c *connection,
	res schema.GroupVersionResource,
	ns string,
) (*unstructured.UnstructuredList, error) {
	resName := res.Resource
	labelSelString := ""
	opts := metav1.ListOptions{}
	withlabels := a.Get.Labels()
	if withlabels != nil {
		// We already validated the label selector during parse-time
		labelsStr := labels.Set(withlabels).String()
		labelSelString = fmt.Sprintf(" (labels: %s)", labelsStr)
		opts.LabelSelector = labelsStr
	}
	if c.resourceNamespaced(res) {
		debug.Println(
			ctx, "kube.get: %s%s (ns: %s)",
			resName, labelSelString, ns,
		)
		return c.client.Resource(res).Namespace(ns).List(
			ctx, opts,
		)
	}
	debug.Println(
		ctx, "kube.get: %s%s (non-namespaced resource)",
		resName, labelSelString,
	)
	return c.client.Resource(res).List(
		ctx, opts,
	)
}

// doGet performs the Get() call for a supplied resource kind and name
func (a *Action) doGet(
	ctx context.Context,
	c *connection,
	res schema.GroupVersionResource,
	ns string,
	name string,
) (*unstructured.Unstructured, error) {
	resName := res.Resource
	if c.resourceNamespaced(res) {
		debug.Println(
			ctx, "kube.get: %s/%s (ns: %s)",
			resName, name, ns,
		)
		return c.client.Resource(res).Namespace(ns).Get(
			ctx,
			name,
			metav1.GetOptions{},
		)
	}
	debug.Println(
		ctx, "kube.get: %s/%s (non-namespaced resource)",
		resName, name,
	)
	return c.client.Resource(res).Get(
		ctx,
		name,
		metav1.GetOptions{},
	)
}

// create executes a Create() call against the Kubernetes API server and
// evaluates any assertions that have been set for the returned results.
func (a *Action) create(
	ctx context.Context,
	c *connection,
	ns string,
	out *interface{},
) error {
	var err error
	var r io.Reader
	if probablyFilePath(a.Create) {
		path := a.Create
		f, err := os.Open(path)
		if err != nil {
			// This should never happen because we check during parse time
			// whether the file can be opened.
			rterr := fmt.Errorf("%w: %s", api.RuntimeError, err)
			return rterr
		}
		defer f.Close()
		r = f
	} else {
		// Consider the string to be YAML/JSON content and marshal that into an
		// unstructured.Unstructured that we then pass to Create()
		r = strings.NewReader(a.Create)
	}

	// This is what we return to the caller via the `out` param. It contains
	// all of the created objects. This is NOT an
	// `unstructured.UnstructuredList` because we may have created multiple
	// objects of different Kinds.
	createdObjs := []*unstructured.Unstructured{}

	objs, err := unstructuredFromReader(r)
	if err != nil {
		rterr := fmt.Errorf("%w: %s", api.RuntimeError, err)
		return rterr
	}
	for _, obj := range objs {
		gvk := obj.GetObjectKind().GroupVersionKind()
		ons := obj.GetNamespace()
		if ons == "" {
			ons = ns
		}
		res, err := c.gvrFromGVK(gvk)
		if err != nil {
			return err
		}
		resName := res.Resource
		debug.Println(ctx, "kube.create: %s (ns: %s)", resName, ons)
		obj, err := c.client.Resource(res).Namespace(ons).Create(
			ctx,
			obj,
			metav1.CreateOptions{},
		)
		if err != nil {
			return err
		}
		createdObjs = append(createdObjs, obj)
	}
	*out = createdObjs
	return nil
}

// apply executes an Apply() call against the Kubernetes API server and
// evaluates any assertions that have been set for the returned results.
func (a *Action) apply(
	ctx context.Context,
	c *connection,
	ns string,
	out *interface{},
) error {
	var err error
	var r io.Reader
	if probablyFilePath(a.Apply) {
		path := a.Apply
		f, err := os.Open(path)
		if err != nil {
			// This should never happen because we check during parse time
			// whether the file can be opened.
			rterr := fmt.Errorf("%w: %s", api.RuntimeError, err)
			return rterr
		}
		defer f.Close()
		r = f
	} else {
		// Consider the string to be YAML/JSON content and marshal that into an
		// unstructured.Unstructured that we then pass to Apply()
		r = strings.NewReader(a.Apply)
	}

	// This is what we return to the caller via the `out` param. It contains
	// all of the applied objects. This is NOT an
	// `unstructured.UnstructuredList` because we may have applied multiple
	// objects of different Kinds.
	appliedObjs := []*unstructured.Unstructured{}

	objs, err := unstructuredFromReader(r)
	if err != nil {
		rterr := fmt.Errorf("%w: %s", api.RuntimeError, err)
		return rterr
	}
	for _, obj := range objs {
		gvk := obj.GetObjectKind().GroupVersionKind()
		ons := obj.GetNamespace()
		if ons == "" {
			ons = ns
		}
		res, err := c.gvrFromGVK(gvk)
		if err != nil {
			return err
		}
		resName := res.Resource
		debug.Println(ctx, "kube.apply: %s (ns: %s)", resName, ons)
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
		if err != nil {
			return err
		}
		appliedObjs = append(appliedObjs, obj)
	}
	*out = appliedObjs
	return nil
}

// delete executes either Delete() call against the Kubernetes API server
// and evaluates any assertions that have been set for the returned results.
func (a *Action) delete(
	ctx context.Context,
	c *connection,
	ns string,
) error {
	if a.Delete.FilePath() != "" {
		path := a.Delete.FilePath()
		f, err := os.Open(path)
		if err != nil {
			// This should never happen because we check during parse time
			// whether the file can be opened.
			rterr := fmt.Errorf("%w: %s", api.RuntimeError, err)
			return rterr
		}
		defer f.Close()
		objs, err := unstructuredFromReader(f)
		if err != nil {
			rterr := fmt.Errorf("%w: %s", api.RuntimeError, err)
			return rterr
		}
		for _, obj := range objs {
			gvk := obj.GetObjectKind().GroupVersionKind()
			res, err := c.gvrFromGVK(gvk)
			if err != nil {
				return err
			}
			name := obj.GetName()
			ons := obj.GetNamespace()
			if ons == "" {
				ons = ns
			}
			if err = a.doDelete(ctx, c, res, name, ns); err != nil {
				return err
			}
		}
		return nil
	}

	kind, name := a.Delete.KindName()
	gvk := schema.GroupVersionKind{
		Kind: kind,
	}
	res, err := c.gvrFromGVK(gvk)
	if err != nil {
		return err
	}
	if name == "" {
		return a.doDeleteCollection(ctx, c, res, ns)
	}
	return a.doDelete(ctx, c, res, ns, name)
}

// doDelete performs the Delete() call on a kind and name
func (a *Action) doDelete(
	ctx context.Context,
	c *connection,
	res schema.GroupVersionResource,
	ns string,
	name string,
) error {
	resName := res.Resource
	debug.Println(
		ctx, "kube.delete: %s/%s (ns: %s)",
		resName, name, ns,
	)
	return c.client.Resource(res).Namespace(ns).Delete(
		ctx,
		name,
		metav1.DeleteOptions{},
	)
}

// doDeleteCollection performs the DeleteCollection() call for the supplied
// resource kind
func (a *Action) doDeleteCollection(
	ctx context.Context,
	c *connection,
	res schema.GroupVersionResource,
	ns string,
) error {
	opts := metav1.ListOptions{}
	withlabels := a.Delete.Labels()
	labelSelString := ""
	if withlabels != nil {
		// We already validated the label selector during parse-time
		labelsStr := labels.Set(withlabels).String()
		labelSelString = fmt.Sprintf(" (labels: %s)", labelsStr)
		opts.LabelSelector = labelsStr
	}
	resName := res.Resource
	debug.Println(
		ctx, "kube.delete: %s%s (ns: %s)",
		resName, labelSelString, ns,
	)
	return c.client.Resource(res).Namespace(ns).DeleteCollection(
		ctx,
		metav1.DeleteOptions{},
		opts,
	)
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
