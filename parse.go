// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gdt-dev/core/api"
	gdtjson "github.com/gdt-dev/core/assertion/json"
	"github.com/gdt-dev/core/parse"
	"github.com/samber/lo"
	"github.com/theory/jsonpath"
	"gopkg.in/yaml.v3"
	kubelabels "k8s.io/apimachinery/pkg/labels"
	kubeselection "k8s.io/apimachinery/pkg/selection"
)

// EitherShortcutOrKubeSpecAt returns a parse error indicating the test author
// included both a shortcut (e.g. `kube.create` or `kube.apply`) AND the
// long-form `kube` object in the same test spec.
func EitherShortcutOrKubeSpecAt(node *yaml.Node) error {
	return &parse.Error{
		Line:   node.Line,
		Column: node.Column,
		Message: "either specify a full KubeSpec in the `kube` field or " +
			"specify one of the shortcuts (e.g. `kube.create` or `kube.apply`",
	}
}

// MoreThanOneKubeActionAt returns a parse error indicating the test author
// included more than one Kubernetes action (e.g. `create` or `apply`) in the
// same KubeSpec.
func MoreThanOneKubeActionAt(node *yaml.Node) error {
	return &parse.Error{
		Line:   node.Line,
		Column: node.Column,
		Message: "you may only specify a single Kubernetes action field " +
			"(e.g. `create`, `apply` or `delete`) in the `kube` object. ",
	}
}

// KubeConfigNotFoundAt returns a parse error indicating a kubeconfig path points
// to a file that does not exist.
func KubeConfigNotFoundAt(path string, node *yaml.Node) error {
	return &parse.Error{
		Line:   node.Line,
		Column: node.Column,
		Message: fmt.Sprintf(
			"specified kube config path %q not found",
			path,
		),
	}
}

// InvalidResourceSpecifierAt returns a parse error indicating the test author
// uses a resource specifier for the `kube.get` or `kube.delete` fields that is
// not valid.
func InvalidResourceSpecifierAt(subject string, node *yaml.Node) error {
	return &parse.Error{
		Line:    node.Line,
		Column:  node.Column,
		Message: fmt.Sprintf("invalid resource specifier: %q", subject),
	}
}

// InvalidResourceSpecifierOrFilepathAt returns a parse error indicating the test
// author uses a resource specifier for the `kube.delete` fields that is not
// valid or is not a filepath.
func InvalidResourceSpecifierOrFilepathAt(
	subject string, node *yaml.Node,
) error {
	return &parse.Error{
		Line:   node.Line,
		Column: node.Column,
		Message: fmt.Sprintf(
			"invalid resource specifier or filepath: %q", subject,
		),
	}
}

// InvalidMatchesAt returns a parse error indicating the `Kube.Assert.Matches`
// value is malformed.
func InvalidMatchesAt(node *yaml.Node) error {
	return &parse.Error{
		Line:    node.Line,
		Column:  node.Column,
		Message: "`kube.assert.matches` not well-formed",
	}
}

// InvalidMatchesUnmarshalErrorAt returns a parse error indicating the
// `Kube.Assert.Matches` value is malformed and we could not unmarshal it.
func InvalidMatchesUnmarshalErrorAt(err error, node *yaml.Node) error {
	return &parse.Error{
		Line:   node.Line,
		Column: node.Column,
		Message: fmt.Sprintf(
			"`kube.assert.matches` not well-formed. unmarshal error: %s",
			err,
		),
	}
}

// InvalidConditionMatchAtreturn a parse error indicating the
// `Kube.Assert.Conditions` value is malformed.
func InvalidConditionMatchAt(err error, node *yaml.Node) error {
	return &parse.Error{
		Line:   node.Line,
		Column: node.Column,
		Message: fmt.Sprintf(
			"`kube.assert.conditions` not well-formed: %s", err,
		),
	}
}

// InvalidWithLabelsAt returns a parse error indicating the test author included
// `kube.with.labels` but did not specify either `kube.get` or `kube.delete`.
func InvalidWithLabelsAt(err error, node *yaml.Node) error {
	return &parse.Error{
		Line:    node.Line,
		Column:  node.Column,
		Message: fmt.Sprintf("with labels invalid: %s", err),
	}
}

// WithLabelsOnlyGetDeleteAt returns a parse error indicating the test author
// included `kube.with.labels` but did not specify either `kube.get` or
// `kube.delete`.
func WithLabelsOnlyGetDeleteAt(node *yaml.Node) error {
	return &parse.Error{
		Line:   node.Line,
		Column: node.Column,
		Message: "with labels may only be specified for " +
			"`kube.get` or `kube.delete`",
	}
}

