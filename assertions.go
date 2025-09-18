// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gdt-dev/core/api"
	gdtjson "github.com/gdt-dev/core/assertion/json"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Expect contains one or more assertions about a kube client call
type Expect struct {
	// Error is a string that is expected to be returned as an error string
	// from the client call
	// TODO(jaypipes): Make this polymorphic to be either a shortcut string
	// (like this) or a struct containing individual error assertion fields.
	Error string `yaml:"error,omitempty"`
	// Len is an integer that is expected to represent the number of items in
	// the response when the Get request was translated into a List operation
	// (i.e. when the resource specified was a plural kind
	Len *int `yaml:"len,omitempty"`
	// NotFound is a bool indicating the result of a call should be a
	// NotFound error. Alternately, the user can set `assert.len = 0` and for
	// single-object-returning calls (e.g. `get` or `delete`) the assertion is
	// equivalent to `assert.notfound = true`
	NotFound bool `yaml:"notfound,omitempty"`
	// Unknown is a bool indicating the test author expects that they will have
	// gotten an error ("the server could not find the requested resource")
	// from the Kubernetes API server. This is mostly good for unit/fuzz
	// testing CRDs.
	Unknown bool `yaml:"unknown,omitempty"`
	// Matches is either a string or a map[string]any containing the
	// resource that the `Kube.Get` should match against. If Matches is a
	// string, the string can be either a file path to a YAML manifest or
	// inline an YAML string containing the resource fields to compare.
	//
	// Only fields present in the Matches resource are compared. There is a
	// check for existence in the retrieved resource as well as a check that
	// the value of the fields match. Only scalar fields are matched entirely.
	// In other words, you do not need to specify every field of a struct field
	// in order to compare the value of a single field in the nested struct.
	//
	// As an example, imagine you wanted to check that a Deployment resource's
	// `Status.ReadyReplicas` field was 2. You do not need to specify all other
	// `Deployment.Status` fields like `Status.Replicas` in order to match the
	// `Status.ReadyReplicas` field value. You only need to include the
	// `Status.ReadyReplicas` field in the `Matches` value as these examples
	// demonstrate:
	//
	// ```yaml
	// tests:
	//  - name: check deployment's ready replicas is 2
	//    kube:
	//      get: deployments/my-deployment
	//      assert:
	//        matches: |
	//          kind: Deployment
	//          metadata:
	//            name: my-deployment
	//          status:
	//            readyReplicas: 2
	// ```
	//
	// you don't even need to include the kind and metadata in `Matches`. If
	// missing, no kind and name matching will be performed.
	//
	// ```yaml
	// tests:
	//  - name: check deployment's ready replicas is 2
	//    kube:
	//      get: deployments/my-deployment
	//      assert:
	//        matches: |
	//          status:
	//            readyReplicas: 2
	// ```
	//
	// In fact, you don't need to use an inline multiline YAML string. You can
	// use a `map[string]any` as well:
	//
	// ```yaml
	// tests:
	//  - name: check deployment's ready replicas is 2
	//    kube:
	//      get: deployments/my-deployment
	//      assert:
	//        matches:
	//          status:
	//            readyReplicas: 2
	// ```
	Matches any `yaml:"matches,omitempty"`
	// JSON contains the assertions about JSON data in a response from the
	// Kubernetes API server.
	JSON *gdtjson.Expect `yaml:"json,omitempty"`
	// Conditions contains the assertions to make about a resource's
	// `Status.Conditions` collection. It is a map, keyed by the ConditionType
	// (matched case-insensitively), of assertions to make about that
	// Condition. The assertions can be:
	//
	// * a string which is the ConditionStatus that should be found for that
	//   Condition
	// * a list of strings containing ConditionStatuses, any of which should be
	//   found for that Condition.
	// * an object of type `ConditionExpect` that contains more fine-grained
	//   assertions about that Condition.
	//
	// A simple example that asserts that a Pod's `Ready` Condition has a
	// status of `True`. Note that both the condition type ("Ready") and the
	// status ("True") are matched case-insensitively, which means you can just
	// use lowercase strings:
	//
	// ```yaml
	// tests:
	//  - kube:
	//      get: pods/nginx
	//      assert:
	//        conditions:
	//          ready: true
	// ```
	//
	// If we wanted to assert that the `ContainersReady` Condition had a status
	// of either `False` or `Unknown`, we could write the test like this:
	//
	// ```yaml
	// tests:
	//  - kube:
	//      get: pods/nginx
	//      assert:
	//        conditions:
	//          containersReady:
	//           - false
	//           - unknown
	// ```
	//
	// Finally, if we wanted to assert that a Deployment's `Progressing`
	// Condition had a Reason field with a value "NewReplicaSetAvailable"
	// (matched case-sensitively), we could do the following:
	//
	// ```yaml
	// tests:
	//  - kube:
	//      get: deployments/nginx
	//      assert:
	//        conditions:
	//          progressing:
	//            status: true
	//            reason: NewReplicaSetAvailable
	// ```
	Conditions map[string]*ConditionMatch `yaml:"conditions,omitempty"`
	// Placement describes expected Pod scheduling spread or pack outcomes.
	Placement *PlacementAssertion `yaml:"placement,omitempty"`
}

