// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"path/filepath"

	kubelabels "k8s.io/apimachinery/pkg/labels"
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
	// LabelsIn is a map, keyed by metadata Label, of slices of string label
	// values to select a resource with an IN() selector.
	LabelsIn map[string][]string `yaml:"labels-in,omitempty"`
	// LabelsNotIn is a map, keyed by metadata Label, of slices of string label
	// values to select a resource with an NOT IN() selector.
	LabelsNotIn   map[string][]string `yaml:"labels-not-in,omitempty"`
	LabelSelector kubelabels.Selector `yaml:"-"`
}

// ResourceIdentifier is a struct used to parse an interface{} that can be
// either a string or a struct containing a selector with things like a label
// key/value map.
type ResourceIdentifier struct {
	Arg           string              `yaml:"-"`
	Name          string              `yaml:"-"`
	LabelSelector kubelabels.Selector `yaml:"-"`
}

// Title returns the resource identifier's kind and name, if present
func (r *ResourceIdentifier) Title() string {
	if r.Name == "" {
		return r.Arg
	}
	return r.Arg + "/" + r.Name
}

func NewResourceIdentifier(
	arg string,
	name string,
	labels map[string]string,
) (*ResourceIdentifier, error) {
	sel := kubelabels.Everything()
	if len(labels) > 0 {
		ls, err := kubelabels.ValidatedSelectorFromSet(labels)
		if err != nil {
			return nil, err
		}
		sel = ls
	}
	ri := &ResourceIdentifier{
		Arg:  arg,
		Name: name,
	}
	if !sel.Empty() {
		ri.LabelSelector = sel.DeepCopySelector()
	}
	return ri, nil
}

// ResourceIdentifierOrFile is a struct used to parse an interface{} that can
// be either a string, a filepath or a struct containing a selector with things
// like a label key/value map.
type ResourceIdentifierOrFile struct {
	fp            string              `yaml:"-"`
	Arg           string              `yaml:"-"`
	Name          string              `yaml:"-"`
	LabelSelector kubelabels.Selector `yaml:"-"`
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

func NewResourceIdentifierOrFile(
	fp string,
	arg string,
	name string,
	labels map[string]string,
) (*ResourceIdentifierOrFile, error) {
	sel := kubelabels.Everything()
	if len(labels) > 0 {
		ls, err := kubelabels.ValidatedSelectorFromSet(labels)
		if err != nil {
			return nil, err
		}
		sel = ls
	}
	ri := &ResourceIdentifierOrFile{
		fp:   fp,
		Arg:  arg,
		Name: name,
	}
	if !sel.Empty() {
		ri.LabelSelector = sel.DeepCopySelector()
	}
	return ri, nil
}
