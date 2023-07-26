// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"os"
	"strings"

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
			s.KubeGet = valNode.Value
		case "kube.create":
			if valNode.Kind != yaml.ScalarNode {
				return errors.ExpectedScalarAt(valNode)
			}
			s.KubeCreate = valNode.Value
		case "kube.apply":
			if valNode.Kind != yaml.ScalarNode {
				return errors.ExpectedScalarAt(valNode)
			}
			s.KubeApply = valNode.Value
		case "kube.delete":
			if valNode.Kind != yaml.ScalarNode {
				return errors.ExpectedScalarAt(valNode)
			}
			s.KubeDelete = valNode.Value
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
	if err := validateKubeSpec(s); err != nil {
		return err
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

// validateKubeSpec ensures that the test author has specified only a single
// action in the KubeSpec and that various KubeSpec fields are set
// appropriately.
func validateKubeSpec(s *Spec) error {
	if moreThanOneAction(s) {
		return ErrMoreThanOneKubeAction
	}
	if s.Kube.Get != "" {
		if err := validateResourceIdentifier(s.Kube.Get); err != nil {
			return err
		}
	}
	if s.Kube.Delete != "" {
		if err := validateResourceIdentifierOrFilepath(s.Kube.Delete); err != nil {
			return err
		}
		if err := validateFileExists(s.Kube.Delete); err != nil {
			return err
		}
	}
	if s.Kube.Create != "" {
		if err := validateFileExists(s.Kube.Create); err != nil {
			return err
		}
	}
	if s.Kube.Apply != "" {
		if err := validateFileExists(s.Kube.Apply); err != nil {
			return err
		}
	}
	with := s.Kube.With
	if with != nil {
		if s.Kube.Get == "" && s.Kube.Delete == "" {
			return ErrWithLabelsOnlyGetDelete
		}
		if with.Labels != nil {
			_, err := labels.ValidatedSelectorFromSet(with.Labels)
			if err != nil {
				return InvalidWithLabels(err)
			}
		}
	}
	if s.Kube.Assert != nil {
		exp := s.Kube.Assert
		if exp.Matches != nil {
			if err := validateMatches(exp.Matches); err != nil {
				return err
			}
		}
	}
	return nil
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

// validateMatches checks what the test author placed in the `Kube.Matches`
// field to see if it contains one of:
// * file path (and checks existence of this file)
// * inline YAML (and checks that can be unmarshaled)
// * map[string]interface{}
func validateMatches(matches interface{}) error {
	switch matches.(type) {
	case string:
		v := matches.(string)
		if probablyFilePath(v) {
			return validateFileExists(v)
		}
		// inline YAML. Let's quickly check it can be unmarshaled into a
		// map[string]interface{}
		var m map[string]interface{}
		if err := yaml.Unmarshal([]byte(v), &m); err != nil {
			return MatchesInvalidUnmarshalError(err)
		}
	case map[string]interface{}:
		return nil
	default:
		return MatchesInvalid(matches)
	}
	return nil
}
