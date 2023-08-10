// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"os"
	"strings"

	gdtjson "github.com/gdt-dev/gdt/assertion/json"
	"github.com/gdt-dev/gdt/errors"
	gdttypes "github.com/gdt-dev/gdt/types"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/labels"
)

func (s *Spec) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return errors.ExpectedMapAt(node)
	}
	// maps/structs are stored in a top-level Node.Content field which is a
	// concatenated slice of Node pointers in pairs of key/values.
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode.Kind != yaml.ScalarNode {
			return errors.ExpectedScalarAt(keyNode)
		}
		key := keyNode.Value
		valNode := node.Content[i+1]
		switch key {
		case "kube":
			if valNode.Kind != yaml.MappingNode {
				return errors.ExpectedMapAt(valNode)
			}
			var ks *KubeSpec
			if err := valNode.Decode(&ks); err != nil {
				return err
			}
			s.Kube = ks
		case "kube.get":
			if valNode.Kind != yaml.ScalarNode {
				return errors.ExpectedScalarAt(valNode)
			}
			v := valNode.Value
			if err := validateResourceIdentifier(v); err != nil {
				return err
			}
			s.KubeGet = v
		case "kube.create":
			if valNode.Kind != yaml.ScalarNode {
				return errors.ExpectedScalarAt(valNode)
			}
			v := valNode.Value
			if err := validateFileExists(v); err != nil {
				return err
			}
			s.KubeCreate = v
		case "kube.apply":
			if valNode.Kind != yaml.ScalarNode {
				return errors.ExpectedScalarAt(valNode)
			}
			s.KubeApply = valNode.Value
		case "kube.delete":
			if valNode.Kind != yaml.ScalarNode {
				return errors.ExpectedScalarAt(valNode)
			}
			v := valNode.Value
			if err := validateResourceIdentifierOrFilepath(v); err != nil {
				return err
			}
			if err := validateFileExists(v); err != nil {
				return err
			}
			s.KubeDelete = v
		case "assert":
			if valNode.Kind != yaml.MappingNode {
				return errors.ExpectedMapAt(valNode)
			}
			var e *Expect
			if err := valNode.Decode(&e); err != nil {
				return err
			}
			s.Assert = e
		default:
			if lo.Contains(gdttypes.BaseSpecFields, key) {
				continue
			}
			return errors.UnknownFieldAt(key, keyNode)
		}
	}
	if err := validateShortcuts(s); err != nil {
		return err
	}
	expandShortcut(s)
	if moreThanOneAction(s) {
		return ErrMoreThanOneKubeAction
	}
	with := s.Kube.With
	if with != nil {
		if s.Kube.Get == "" && s.Kube.Delete == "" {
			return ErrWithLabelsOnlyGetDelete
		}
	}
	return nil
}

func (s *KubeSpec) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return errors.ExpectedMapAt(node)
	}
	// maps/structs are stored in a top-level Node.Content field which is a
	// concatenated slice of Node pointers in pairs of key/values.
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode.Kind != yaml.ScalarNode {
			return errors.ExpectedScalarAt(keyNode)
		}
		key := keyNode.Value
		valNode := node.Content[i+1]
		switch key {
		case "config":
			if valNode.Kind != yaml.ScalarNode {
				return errors.ExpectedScalarAt(valNode)
			}
			fp := valNode.Value
			if err := validateFileExists(fp); err != nil {
				return err
			}
			s.Config = fp
		case "context":
			if valNode.Kind != yaml.ScalarNode {
				return errors.ExpectedScalarAt(valNode)
			}
			// NOTE(jaypipes): We can't validate the kubectx exists yet because
			// fixtures may advertise a kube config and we look up the context
			// in s.Config() method
			s.Context = valNode.Value
		case "namespace":
			if valNode.Kind != yaml.ScalarNode {
				return errors.ExpectedScalarAt(valNode)
			}
			s.Namespace = valNode.Value
		case "apply":
			if valNode.Kind != yaml.ScalarNode {
				return errors.ExpectedScalarAt(valNode)
			}
			v := valNode.Value
			if err := validateFileExists(v); err != nil {
				return err
			}
			s.Apply = v
		case "create":
			if valNode.Kind != yaml.ScalarNode {
				return errors.ExpectedScalarAt(valNode)
			}
			v := valNode.Value
			if err := validateFileExists(v); err != nil {
				return err
			}
			s.Create = v
		case "get":
			if valNode.Kind != yaml.ScalarNode {
				return errors.ExpectedScalarAt(valNode)
			}
			v := valNode.Value
			if err := validateResourceIdentifier(v); err != nil {
				return err
			}
			s.Get = v
		case "delete":
			if valNode.Kind != yaml.ScalarNode {
				return errors.ExpectedScalarAt(valNode)
			}
			v := valNode.Value
			if err := validateResourceIdentifierOrFilepath(v); err != nil {
				return err
			}
			if err := validateFileExists(v); err != nil {
				return err
			}
			s.Delete = v
		case "with":
			if valNode.Kind != yaml.MappingNode {
				return errors.ExpectedMapAt(valNode)
			}
			var w *With
			if err := valNode.Decode(&w); err != nil {
				return err
			}
			if w.Labels != nil {
				_, err := labels.ValidatedSelectorFromSet(w.Labels)
				if err != nil {
					return InvalidWithLabels(err, valNode)
				}
			}
			s.With = w
		default:
			return errors.UnknownFieldAt(key, keyNode)
		}
	}
	return nil
}

