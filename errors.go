// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"fmt"

	"github.com/gdt-dev/gdt/api"
	"gopkg.in/yaml.v3"
)

var (
	// ErrExpectedMapOrYAMLString is returned when a field that can contain a
	// map[string]interface{} or an embedded YAML string did not contain either
	// of those things.
	// TODO(jaypipes): Move to gdt core?
	ErrExpectedMapOrYAMLString = fmt.Errorf(
		"%w: expected either map[string]interface{} "+
			"or a string with embedded YAML",
		api.ErrParse,
	)
	// ErrEitherShortcutOrKubeSpec is returned when the test author
	// included both a shortcut (e.g. `kube.create` or `kube.apply`) AND the
	// long-form `kube` object in the same test spec.
	ErrEitherShortcutOrKubeSpec = fmt.Errorf(
		"%w: either specify a full KubeSpec in the `kube` field or specify "+
			"one of the shortcuts (e.g. `kube.create` or `kube.apply`",
		api.ErrParse,
	)
	// ErrMoreThanOneKubeAction is returned when the test author
	// included more than one Kubernetes action (e.g. `create` or `apply`) in
	// the same KubeSpec.
	ErrMoreThanOneKubeAction = fmt.Errorf(
		"%w: you may only specify a single Kubernetes action field "+
			"(e.g. `create`, `apply` or `delete`) in the `kube` object. ",
		api.ErrParse,
	)
	// ErrKubeConfigNotFound is returned when a kubeconfig path points
	// to a file that does not exist.
	ErrKubeConfigNotFound = fmt.Errorf(
		"%w: specified kube config path not found",
		api.ErrParse,
	)
	// ErrResourceSpecifier is returned when the test author uses a
	// resource specifier for the `kube.get` or `kube.delete` fields that is
	// not valid.
	ErrResourceSpecifierInvalid = fmt.Errorf(
		"%w: invalid resource specifier",
		api.ErrParse,
	)
	// ErrResourceSpecifierOrFilepath is returned when the test author
	// uses a resource specifier for the `kube.delete` fields that is not valid
	// or is not a filepath.
	ErrResourceSpecifierInvalidOrFilepath = fmt.Errorf(
		"%w: invalid resource specifier or filepath",
		api.ErrParse,
	)
	// ErrMatchesInvalid is returned when the `Kube.Assert.Matches` value is
	// malformed.
	ErrMatchesInvalid = fmt.Errorf(
		"%w: `kube.assert.matches` not well-formed",
		api.ErrParse,
	)
	// ErrConditionMatchInvalid is returned when the `Kube.Assert.Conditions`
	// value is malformed.
	ErrConditionMatchInvalid = fmt.Errorf(
		"%w: `kube.assert.conditions` not well-formed",
		api.ErrParse,
	)
	// ErrWithLabelsOnlyGetDelete is returned when the test author included
	// `kube.with.labels` but did not specify either `kube.get` or
	// `kube.delete`.
	ErrWithLabelsInvalid = fmt.Errorf(
		"%w: with labels invalid",
		api.ErrParse,
	)
	// ErrWithLabelsOnlyGetDelete is returned when the test author included
	// `kube.with.labels` but did not specify either `kube.get` or
	// `kube.delete`.
	ErrWithLabelsOnlyGetDelete = fmt.Errorf(
		"%w: with labels may only be specified for "+
			"`kube.get` or `kube.delete`",
		api.ErrParse,
	)
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

// EitherShortcutOrKubeSpecAt returns ErrEitherShortcutOrKubeSpec for a given
// YAML node
func EitherShortcutOrKubeSpecAt(node *yaml.Node) error {
	return fmt.Errorf(
		"%w at line %d, column %d",
		ErrEitherShortcutOrKubeSpec, node.Line, node.Column,
	)
}

// MoreThanOneKubeActionAt returns ErrMoreThanOneKubeAction for a given YAML
// node
func MoreThanOneKubeActionAt(node *yaml.Node) error {
	return fmt.Errorf(
		"%w at line %d, column %d",
		ErrMoreThanOneKubeAction, node.Line, node.Column,
	)
}

// ExpectedMapOrYAMLStringAt returns ErrExpectedMapOrYAMLString for a given
// YAML node
func ExpectedMapOrYAMLStringAt(node *yaml.Node) error {
	return fmt.Errorf(
		"%w at line %d, column %d",
		ErrExpectedMapOrYAMLString, node.Line, node.Column,
	)
}

// KubeConfigNotFound returns ErrKubeConfigNotFound for a given filepath
func KubeConfigNotFound(path string) error {
	return fmt.Errorf("%w: %s", ErrKubeConfigNotFound, path)
}

// InvalidResourceSpecifier returns ErrResourceSpecifier for a given
// supplied resource specifier.
func InvalidResourceSpecifier(subject string, node *yaml.Node) error {
	return fmt.Errorf(
		"%w: %s at line %d, column %d",
		ErrResourceSpecifierInvalid, subject, node.Line, node.Column,
	)
}

// InvalidResourceSpecifierOrFilepath returns
// ErrResourceSpecifierOrFilepath for a given supplied subject.
func InvalidResourceSpecifierOrFilepath(
	subject string, node *yaml.Node,
) error {
	return fmt.Errorf(
		"%w: %s at line %d, column %d",
		ErrResourceSpecifierInvalidOrFilepath, subject, node.Line, node.Column,
	)
}

// InvalidWithLabels returns ErrWithLabels with an error containing more
// context.
func InvalidWithLabels(err error, node *yaml.Node) error {
	return fmt.Errorf(
		"%w: %s at line %d, column %d",
		ErrWithLabelsInvalid, err, node.Line, node.Column,
	)
}

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

// MatchesInvalid returns ErrMatchesInvalid when a `kube.assert.matches` field
// is not well-formed.
func MatchesInvalid(matches interface{}) error {
	return fmt.Errorf(
		"%w: need string or map[string]interface{} but got %T",
		ErrMatchesInvalid, matches,
	)
}

// ConditionMatchInvalid returns ErrConditionMatchInvalid when a
// `kube.assert.conditions` field contains invalid YAML content.
func ConditionMatchInvalid(node *yaml.Node, err error) error {
	return fmt.Errorf(
		"%w at line %d, column %d: %s",
		ErrConditionMatchInvalid, node.Line, node.Column, err,
	)
}

// MatchesInvalidUnmarshalError returns ErrMatchesInvalid when a
// `kube.assert.matches` field contains invalid YAML content.
func MatchesInvalidUnmarshalError(err error) error {
	return fmt.Errorf("%w: %s", ErrMatchesInvalid, err)
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
