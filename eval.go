// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"context"
	"testing"
	"time"

	"github.com/cenkalti/backoff"
	gdtcontext "github.com/gdt-dev/gdt/context"
	"github.com/gdt-dev/gdt/debug"
	gdterrors "github.com/gdt-dev/gdt/errors"
	"github.com/gdt-dev/gdt/result"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
		action := &s.Kube.Action
		if action.Get != nil {
			kind, name := action.Get.KindName()
			gvk := schema.GroupVersionKind{
				Kind: kind,
			}
			gvr, err := c.gvrFromGVK(gvk)
			a := newAssertions(s.Assert, err, nil)
			if !a.OK() {
				res = result.New(result.WithFailures(a.Failures()...))
			}
			// if the Spec has no timeout, default it to a reasonable value
			var cancel context.CancelFunc
			_, hasDeadline := ctx.Deadline()
			if !hasDeadline {
				ctx, cancel = context.WithTimeout(ctx, defaultGetTimeout)
				defer cancel()
			}

			// retry the Get/List and test the assertions until they succeed or the timeout expires.
			bo := backoff.WithContext(backoff.NewExponentialBackOff(), ctx)
			ticker := backoff.NewTicker(bo)
			attempts := 0
			start := time.Now().UTC()
			for tick := range ticker.C {
				attempts++
				after := tick.Sub(start)

				if name == "" {
					list, err := action.doList(ctx, t, c, gvr, s.Namespace())
					a = newAssertions(s.Assert, err, list)
				} else {
					obj, err := action.doGet(ctx, t, c, gvr, name, s.Namespace())
					a = newAssertions(s.Assert, err, obj)
				}
				success := a.OK()
				term := a.Terminal()
				debug.Println(
					ctx, t, "%s (try %d after %s) ok: %v, terminal: %v",
					s.Title(), attempts, after, success, term,
				)
				if success || term {
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
			res = result.New(result.WithFailures(a.Failures()...))
		}
		if s.Kube.Create != "" {
			res, err := action.doCreate(ctx, t, c)
			// TODO(jaypipes): Clearly this is applying the same assertion to each
			// object that was created, which is wrong. When I add the polymorphism
			// to the Assertions struct, I will modify this block to look for an
			// indexed set of error assertions.
			a = newAssertions(s.Assert, err, res)
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