func (s *Spec) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return parse.ExpectedMapAt(node)
	}
	vars := Variables{}
	// We do an initial pass over the shortcut fields, then all the
	// non-shortcut fields after that.
	var ks *KubeSpec

	// maps/structs are stored in a top-level Node.Content field which is a
	// concatenated slice of Node pointers in pairs of key/values.
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode.Kind != yaml.ScalarNode {
			return parse.ExpectedScalarAt(keyNode)
		}
		key := keyNode.Value
		valNode := node.Content[i+1]
		switch key {
		case "kube.get":
			if valNode.Kind != yaml.ScalarNode && valNode.Kind != yaml.MappingNode {
				return parse.ExpectedScalarAt(valNode)
			}
			if ks != nil {
				return MoreThanOneKubeActionAt(valNode)
			}
			var v *ResourceIdentifier
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			ks = &KubeSpec{}
			ks.Get = v
			s.Kube = ks
		case "kube.create":
			if valNode.Kind != yaml.ScalarNode {
				return parse.ExpectedScalarAt(valNode)
			}
			if ks != nil {
				return MoreThanOneKubeActionAt(valNode)
			}
			v := valNode.Value
			if probablyFilePath(v) {
				if !fileExists(v) {
					return parse.FileNotFoundAt(v, valNode)
				}
			}
			ks = &KubeSpec{}
			ks.Create = v
			s.Kube = ks
		case "kube.apply":
			if valNode.Kind != yaml.ScalarNode {
				return parse.ExpectedScalarAt(valNode)
			}
			if ks != nil {
				return MoreThanOneKubeActionAt(valNode)
			}
			v := valNode.Value
			ks = &KubeSpec{}
			ks.Apply = v
			s.Kube = ks
		case "kube.delete":
			if valNode.Kind != yaml.ScalarNode && valNode.Kind != yaml.MappingNode {
				return parse.ExpectedScalarAt(valNode)
			}
			if ks != nil {
				return MoreThanOneKubeActionAt(valNode)
			}
			var v *ResourceIdentifierOrFile
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			ks = &KubeSpec{}
			ks.Delete = v
			s.Kube = ks
		}
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode.Kind != yaml.ScalarNode {
			return parse.ExpectedScalarAt(keyNode)
		}
		key := keyNode.Value
		valNode := node.Content[i+1]
		switch key {
		case "kube":
			if valNode.Kind != yaml.MappingNode {
				return parse.ExpectedMapAt(valNode)
			}
			if ks != nil {
				return EitherShortcutOrKubeSpecAt(valNode)
			}
			if err := valNode.Decode(&ks); err != nil {
				return err
			}
			s.Kube = ks
		case "var":
			if valNode.Kind != yaml.MappingNode {
				return parse.ExpectedMapAt(valNode)
			}
			var specVars Variables
			if err := valNode.Decode(&specVars); err != nil {
				return err
			}
			vars = lo.Assign(specVars, vars)
		case "assert":
			if valNode.Kind != yaml.MappingNode {
				return parse.ExpectedMapAt(valNode)
			}
			var e *Expect
			if err := valNode.Decode(&e); err != nil {
				return err
			}
			s.Assert = e
		case "require":
			if valNode.Kind != yaml.MappingNode {
				return parse.ExpectedMapAt(valNode)
			}
			var e *Expect
			if err := valNode.Decode(&e); err != nil {
				return err
			}
			e.Require = true
			s.Assert = e
		case "kube.get", "kube.create", "kube.delete", "kube.apply":
			continue
		default:
			if lo.Contains(api.BaseSpecFields, key) {
				continue
			}
			return parse.UnknownFieldAt(key, keyNode)
		}
	}
	if len(vars) > 0 {
		s.Var = vars
	}
	return nil
}

func (s *KubeSpec) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return parse.ExpectedMapAt(node)
	}
	// maps/structs are stored in a top-level Node.Content field which is a
	// concatenated slice of Node pointers in pairs of key/values.
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode.Kind != yaml.ScalarNode {
			return parse.ExpectedScalarAt(keyNode)
		}
		key := keyNode.Value
		valNode := node.Content[i+1]
		switch key {
		case "config":
			if valNode.Kind != yaml.ScalarNode {
				return parse.ExpectedScalarAt(valNode)
			}
			fp := valNode.Value
			if !fileExists(fp) {
				return parse.FileNotFoundAt(fp, valNode)
			}
			s.Config = fp
		case "context":
			if valNode.Kind != yaml.ScalarNode {
				return parse.ExpectedScalarAt(valNode)
			}
			// NOTE(jaypipes): We can't validate the kubectx exists yet because
			// fixtures may advertise a kube config and we look up the context
			// in s.Config() method
			s.Context = valNode.Value
		case "namespace":
			if valNode.Kind != yaml.ScalarNode {
				return parse.ExpectedScalarAt(valNode)
			}
			s.Namespace = valNode.Value
		case "get", "create", "apply", "delete":
			// Because Action is an embedded struct and we parse it below, just
			// ignore these fields in the top-level `kube:` field for now.
		default:
			return parse.UnknownFieldAt(key, keyNode)
		}
	}
	var a Action
	if err := node.Decode(&a); err != nil {
		return err
	}
	s.Action = a
	return nil
}

