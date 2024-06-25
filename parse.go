// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"os"

	"github.com/gdt-dev/gdt/api"
	gdtjson "github.com/gdt-dev/gdt/assertion/json"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
)

func (s *Spec) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return api.ExpectedMapAt(node)
	}
	// We do an initial pass over the shortcut fields, then all the
	// non-shortcut fields after that.
	var ks *KubeSpec

	// maps/structs are stored in a top-level Node.Content field which is a
	// concatenated slice of Node pointers in pairs of key/values.
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode.Kind != yaml.ScalarNode {
			return api.ExpectedScalarAt(keyNode)
		}
		key := keyNode.Value
		valNode := node.Content[i+1]
		switch key {
		case "kube.get":
			if valNode.Kind != yaml.ScalarNode && valNode.Kind != yaml.MappingNode {
				return api.ExpectedScalarAt(valNode)
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
				return api.ExpectedScalarAt(valNode)
			}
			if ks != nil {
				return MoreThanOneKubeActionAt(valNode)
			}
			v := valNode.Value
			if probablyFilePath(v) {
				if !fileExists(v) {
					return api.FileNotFound(v, valNode)
				}
			}
			ks = &KubeSpec{}
			ks.Create = v
			s.Kube = ks
		case "kube.apply":
			if valNode.Kind != yaml.ScalarNode {
				return api.ExpectedScalarAt(valNode)
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
				return api.ExpectedScalarAt(valNode)
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
			return api.ExpectedScalarAt(keyNode)
		}
		key := keyNode.Value
		valNode := node.Content[i+1]
		switch key {
		case "kube":
			if valNode.Kind != yaml.MappingNode {
				return api.ExpectedMapAt(valNode)
			}
			if ks != nil {
				return EitherShortcutOrKubeSpecAt(valNode)
			}
			if err := valNode.Decode(&ks); err != nil {
				return err
			}
			s.Kube = ks
		case "assert":
			if valNode.Kind != yaml.MappingNode {
				return api.ExpectedMapAt(valNode)
			}
			var e *Expect
			if err := valNode.Decode(&e); err != nil {
				return err
			}
			s.Assert = e
		case "kube.get", "kube.create", "kube.delete", "kube.apply":
			continue
		default:
			if lo.Contains(api.BaseSpecFields, key) {
				continue
			}
			return api.UnknownFieldAt(key, keyNode)
		}
	}
	return nil
}

func (s *KubeSpec) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return api.ExpectedMapAt(node)
	}
	// maps/structs are stored in a top-level Node.Content field which is a
	// concatenated slice of Node pointers in pairs of key/values.
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode.Kind != yaml.ScalarNode {
			return api.ExpectedScalarAt(keyNode)
		}
		key := keyNode.Value
		valNode := node.Content[i+1]
		switch key {
		case "config":
			if valNode.Kind != yaml.ScalarNode {
				return api.ExpectedScalarAt(valNode)
			}
			fp := valNode.Value
			if !fileExists(fp) {
				return api.FileNotFound(fp, valNode)
			}
			s.Config = fp
		case "context":
			if valNode.Kind != yaml.ScalarNode {
				return api.ExpectedScalarAt(valNode)
			}
			// NOTE(jaypipes): We can't validate the kubectx exists yet because
			// fixtures may advertise a kube config and we look up the context
			// in s.Config() method
			s.Context = valNode.Value
		case "namespace":
			if valNode.Kind != yaml.ScalarNode {
				return api.ExpectedScalarAt(valNode)
			}
			s.Namespace = valNode.Value
		case "get", "create", "apply", "delete":
			// Because Action is an embedded struct and we parse it below, just
			// ignore these fields in the top-level `kube:` field for now.
		default:
			return api.UnknownFieldAt(key, keyNode)
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
		return api.ExpectedMapAt(node)
	}
	// maps/structs are stored in a top-level Node.Content field which is a
	// concatenated slice of Node pointers in pairs of key/values.
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode.Kind != yaml.ScalarNode {
			return api.ExpectedScalarAt(keyNode)
		}
		key := keyNode.Value
		valNode := node.Content[i+1]
		switch key {
		case "apply":
			if valNode.Kind != yaml.ScalarNode {
				return api.ExpectedScalarAt(valNode)
			}
			v := valNode.Value
			if probablyFilePath(v) {
				if !fileExists(v) {
					return api.FileNotFound(v, valNode)
				}
			}
			a.Apply = v
		case "create":
			if valNode.Kind != yaml.ScalarNode {
				return api.ExpectedScalarAt(valNode)
			}
			v := valNode.Value
			if probablyFilePath(v) {
				if !fileExists(v) {
					return api.FileNotFound(v, valNode)
				}
			}
			a.Create = v
		case "get":
			if valNode.Kind != yaml.ScalarNode && valNode.Kind != yaml.MappingNode {
				return api.ExpectedScalarOrMapAt(valNode)
			}
			var v *ResourceIdentifier
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			a.Get = v
		case "delete":
			if valNode.Kind != yaml.ScalarNode && valNode.Kind != yaml.MappingNode {
				return api.ExpectedScalarOrMapAt(valNode)
			}
			var v *ResourceIdentifierOrFile
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			a.Delete = v
		}
	}
	if moreThanOneAction(a) {
		return ErrMoreThanOneKubeAction
	}
	return nil
}

func (e *Expect) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return api.ExpectedMapAt(node)
	}
	// maps/structs are stored in a top-level Node.Content field which is a
	// concatenated slice of Node pointers in pairs of key/values.
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode.Kind != yaml.ScalarNode {
			return api.ExpectedScalarAt(keyNode)
		}
		key := keyNode.Value
		valNode := node.Content[i+1]
		switch key {
		case "error":
			if valNode.Kind != yaml.ScalarNode {
				return api.ExpectedScalarAt(valNode)
			}
			var v string
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			e.Error = v
		case "len":
			if valNode.Kind != yaml.ScalarNode {
				return api.ExpectedScalarAt(valNode)
			}
			var v *int
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			e.Len = v
		case "unknown":
			if valNode.Kind != yaml.ScalarNode {
				return api.ExpectedScalarAt(valNode)
			}
			var v bool
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			e.Unknown = v
		case "notfound":
			if valNode.Kind != yaml.ScalarNode {
				return api.ExpectedScalarAt(valNode)
			}
			var v bool
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			e.NotFound = v
		case "json":
			if valNode.Kind != yaml.MappingNode {
				return api.ExpectedMapAt(valNode)
			}
			var v *gdtjson.Expect
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			e.JSON = v
		case "conditions":
			if valNode.Kind != yaml.MappingNode {
				return api.ExpectedMapAt(valNode)
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
					return ExpectedMapOrYAMLStringAt(valNode)
				}
				var v string
				if err := valNode.Decode(&v); err != nil {
					return err
				}
				if probablyFilePath(v) {
					if !fileExists(v) {
						return api.FileNotFound(v, valNode)
					}
				}
				// inline YAML. check it can be unmarshaled into a
				// map[string]interface{}
				var m map[string]interface{}
				if err := yaml.Unmarshal([]byte(v), &m); err != nil {
					return MatchesInvalidUnmarshalError(err)
				}
				e.Matches = m
			} else {
				return ExpectedMapOrYAMLStringAt(valNode)
			}
		case "placement":
			if valNode.Kind != yaml.MappingNode {
				return api.ExpectedMapAt(valNode)
			}
			var v *PlacementAssertion
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			e.Placement = v
		default:
			return api.UnknownFieldAt(key, keyNode)
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