// conditionMatch is a struct with fields that we will match a resource's
// `Condition` against.
type conditionMatch struct {
	Status *api.FlexStrings `yaml:"status,omitempty"`
	Reason string           `yaml:"reason,omitempty"`
}

// ConditionMatch can be a string (the ConditionStatus to match), a slice of
// strings (any of the ConditionStatus values to match) or an object with
// Status and Reason fields describing the Condition fields we want to match
// on.
type ConditionMatch struct {
	conditionMatch
}

// PlacementAssertion describes an expectation for Pod scheduling outcomes.
type PlacementAssertion struct {
	// Spread contains zero or more topology keys that gdt-kube will assert an
	// even spread across.
	Spread *api.FlexStrings `yaml:"spread,omitempty"`
	// Pack contains zero or more topology keys that gdt-kube will assert
	// bin-packing of resources within.
	Pack *api.FlexStrings `yaml:"pack,omitempty"`
}

// assertions contains all assertions made for the exec test
type assertions struct {
	// c is the connection to the Kubernetes API for when the assertions needs
	// to query for things like placement outcomes or Node resources.
	c *connection
	// failures contains the set of error messages for failed assertions
	failures []error
	// exp contains the expected conditions to assert against
	exp *Expect
	// err is the error returned by the client or action. This is evaluated
	// against a set of expected conditions.
	err error
	// r is either an `unstructured.Unstructured` or an
	// `unstructured.UnstructuredList` response returned from the kube client
	// call.
	r any
}

// Fail appends a supplied error to the set of failed assertions
func (a *assertions) Fail(err error) {
	a.failures = append(a.failures, err)
}

// Failures returns a slice of errors for all failed assertions
func (a *assertions) Failures() []error {
	if a == nil {
		return []error{}
	}
	return a.failures
}

// OK checks all the assertions against the supplied arguments and returns true
// if all assertions pass.
func (a *assertions) OK(ctx context.Context) bool {
	exp := a.exp
	if exp == nil {
		if a.err != nil {
			a.Fail(api.UnexpectedError(a.err))
			return false
		}
		return true
	}
	if !a.errorOK() {
		return false
	}
	if !a.lenOK() {
		return false
	}
	if !a.matchesOK(ctx) {
		return false
	}
	if !a.conditionsOK() {
		return false
	}
	if !a.jsonOK(ctx) {
		return false
	}
	if !a.placementOK(ctx) {
		return false
	}
	return true
}

// errorOK returns true if the supplied error matches the Error conditions,
// false otherwise.
func (a *assertions) errorOK() bool {
	exp := a.exp
	// We first evaluate whether an error we have received should be
	// "swallowed" because it was expected. If we still have an error after
	// swallowing all unexpected errors, then that is an unexpected error and
	// we fail.
	if a.err != nil {
		if errors.Is(a.err, ErrResourceUnknown) {
			if !exp.Unknown {
				a.Fail(a.err)
				return false
			}
			// "Swallow" the Unknown error since we expected it.
			a.err = nil
		}
		// check if the error is like one returned from Get or Delete
		// that has a 404 ErrStatus.Code in it
		apierr, ok := a.err.(*apierrors.StatusError)
		if ok {
			if a.expectsNotFound() {
				if http.StatusNotFound != int(apierr.ErrStatus.Code) {
					msg := fmt.Sprintf("got status code %d", apierr.ErrStatus.Code)
					a.Fail(ExpectedNotFound(msg))
					return false
				}
				// "Swallow" the NotFound error since we expected it.
				a.err = nil
			} else {
				a.Fail(apierr)
				return false
			}
		}
	}
	if exp.Error != "" && a.r != nil {
		if a.err == nil {
			a.Fail(api.UnexpectedError(a.err))
			return false
		}
		if !strings.Contains(a.err.Error(), exp.Error) {
			a.Fail(api.NotIn(a.err.Error(), exp.Error))
			return false
		}
	}
	if a.err != nil {
		a.Fail(api.UnexpectedError(a.err))
		return false
	}
	return true
}

func (a *assertions) expectsNotFound() bool {
	exp := a.exp
	return (exp.Len != nil && *exp.Len == 0) || exp.NotFound
}

