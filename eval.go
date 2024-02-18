// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"context"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gdt-dev/gdt/debug"
	gdterrors "github.com/gdt-dev/gdt/errors"
	"github.com/gdt-dev/gdt/result"
	gdttypes "github.com/gdt-dev/gdt/types"
)

const (
	// defaultGetTimeout is used as a retry max time if the spec's Timeout has
	// not been specified.
	defaultGetTimeout = time.Second * 5
)

// Eval performs an action and evaluates the results of that action, returning
// a Result that informs the Scenario about what failed or succeeded. A new
// Kubernetes client request is made during this call.
func (s *Spec) Eval(ctx context.Context, t *testing.T) *result.Result {
	c, err := s.connect(ctx)
	if err != nil {
		return result.New(
			result.WithRuntimeError(ConnectError(err)),
		)
	}

	var a gdttypes.Assertions
	ns := s.Namespace()

	// if the Spec has no timeout, default it to a reasonable value
	var cancel context.CancelFunc
	_, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		ctx, cancel = context.WithTimeout(ctx, defaultGetTimeout)
		defer cancel()
	}

	// retry the action and test the assertions until they succeed, there is a
	// terminal failure, or the timeout expires.
	bo := backoff.WithContext(backoff.NewExponentialBackOff(), ctx)
	ticker := backoff.NewTicker(bo)
	attempts := 0
	start := time.Now().UTC()
	for tick := range ticker.C {
		attempts++
		after := tick.Sub(start)

		var out interface{}
		err := s.Kube.Do(ctx, t, c, ns, &out)
		if err != nil {
			if err == gdterrors.ErrTimeoutExceeded {
				return result.New(result.WithFailures(gdterrors.ErrTimeoutExceeded))
			}
			if err == gdterrors.RuntimeError {
				return result.New(result.WithRuntimeError(err))
			}
		}
		a = newAssertions(s.Assert, err, &out)
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