func (a *Action) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return parse.ExpectedMapAt(node)
	}
	// maps/structs are stored in a top-level Node.Content field which is a
	// concatenated slice of Node pointers in pairs of key/values.
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode.Kind != yaml.ScalarNode {
			return parse.ExpectedScalarAt(keyNode)
		}
		key := keyNode.Value
		valNode := node.Content[i+1]
		switch key {
		case "apply":
			if valNode.Kind != yaml.ScalarNode {
				return parse.ExpectedScalarAt(valNode)
			}
			v := valNode.Value
			if probablyFilePath(v) {
				if !fileExists(v) {
					return parse.FileNotFoundAt(v, valNode)
				}
			}
			a.Apply = v
		case "create":
			if valNode.Kind != yaml.ScalarNode {
				return parse.ExpectedScalarAt(valNode)
			}
			v := valNode.Value
			if probablyFilePath(v) {
				if !fileExists(v) {
					return parse.FileNotFoundAt(v, valNode)
				}
			}
			a.Create = v
		case "get":
			if valNode.Kind != yaml.ScalarNode && valNode.Kind != yaml.MappingNode {
				return parse.ExpectedScalarOrMapAt(valNode)
			}
			var v *ResourceIdentifier
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			a.Get = v
		case "delete":
			if valNode.Kind != yaml.ScalarNode && valNode.Kind != yaml.MappingNode {
				return parse.ExpectedScalarOrMapAt(valNode)
			}
			var v *ResourceIdentifierOrFile
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			a.Delete = v
		}
	}
	if moreThanOneAction(a) {
		return MoreThanOneKubeActionAt(node)
	}
	return nil
}

func (e *Expect) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return parse.ExpectedMapAt(node)
	}
	// maps/structs are stored in a top-level Node.Content field which is a
	// concatenated slice of Node pointers in pairs of key/values.
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode.Kind != yaml.ScalarNode {
			return parse.ExpectedScalarAt(keyNode)
		}
		key := keyNode.Value
		valNode := node.Content[i+1]
		switch key {
		case "require", "stop-on-fail", "stop_on_fail", "stop.on.fail",
			"fail-stop", "fail.stop", "fail_stop":
			if valNode.Kind != yaml.ScalarNode {
				return parse.ExpectedScalarAt(valNode)
			}
			req, err := strconv.ParseBool(valNode.Value)
			if err != nil {
				return parse.ExpectedBoolAt(valNode)
			}
			e.Require = req
		case "error":
			if valNode.Kind != yaml.ScalarNode {
				return parse.ExpectedScalarAt(valNode)
			}
			var v string
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			e.Error = v
		case "len":
			if valNode.Kind != yaml.ScalarNode {
				return parse.ExpectedScalarAt(valNode)
			}
			var v *int
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			e.Len = v
		case "unknown":
			if valNode.Kind != yaml.ScalarNode {
				return parse.ExpectedScalarAt(valNode)
			}
			var v bool
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			e.Unknown = v
		case "notfound":
			if valNode.Kind != yaml.ScalarNode {
				return parse.ExpectedScalarAt(valNode)
			}
			var v bool
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			e.NotFound = v
		case "json":
			if valNode.Kind != yaml.MappingNode {
				return parse.ExpectedMapAt(valNode)
			}
			var v *gdtjson.Expect
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			e.JSON = v
		case "conditions":
			if valNode.Kind != yaml.MappingNode {
				return parse.ExpectedMapAt(valNode)
			}
			var v map[string]*ConditionMatch
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			e.Conditions = v
		case "matches":
			if valNode.Kind == yaml.MappingNode {
				var v map[string]interface{}
				if err := valNode.Decode(&v); err != nil {
					return err
				}
				e.Matches = v
			} else if valNode.Kind == yaml.ScalarNode {
				if valNode.Tag == "!!null" {
					return parse.ExpectedMapOrYAMLStringAt(valNode)
				}
				var v string
				if err := valNode.Decode(&v); err != nil {
					return err
				}
				if probablyFilePath(v) {
					if !fileExists(v) {
						return parse.FileNotFoundAt(v, valNode)
					}
				}
				// inline YAML. check it can be unmarshaled into a
				// map[string]interface{}
				var m map[string]interface{}
				if err := yaml.Unmarshal([]byte(v), &m); err != nil {
					return InvalidMatchesUnmarshalErrorAt(err, valNode)
				}
				e.Matches = m
			} else {
				return parse.ExpectedMapOrYAMLStringAt(valNode)
			}
		case "placement":
			if valNode.Kind != yaml.MappingNode {
				return parse.ExpectedMapAt(valNode)
			}
			var v *PlacementAssertion
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			e.Placement = v
		default:
			return parse.UnknownFieldAt(key, keyNode)
		}
	}
	return nil
}

