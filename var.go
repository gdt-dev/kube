// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"context"
	"fmt"

	"github.com/PaesslerAG/jsonpath"
	"github.com/gdt-dev/gdt/api"
	gdtjson "github.com/gdt-dev/gdt/assertion/json"
	"github.com/gdt-dev/gdt/debug"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	// defining the JSONPath language here allows us to disaggregate parse
	// errors from runtime errors when evaluating a JSONPath expression.
	lang = jsonpath.Language()
)

type VarEntry struct {
	// From is a string that indicates where the value of the variable will be
	// sourced from. This string is a JSONPath expression that contains
	// instructions on how to extract a particular field from a Kubernetes
	// resource fetched in the `kube.get` command.
	From string `yaml:"from"`
}

// UnmarshalYAML is a custom unmarshaler that ensures that JSONPath expressions
// contained in the VarEntry are valid.
func (e *VarEntry) UnmarshalYAML(node *yaml.Node) error {
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
		case "from":
			if valNode.Kind != yaml.ScalarNode {
				return api.ExpectedScalarAt(valNode)
			}
			var path string
			if err := valNode.Decode(&path); err != nil {
				return err
			}
			if len(path) == 0 || path[0] != '$' {
				return gdtjson.JSONPathInvalidNoRoot(path, valNode)
			}
			if _, err := lang.NewEvaluable(path); err != nil {
				return gdtjson.JSONPathInvalid(path, err, valNode)
			}
			e.From = path
		}
	}
	return nil
}

// Variables allows the test author to save arbitrary data to the test scenario,
// facilitating the passing of variables between test specs potentially
// provided by different gdt Plugins.
type Variables map[string]VarEntry

// saveVars examines the supplied Variables and what we got back from the
// Action.Do() call and sets any variables in the run data context key.
func saveVars(
	ctx context.Context,
	vars Variables,
	out any,
	res *api.Result,
) error {
	for varName, entry := range vars {
		path := entry.From
		extracted, err := extractFrom(path, out)
		if err != nil {
			return err
		}
		debug.Println(ctx, "save.vars: %s -> %v", varName, extracted)
		res.SetData(varName, extracted)
	}
	return nil
}

func extractFrom(path string, out any) (any, error) {
	var normalized any
	switch out := out.(type) {
	case *unstructured.Unstructured:
		normalized = out.Object
	case *unstructured.UnstructuredList:
		results := make([]map[string]any, len(out.Items))
		for x, item := range out.Items {
			results[x] = item.Object
		}
		normalized = results
	case map[string]any:
		normalized = out
	case []map[string]any:
		normalized = out
	default:
		return nil, fmt.Errorf("unhandled extract type %T", out)
	}
	got, err := jsonpath.Get(path, normalized)
	if err != nil {
		// Shouldn't happen since during parse we validate the JSONPath
		// expression is valid, but double-check anyway.
		return nil, gdtjson.JSONPathNotFound(path, err)
	}
	return got, nil
}
