// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"context"

	"github.com/gdt-dev/gdt/api"
	"github.com/gdt-dev/gdt/debug"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Eval performs an action and evaluates the results of that action, returning
// a Result that informs the Scenario about what failed or succeeded. A new
// Kubernetes client request is made during this call.
func (s *Spec) Eval(ctx context.Context) (*api.Result, error) {
	c, err := s.connect(ctx)
	if err != nil {
		return nil, ConnectError(err)
	}

	ns := s.Namespace()
	nsCreated, err := ensureNamespace(ctx, c, ns)
	if err != nil {
		return nil, err
	}
	if nsCreated {
		debug.Println(ctx, "auto-created namespace: %s", ns)
	}

	var out interface{}
	err = s.Kube.Do(ctx, c, ns, &out)
	if err != nil {
		if err == api.ErrTimeoutExceeded {
			return api.NewResult(api.WithFailures(api.ErrTimeoutExceeded)), nil
		}
		if err == api.RuntimeError {
			return nil, err
		}
	}
	a := newAssertions(c, s.Assert, err, out)
	if a.OK(ctx) {
		res := api.NewResult()
		if err := saveVars(ctx, s.Var, out, res); err != nil {
			return nil, err
		}
		return res, nil
	}
	return api.NewResult(api.WithFailures(a.Failures()...)), nil
}

// ensureNamespace automatically creates a supplied Kubernetes Namespace if it
// does not already exist, returning whether the namespace was created.
func ensureNamespace(
	ctx context.Context,
	c *connection,
	ns string,
) (bool, error) {
	res, err := c.gvrFromArg("namespaces")
	if err != nil {
		return false, err
	}
	nsObj, err := c.client.Resource(res).Get(
		ctx,
		ns,
		metav1.GetOptions{},
	)
	if err != nil && !kubeerrors.IsNotFound(err) {
		return false, err
	}
	if nsObj == nil {
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Namespace",
				"metadata": map[string]interface{}{
					"name": ns,
				},
			},
		}
		_, err = c.client.Resource(res).Create(
			ctx,
			obj,
			metav1.CreateOptions{},
		)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}