// UnmarshalYAML is a custom unmarshaler that understands that the value of the
// ConditionMatch can be either a string, a slice of strings, or an object with .
func (m *ConditionMatch) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode || node.Kind == yaml.SequenceNode {
		var fs api.FlexStrings
		if err := node.Decode(&fs); err != nil {
			return InvalidConditionMatchAt(err, node)
		}
		m.conditionMatch = conditionMatch{Status: &fs}
	}
	if node.Kind == yaml.MappingNode {
		var cm conditionMatch
		if err := node.Decode(&cm); err != nil {
			return InvalidConditionMatchAt(err, node)
		}
		m.conditionMatch = cm
		return nil
	}
	return nil
}

// UnmarshalYAML is a custom unmarshaler that understands that the value of the
// ResourceIdentifier can be either a string or a selector.
func (r *ResourceIdentifier) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode && node.Kind != yaml.MappingNode {
		return parse.ExpectedScalarOrMapAt(node)
	}
	var s string
	// A resource identifier can be a string of the form {type}/{name} or
	// {type}.
	if err := node.Decode(&s); err == nil {
		if strings.ContainsAny(s, " ,;\n\t\r") {
			return InvalidResourceSpecifierAt(s, node)
		}
		if strings.Count(s, "/") > 1 {
			return InvalidResourceSpecifierAt(s, node)
		}
		r.Arg, r.Name = splitArgName(s)
		return nil
	}
	// Otherwise the resource identifier should be specified broken out as a
	// struct with a `type` and `labels` field.
	var ri resourceIdentifierWithSelector
	if err := node.Decode(&ri); err != nil {
		return err
	}
	r.Arg = ri.Type
	r.Name = ri.Name
	r.LabelSelector = ri.LabelSelector
	return nil
}

// UnmarshalYAML is a custom unmarshaler that understands that the value of the
// ResourceIdentifierOrFile can be either a string or a selector.
func (r *ResourceIdentifierOrFile) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode && node.Kind != yaml.MappingNode {
		return parse.ExpectedScalarOrMapAt(node)
	}
	var s string
	// A resource identifier can be a filepath, a string of the form
	// {type}/{name} or {type}.
	if err := node.Decode(&s); err == nil {
		if probablyFilePath(s) {
			if !fileExists(s) {
				return parse.FileNotFoundAt(s, node)
			}
			r.fp = s
			return nil
		}
		if strings.ContainsAny(s, " ,;\n\t\r") {
			return InvalidResourceSpecifierOrFilepathAt(s, node)
		}
		if strings.Count(s, "/") > 1 {
			return InvalidResourceSpecifierOrFilepathAt(s, node)
		}
		r.Arg, r.Name = splitArgName(s)
		return nil
	}
	// Otherwise the resource identifier should be specified broken out as a
	// struct with a `type` and `labels` field.
	var ri resourceIdentifierWithSelector
	if err := node.Decode(&ri); err != nil {
		return err
	}
	r.Arg = ri.Type
	r.Name = ri.Name
	r.LabelSelector = ri.LabelSelector
	return nil
}

