// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"fmt"

	"github.com/gdt-dev/core/api"
)

var (
	// ErrResourceUnknown is returned when an unknown resource kind is
	// specified for a create/apply/delete target. This is a runtime error
	// because we rely on the discovery client to determine whether a resource
	// kind is valid.
	ErrResourceUnknown = fmt.Errorf(
		"%w: resource unknown",
		api.ErrFailure,
	)
	// ErrExpectedNotFound is returned when we expected to get either a
	// NotFound response code (get) or an empty set of results (list) but did
	// not find that.
	ErrExpectedNotFound = fmt.Errorf(
		"%w: expected not found",
		api.ErrFailure,
	)
	// ErrMatchesNotEqual is returned when we failed to match a resource to an
	// object field in a `kube.assert.matches` object.
	ErrMatchesNotEqual = fmt.Errorf(
		"%w: match field not equal",
		api.ErrFailure,
	)
	// ErrConditionDoesNotMatch is returned when we failed to match a resource to an
	// Condition match expression in a `kube.assert.matches` object.
	ErrConditionDoesNotMatch = fmt.Errorf(
		"%w: condition does not match expectation",
		api.ErrFailure,
	)
	// ErrConnect is returned when we failed to create a client config to
	// connect to the Kubernetes API server.
	ErrConnect = fmt.Errorf(
		"%w: k8s connect failure",
		api.RuntimeError,
	)
)

// ResourceUnknown returns ErrRuntimeResourceUnknown for a given resource or
// kind arg string
func ResourceUnknown(arg string) error {
	return fmt.Errorf("%w: %s", ErrResourceUnknown, arg)
}

// ExpectedNotFound returns ErrExpectedNotFound for a given status code or
// number of items.
func ExpectedNotFound(msg string) error {
	return fmt.Errorf("%w: %s", ErrExpectedNotFound, msg)
}

// MatchesNotEqual returns ErrMatchesNotEqual when a `kube.assert.matches`
// object did not match the returned resource.
func MatchesNotEqual(msg string) error {
	return fmt.Errorf("%w: %s", ErrMatchesNotEqual, msg)
}

// ConditionDoesNotMatch returns ErrConditionDoesNotMatch when a
// `kube.assert.conditions` object did not match the returned resource.
func ConditionDoesNotMatch(msg string) error {
	return fmt.Errorf("%w: %s", ErrConditionDoesNotMatch, msg)
}

// ConnectError returns ErrConnnect when an error is found trying to construct
// a Kubernetes client connection.
func ConnectError(err error) error {
	return fmt.Errorf("%w: %s", ErrConnect, err)
}