func (e *Expect) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return errors.ExpectedMapAt(node)
	}
	// maps/structs are stored in a top-level Node.Content field which is a
	// concatenated slice of Node pointers in pairs of key/values.
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode.Kind != yaml.ScalarNode {
			return errors.ExpectedScalarAt(keyNode)
		}
		key := keyNode.Value
		valNode := node.Content[i+1]
		switch key {
		case "error":
			if valNode.Kind != yaml.ScalarNode {
				return errors.ExpectedScalarAt(valNode)
			}
			var v string
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			e.Error = v
		case "len":
			if valNode.Kind != yaml.ScalarNode {
				return errors.ExpectedScalarAt(valNode)
			}
			var v *int
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			e.Len = v
		case "unknown":
			if valNode.Kind != yaml.ScalarNode {
				return errors.ExpectedScalarAt(valNode)
			}
			var v bool
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			e.Unknown = v
		case "notfound":
			if valNode.Kind != yaml.ScalarNode {
				return errors.ExpectedScalarAt(valNode)
			}
			var v bool
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			e.NotFound = v
		case "json":
			if valNode.Kind != yaml.MappingNode {
				return errors.ExpectedMapAt(valNode)
			}
			var v *gdtjson.Expect
			if err := valNode.Decode(&v); err != nil {
				return err
			}
			e.JSON = v
		case "conditions":
			if valNode.Kind != yaml.MappingNode {
				return errors.ExpectedMapAt(valNode)
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
				if err := validateFileExists(v); err != nil {
					return err
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
		default:
			return errors.UnknownFieldAt(key, keyNode)
		}
	}
	return nil
}

// validateShortcuts ensures that the test author has specified only a single
// shortcut (e.g. `kube.create`) and that if a shortcut is specified, any
// long-form KubeSpec is not present.
func validateShortcuts(s *Spec) error {
	foundShortcuts := 0
	if s.KubeGet != "" {
		foundShortcuts += 1
	}
	if s.KubeCreate != "" {
		foundShortcuts += 1
	}
	if s.KubeApply != "" {
		foundShortcuts += 1
	}
	if s.KubeDelete != "" {
		foundShortcuts += 1
	}
	if s.Kube == nil {
		if foundShortcuts > 1 {
			return ErrMoreThanOneShortcut
		} else if foundShortcuts == 0 {
			return ErrEitherShortcutOrKubeSpec
		}
	} else {
		if foundShortcuts > 0 {
			return ErrEitherShortcutOrKubeSpec
		}
	}
	return nil
}

// expandShortcut looks at the shortcut fields (e.g. `kube.create`) and expands
// the shortcut into a full KubeSpec.
func expandShortcut(s *Spec) {
	if s.Kube != nil {
		return
	}
	ks := &KubeSpec{}
	if s.KubeGet != "" {
		ks.Get = s.KubeGet
	}
	if s.KubeCreate != "" {
		ks.Create = s.KubeCreate
	}
	if s.KubeApply != "" {
		ks.Apply = s.KubeApply
	}
	if s.KubeDelete != "" {
		ks.Delete = s.KubeDelete
	}
	s.Kube = ks
}

// moreThanOneAction returns true if the test author has specified more than a
// single action in the KubeSpec.
func moreThanOneAction(s *Spec) bool {
	foundActions := 0
	if s.Kube.Get != "" {
		foundActions += 1
	}
	if s.Kube.Create != "" {
		foundActions += 1
	}
	if s.Kube.Apply != "" {
		foundActions += 1
	}
	if s.Kube.Delete != "" {
		foundActions += 1
	}
	return foundActions > 1
}

// validateFileExists returns an error if the supplied path looks like a file
// path but the file does not exist.
func validateFileExists(path string) error {
	if probablyFilePath(path) {
		_, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return errors.FileNotFound(path)
			}
			return err
		}
	}
	return nil
}

// validateResourceIdentifierOrFilepath returns an error if the supplied
// argument is not a filepath and contains an ill-formed Kind, Alias or
// Kind/Name specifier. Only a single Kind may be specified (i.e. no commas or
// spaces are allowed in the supplied string.)
func validateResourceIdentifierOrFilepath(subject string) error {
	if probablyFilePath(subject) {
		return nil
	}
	if strings.ContainsAny(subject, " ,;\n\t\r") {
		return InvalidResourceSpecifierOrFilepath(subject)
	}
	if strings.Count(subject, "/") > 1 {
		return InvalidResourceSpecifierOrFilepath(subject)
	}
	return nil
}

// validateResourceIdentifier returns an error if the supplied argument
// contains an ill-formed Kind, Alias or Kind/Name specifier. Only a single
// Kind may be specified (i.e. no commas or spaces are allowed in the supplied
// string.)
func validateResourceIdentifier(subject string) error {
	if strings.ContainsAny(subject, " ,;\n\t\r") {
		return InvalidResourceSpecifier(subject)
	}
	if strings.Count(subject, "/") > 1 {
		return InvalidResourceSpecifier(subject)
	}
	return nil
}
