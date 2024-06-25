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
	// Labels is a map, keyed by metadata Label, of Label values to select a
	// resource by
	Labels map[string]string `yaml:"labels,omitempty"`
}

// ResourceIdentifier is a struct used to parse an interface{} that can be
// either a string or a struct containing a selector with things like a label
// key/value map.
type ResourceIdentifier struct {
	kind   string            `yaml:"-"`
	name   string            `yaml:"-"`
	labels map[string]string `yaml:"-"`
}

// Title returns the resource identifier's kind and name, if present
func (r *ResourceIdentifier) Title() string {
	if r.name == "" {
		return r.kind
	}
	return r.kind + "/" + r.name
}

// KindName returns the resource identifier's kind and name
func (r *ResourceIdentifier) KindName() (string, string) {
	return r.kind, r.name
}

// Labels returns the resource identifier's labels map, if present
func (r *ResourceIdentifier) Labels() map[string]string {
	return r.labels
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
		r.kind, r.name = splitKindName(s)
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
	r.kind = ri.Type
	r.name = ""
	r.labels = ri.Labels
	return nil
}

func NewResourceIdentifier(
	kind string,
	name string,
	labels map[string]string,
) *ResourceIdentifier {
	return &ResourceIdentifier{
		kind:   kind,
		name:   name,
		labels: labels,
	}
}

// ResourceIdentifierOrFile is a struct used to parse an interface{} that can
// be either a string, a filepath or a struct containing a selector with things
// like a label key/value map.
type ResourceIdentifierOrFile struct {
	fp     string            `yaml:"-"`
	kind   string            `yaml:"-"`
	name   string            `yaml:"-"`
	labels map[string]string `yaml:"-"`
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
	if r.name == "" {
		return r.kind
	}
	return r.kind + "/" + r.name
}

// KindName returns the resource identifier's kind and name
func (r *ResourceIdentifierOrFile) KindName() (string, string) {
	return r.kind, r.name
}

// Labels returns the resource identifier's labels map, if present
func (r *ResourceIdentifierOrFile) Labels() map[string]string {
	return r.labels
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
		r.kind, r.name = splitKindName(s)
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
	r.kind = ri.Type
	r.name = ""
	r.labels = ri.Labels
	return nil
}

func NewResourceIdentifierOrFile(
	fp string,
	kind string,
	name string,
	labels map[string]string,
) *ResourceIdentifierOrFile {
	return &ResourceIdentifierOrFile{
		fp:     fp,
		kind:   kind,
		name:   name,
		labels: labels,
	}
}

// splitKindName returns the Kind for a supplied `Get` or `Delete` command
// where the user can specify either a resource kind or alias, e.g. "pods" or
// "po", or the resource kind followed by a forward slash and a resource name.
func splitKindName(subject string) (string, string) {
	kind, name, _ := strings.Cut(subject, "/")
	return kind, name
}
