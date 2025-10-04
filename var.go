// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"context"
	"fmt"

	"github.com/gdt-dev/core/api"
	gdtjson "github.com/gdt-dev/core/assertion/json"
	"github.com/gdt-dev/core/debug"
	"github.com/theory/jsonpath"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type VarEntry struct {
	// From is a string that indicates where the value of the variable will be
	// sourced from. This string is a JSONPath expression that contains
	// instructions on how to extract a particular field from a Kubernetes
	// resource fetched in the `kube.get` command.
	From string `yaml:"from"`
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
		debug.Printf(ctx, "save.vars: %s -> %v", varName, extracted)
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
		results := make([]any, len(out.Items))
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
	p, err := jsonpath.Parse(path)
	if err != nil {
		// Not terminal because during parse we validate the JSONPath
		// expression is valid.
		return nil, gdtjson.JSONPathNotFound(path, err)
	}
	nodes := p.Select(normalized)
	if len(nodes) == 0 {
		return nil, gdtjson.JSONPathNotFound(path, err)
	}
	got := nodes[0]
	return got, nil
}
