// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"context"

	"github.com/gdt-dev/gdt/api"
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
		return api.NewResult(), nil
	}
	return api.NewResult(api.WithFailures(a.Failures()...)), nil
}
