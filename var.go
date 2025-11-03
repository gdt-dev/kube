// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"context"
	"fmt"

	"github.com/gdt-dev/core/api"
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
		extracted, err := extractFrom(varName, path, out)
		if err != nil {
			return err
		}
		debug.Printf(ctx, "save.vars: %s -> %v", varName, extracted)
		res.SetData(varName, extracted)
	}
	return nil
}

func extractFrom(
	varName string,
	path string,
	out any,
) (any, error) {
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
	// Ignore error because during parse we validate the JSONPath expression is
	// valid.
	p, _ := jsonpath.Parse(path)
	nodes := p.Select(normalized)
	if len(nodes) == 0 {
		// This IS terminal because it means that the returned results of the
		// kube.get call did not match the expected JSONPath and that's a
		// RuntimeError because we cannot continue execution if we don't match
		// the JSONPath query.
		return nil, api.JSONPathVarFromNotMatched(varName, path)
	}
	got := nodes[0]
	return got, nil
}