// lenOK returns true if the subject matches the Len condition, false otherwise
func (a *assertions) lenOK() bool {
	exp := a.exp
	if exp.Len != nil && a.hasSubject() {
		// if the supplied resp is a list of objects returned by the dynamic
		// client check its length
		list, ok := a.r.(*unstructured.UnstructuredList)
		if ok && list != nil {
			if len(list.Items) != *exp.Len {
				a.Fail(api.NotEqualLength(*exp.Len, len(list.Items)))
				return false
			}
		}
	}
	return true
}

// matchesOK returns true if the subject matches the Matches condition, false
// otherwise
func (a *assertions) matchesOK(ctx context.Context) bool {
	exp := a.exp
	if exp.Matches != nil && a.hasSubject() {
		matchObj := matchObjectFromAny(ctx, exp.Matches)
		res, ok := a.r.(*unstructured.Unstructured)
		if ok {
			delta := compareResourceToMatchObject(res, matchObj)
			if !delta.Empty() {
				for _, diff := range delta.Differences() {
					a.Fail(MatchesNotEqual(diff))
				}
				return false
			}
			return true
		}

		// TODO(jaypipes): if the supplied resp is a list of objects returned
		// by the dynamic client check each against the supplied matches
		// fields.
		//list, ok := a.r.(*unstructured.UnstructuredList)
		//if ok {
		//	for _, obj := range list.Items {
		//      diff := compareResourceToMatchObject(obj, matchObj)
		//
		//		a.Fail(api.NotEqualLength(*exp.Len, len(list.Items)))
		//		return false
		//	}
		//}
	}
	return true
}

// conditionsOK returns true if the subject matches the Conditions condition,
// false otherwise
func (a *assertions) conditionsOK() bool {
	exp := a.exp
	if exp.Conditions != nil && a.hasSubject() {
		res, ok := a.r.(*unstructured.Unstructured)
		if ok {
			delta := compareConditions(res, exp.Conditions)
			if !delta.Empty() {
				for _, diff := range delta.Differences() {
					a.Fail(ConditionDoesNotMatch(diff))
				}
				return false
			}
			return true
		}

		// TODO(jaypipes): if the supplied resp is a list of objects returned
		// by the dynamic client check each against the supplied matches
		// fields.
		//list, ok := a.r.(*unstructured.UnstructuredList)
		//if ok {
		//	for _, obj := range list.Items {
		//      diff := compareResourceToMatchObject(obj, matchObj)
		//
		//		a.Fail(api.NotEqualLength(*exp.Len, len(list.Items)))
		//		return false
		//	}
		//}
	}
	return true
}

// jsonOK returns true if the subject matches the JSON conditions, false
// otherwise
func (a *assertions) jsonOK(ctx context.Context) bool {
	exp := a.exp
	if exp.JSON != nil && a.hasSubject() {
		var err error
		var b []byte
		res, ok := a.r.(*unstructured.Unstructured)
		if ok {
			if b, err = json.Marshal(res); err != nil {
				panic("unable to marshal unstructured.Unstructured")
			}
		}
		ja := gdtjson.New(exp.JSON, b)
		if !ja.OK(ctx) {
			for _, f := range ja.Failures() {
				a.Fail(f)
			}
			return false
		}
	}
	return true
}

// placementOK returns true if the subject matches the Placement conditions,
// false otherwise
func (a *assertions) placementOK(ctx context.Context) bool {
	exp := a.exp
	if exp.Placement != nil && a.hasSubject() {
		// TODO(jaypipes): Handle list returns...
		res, ok := a.r.(*unstructured.Unstructured)
		if !ok {
			panic("expected result to be unstructured.Unstructured")
		}
		spread := exp.Placement.Spread
		if spread != nil {
			ok = a.placementSpreadOK(ctx, res, spread.Values())
		}
		pack := exp.Placement.Pack
		if pack != nil {
			ok = ok && a.placementPackOK(ctx, res, pack.Values())
		}
		return ok
	}
	return true
}

// hasSubject returns true if the assertions `r` field (which contains the
// subject of which we inspect) is not `nil`.
func (a *assertions) hasSubject() bool {
	switch a.r.(type) {
	case *unstructured.Unstructured:
		v := a.r.(*unstructured.Unstructured)
		return v != nil
	case *unstructured.UnstructuredList:
		v := a.r.(*unstructured.UnstructuredList)
		return v != nil
	}
	return false
}

// newAssertions returns an assertions object populated with the supplied http
// spec assertions
func newAssertions(
	c *connection,
	exp *Expect,
	err error,
	r any,
) api.Assertions {
	return &assertions{
		c:        c,
		failures: []error{},
		exp:      exp,
		err:      err,
		r:        r,
	}
}
