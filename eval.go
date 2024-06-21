// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"context"

	gdterrors "github.com/gdt-dev/gdt/errors"
	"github.com/gdt-dev/gdt/result"
)

// Eval performs an action and evaluates the results of that action, returning
// a Result that informs the Scenario about what failed or succeeded. A new
// Kubernetes client request is made during this call.
func (s *Spec) Eval(ctx context.Context) (*result.Result, error) {
	c, err := s.connect(ctx)
	if err != nil {
		return nil, ConnectError(err)
	}

	ns := s.Namespace()

	var out interface{}
	err = s.Kube.Do(ctx, c, ns, &out)
	if err != nil {
		if err == gdterrors.ErrTimeoutExceeded {
			return result.New(result.WithFailures(gdterrors.ErrTimeoutExceeded)), nil
		}
		if err == gdterrors.RuntimeError {
			return nil, err
		}
	}
	a := newAssertions(c, s.Assert, err, out)
	if a.OK(ctx) {
		return result.New(), nil
	}
	return result.New(result.WithFailures(a.Failures()...)), nil
}