func (r *resourceIdentifierWithSelector) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return parse.ExpectedMapAt(node)
	}
	sel := kubelabels.Everything()
	// maps/structs are stored in a top-level Node.Content field which is a
	// concatenated slice of Node pointers in pairs of key/values.
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode.Kind != yaml.ScalarNode {
			return parse.ExpectedScalarAt(keyNode)
		}
		key := keyNode.Value
		valNode := node.Content[i+1]
		switch key {
		case "type":
			if valNode.Kind != yaml.ScalarNode {
				return parse.ExpectedScalarAt(valNode)
			}
			r.Type = valNode.Value
		case "name":
			if valNode.Kind != yaml.ScalarNode {
				return parse.ExpectedScalarAt(valNode)
			}
			r.Name = valNode.Value
		case "labels", "labels-all", "labels_all":
			if valNode.Kind != yaml.ScalarNode && valNode.Kind != yaml.MappingNode {
				return parse.ExpectedScalarOrMapAt(valNode)
			}
			if valNode.Kind == yaml.ScalarNode {
				s, err := kubelabels.Parse(valNode.Value)
				if err != nil {
					return InvalidWithLabelsAt(err, valNode)
				}
				// NOTE(jaypipes): If the `labels` key was found and the value
				// was a string that successfully parsed according to the
				// kubectl labels selector format, ignore any other
				// labels-in/labels-not-in keys.
				r.LabelSelector = s.DeepCopySelector()
				return nil
			} else {
				var m map[string]string
				if err := valNode.Decode(&m); err != nil {
					return err
				}
				s, err := kubelabels.ValidatedSelectorFromSet(m)
				if err != nil {
					return InvalidWithLabelsAt(err, valNode)
				}
				newReqs, _ := s.Requirements()
				for _, req := range newReqs {
					sel = sel.Add(req)
				}
			}
		case "labels-in", "labels_in", "labels-any", "labels_any":
			if valNode.Kind != yaml.MappingNode {
				return parse.ExpectedMapAt(valNode)
			}
			var m map[string][]string
			if err := valNode.Decode(&m); err != nil {
				return err
			}
			for k, vals := range m {
				req, err := kubelabels.NewRequirement(
					k, kubeselection.In, vals,
				)
				if err != nil {
					return InvalidWithLabelsAt(err, valNode)
				}
				sel = sel.Add(*req)
			}
		case "labels-not-in", "labels_not_in", "labels-not-any", "labels_not_any":
			if valNode.Kind != yaml.MappingNode {
				return parse.ExpectedMapAt(valNode)
			}
			var m map[string][]string
			if err := valNode.Decode(&m); err != nil {
				return err
			}
			for k, vals := range m {
				req, err := kubelabels.NewRequirement(
					k, kubeselection.NotIn, vals,
				)
				if err != nil {
					return InvalidWithLabelsAt(err, valNode)
				}
				sel = sel.Add(*req)
			}
		default:
			return parse.UnknownFieldAt(key, keyNode)
		}
	}
	if !sel.Empty() {
		r.LabelSelector = sel.DeepCopySelector()
	}
	return nil
}

// UnmarshalYAML is a custom unmarshaler that ensures that JSONPath expressions
// contained in the VarEntry are valid.
func (e *VarEntry) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return parse.ExpectedMapAt(node)
	}
	// maps/structs are stored in a top-level Node.Content field which is a
	// concatenated slice of Node pointers in pairs of key/values.
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode.Kind != yaml.ScalarNode {
			return parse.ExpectedScalarAt(keyNode)
		}
		key := keyNode.Value
		valNode := node.Content[i+1]
		switch key {
		case "from":
			if valNode.Kind != yaml.ScalarNode {
				return parse.ExpectedScalarAt(valNode)
			}
			var path string
			if err := valNode.Decode(&path); err != nil {
				return err
			}
			if len(path) == 0 || path[0] != '$' {
				return gdtjson.JSONPathInvalidNoRoot(path, valNode)
			}
			if _, err := jsonpath.Parse(path); err != nil {
				return gdtjson.JSONPathInvalid(path, err, valNode)
			}
			e.From = path
		}
	}
	return nil
}

// moreThanOneAction returns true if the test author has specified more than a
// single action in the KubeSpec.
func moreThanOneAction(a *Action) bool {
	foundActions := 0
	if a.Get != nil {
		foundActions += 1
	}
	if a.Create != "" {
		foundActions += 1
	}
	if a.Apply != "" {
		foundActions += 1
	}
	if a.Delete != nil {
		foundActions += 1
	}
	return foundActions > 1
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// splitArgName returns the resource or kind arg string for a supplied `Get` or
// `Delete` command where the user can specify either a resource kind or alias,
// e.g. "pods" or "po", or the resource kind followed by a forward slash and a
// resource name.
//
// Valid resource/kind arg plus name strings:
//
// * "pods"
// * "pod"
// * "pods/name"
// * "pod/name"
// * "deployments.apps/name"
// * "deployments.v1.apps/name"
// * "Deployment/name"
// * "Deployment.apps/name"
// * "Deployment.v1.apps/name"
func splitArgName(subject string) (string, string) {
	arg, name, _ := strings.Cut(subject, "/")
	return arg, name
}
