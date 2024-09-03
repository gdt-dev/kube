// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"path/filepath"
	"strings"

	"github.com/gdt-dev/gdt/api"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/labels"
)

// resourceIdentifierWithSelector is the full long-form resource identifier as
// a struct
type resourceIdentifierWithSelector struct {
	// Type is the resource type to select. This should *not* be a type/name
	// combination.
	Type string `yaml:"type"`
	// Name is the optional name of the resource to get
	Name string `yaml:"name,omitempty"`
	// Labels is a map, keyed by metadata Label, of Label values to select a
	// resource by
	Labels map[string]string `yaml:"labels,omitempty"`
}

// ResourceIdentifier is a struct used to parse an interface{} that can be
// either a string or a struct containing a selector with things like a label
// key/value map.
type ResourceIdentifier struct {
	Arg    string            `yaml:"-"`
	Name   string            `yaml:"-"`
	Labels map[string]string `yaml:"-"`
}

// Title returns the resource identifier's kind and name, if present
func (r *ResourceIdentifier) Title() string {
	if r.Name == "" {
		return r.Arg
	}
	return r.Arg + "/" + r.Name
}

// UnmarshalYAML is a custom unmarshaler that understands that the value of the
// ResourceIdentifier can be either a string or a selector.
func (r *ResourceIdentifier) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode && node.Kind != yaml.MappingNode {
		return api.ExpectedScalarOrMapAt(node)
	}
	var s string
	// A resource identifier can be a string of the form {type}/{name} or
	// {type}.
	if err := node.Decode(&s); err == nil {
		if strings.ContainsAny(s, " ,;\n\t\r") {
			return InvalidResourceSpecifier(s, node)
		}
		if strings.Count(s, "/") > 1 {
			return InvalidResourceSpecifier(s, node)
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
	_, err := labels.ValidatedSelectorFromSet(ri.Labels)
	if err != nil {
		return InvalidWithLabels(err, node)
	}
	r.Arg = ri.Type
	r.Name = ri.Name
	r.Labels = ri.Labels
	return nil
}

func NewResourceIdentifier(
	arg string,
	name string,
	labels map[string]string,
) *ResourceIdentifier {
	return &ResourceIdentifier{
		Arg:    arg,
		Name:   name,
		Labels: labels,
	}
}

// ResourceIdentifierOrFile is a struct used to parse an interface{} that can
// be either a string, a filepath or a struct containing a selector with things
// like a label key/value map.
type ResourceIdentifierOrFile struct {
	fp     string            `yaml:"-"`
	Arg    string            `yaml:"-"`
	Name   string            `yaml:"-"`
	Labels map[string]string `yaml:"-"`
}

// FilePath returns the resource identifier's file path, if present
func (r *ResourceIdentifierOrFile) FilePath() string {
	return r.fp
}

// Title returns the resource identifier's file name, if present, or the kind
// and name, if present
func (r *ResourceIdentifierOrFile) Title() string {
	if r.fp != "" {
		return filepath.Base(r.fp)
	}
	if r.Name == "" {
		return r.Arg
	}
	return r.Arg + "/" + r.Name
}

// UnmarshalYAML is a custom unmarshaler that understands that the value of the
// ResourceIdentifierOrFile can be either a string or a selector.
func (r *ResourceIdentifierOrFile) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode && node.Kind != yaml.MappingNode {
		return api.ExpectedScalarOrMapAt(node)
	}
	var s string
	// A resource identifier can be a filepath, a string of the form
	// {type}/{name} or {type}.
	if err := node.Decode(&s); err == nil {
		if probablyFilePath(s) {
			if !fileExists(s) {
				return api.FileNotFound(s, node)
			}
			r.fp = s
			return nil
		}
		if strings.ContainsAny(s, " ,;\n\t\r") {
			return InvalidResourceSpecifierOrFilepath(s, node)
		}
		if strings.Count(s, "/") > 1 {
			return InvalidResourceSpecifierOrFilepath(s, node)
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
	_, err := labels.ValidatedSelectorFromSet(ri.Labels)
	if err != nil {
		return InvalidWithLabels(err, node)
	}
	r.Arg = ri.Type
	r.Name = ri.Name
	r.Labels = ri.Labels
	return nil
}

func NewResourceIdentifierOrFile(
	fp string,
	arg string,
	name string,
	labels map[string]string,
) *ResourceIdentifierOrFile {
	return &ResourceIdentifierOrFile{
		fp:     fp,
		Arg:    arg,
		Name:   name,
		Labels: labels,
	}
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
