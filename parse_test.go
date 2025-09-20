// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube_test

import (
	"path/filepath"
	"testing"

	"github.com/gdt-dev/core/api"
	"github.com/gdt-dev/core/parse"
	"github.com/gdt-dev/core/suite"
	"github.com/gdt-dev/gdt"
	gdtkube "github.com/gdt-dev/kube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFailureBadDefaults(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "bad-defaults.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorContains(err, "expected map")
	assert.Error(err, &parse.Error{})
	require.Nil(s)
}

func TestFailureDefaultsConfigNotFound(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "defaults-config-not-found.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorContains(err, "specified kube config path")
	assert.Error(err, &parse.Error{})
	require.Nil(s)
}

func TestFailureBothShortcutAndKubeSpec(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "shortcut-and-long-kube.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorContains(err, "either specify a full KubeSpec")
	assert.Error(err, &parse.Error{})
	require.Nil(s)
}

func TestFailureMoreThanOneKubeAction(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "more-than-one-kube-action.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorContains(err, "you may only specify a single Kubernetes action")
	assert.Error(err, &parse.Error{})
	require.Nil(s)
}

func TestFailureInvalidResourceSpecifierNoMultipleResources(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "invalid-resource-specifier-multiple-resources.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorContains(err, "invalid resource specifier")
	assert.Error(err, &parse.Error{})
	require.Nil(s)
}

func TestFailureInvalidResourceSpecifierMutipleForwardSlashes(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "invalid-resource-specifier-multiple-forward-slashes.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorContains(err, "invalid resource specifier")
	assert.Error(err, &parse.Error{})
	require.Nil(s)
}

func TestFailureInvalidDeleteNotFilepathOrResourceSpecifier(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "invalid-delete-not-filepath-or-resource-specifier.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorContains(err, "invalid resource specifier or filepath")
	assert.Error(err, &parse.Error{})
	require.Nil(s)
}

func TestFailureCreateFileNotFound(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "create-file-not-found.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorContains(err, "file not found")
	assert.Error(err, &parse.Error{})
	require.Nil(s)
}

func TestFailureDeleteFileNotFound(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "delete-file-not-found.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorContains(err, "file not found")
	assert.Error(err, &parse.Error{})
	require.Nil(s)
}

func TestFailureBadMatchesFileNotFound(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "bad-matches-file-not-found.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorContains(err, "file not found")
	assert.Error(err, &parse.Error{})
	require.Nil(s)
}

func TestFailureBadMatchesInvalidYAML(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "bad-matches-invalid-yaml.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorContains(err, "`kube.assert.matches` not well-formed")
	assert.Error(err, &parse.Error{})
	require.Nil(s)
}

func TestFailureBadMatchesEmpty(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "bad-matches-empty.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorContains(err, "expected either map[string]interface{} or a string with embedded YAML")
	assert.Error(err, &parse.Error{})
	require.Nil(s)
}

func TestFailureBadMatchesNotMapAny(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "bad-matches-not-map-any.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorContains(err, "`kube.assert.matches` not well-formed")
	assert.Error(err, &parse.Error{})
	require.Nil(s)
}

func TestFailureBadPlacementNotObject(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "bad-placement-not-object.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorContains(err, "expected map")
	assert.Error(err, &parse.Error{})
	require.Nil(s)
}

func TestWithLabelsInvalid(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "with-labels-invalid.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorContains(err, "with labels invalid")
	assert.Error(err, &parse.Error{})
	require.Nil(s)
}

func TestFailureBadVarType(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "bad-var-type.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorContains(err, "expected map")
	assert.Error(err, &parse.Error{})
	require.Nil(s)
}

func TestFailureBadVarJSONPathNoRoot(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "bad-var-jsonpath-noroot.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorContains(err, "expression must start with")
	assert.Error(err, &parse.Error{})
	require.Nil(s)
}

func TestFailureBadVarJSONPath(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "bad-var-jsonpath.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorContains(err, "JSONPath invalid")
	assert.Error(err, &parse.Error{})
	require.Nil(s)
}

func TestParse(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	t.Setenv("pod_name", "foo")

	fp := filepath.Join("testdata", "parse.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	podYAML := `apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  containers:
   - name: nginx
     image: nginx:1.7.9
`
	var zero int

	expTests := []api.Evaluable{
		&gdtkube.Spec{
			Spec: api.Spec{
				Index:    0,
				Name:     "create a pod from YAML using kube.create shortcut",
				Defaults: &api.Defaults{},
			},
			Kube: &gdtkube.KubeSpec{
				Action: gdtkube.Action{
					Create: podYAML,
				},
			},
		},
		&gdtkube.Spec{
			Spec: api.Spec{
				Index:    1,
				Name:     "apply a pod from a file using kube.apply shortcut",
				Defaults: &api.Defaults{},
			},
			Kube: &gdtkube.KubeSpec{
				Action: gdtkube.Action{
					Apply: "manifests/nginx-pod.yaml",
				},
			},
		},
		&gdtkube.Spec{
			Spec: api.Spec{
				Index:    2,
				Name:     "create a pod from YAML",
				Defaults: &api.Defaults{},
			},
			Kube: &gdtkube.KubeSpec{
				Action: gdtkube.Action{
					Create: podYAML,
				},
			},
		},
		&gdtkube.Spec{
			Spec: api.Spec{
				Index:    3,
				Name:     "delete a pod from a file",
				Defaults: &api.Defaults{},
			},
			Kube: &gdtkube.KubeSpec{
				Action: gdtkube.Action{
					Delete: gdtkube.NewResourceIdentifierOrFile(
						"manifests/nginx-pod.yaml",
						"", "", nil,
					),
				},
			},
		},
		&gdtkube.Spec{
			Spec: api.Spec{
				Index:    4,
				Name:     "fetch a pod via kube.get shortcut",
				Defaults: &api.Defaults{},
			},
			Kube: &gdtkube.KubeSpec{
				Action: gdtkube.Action{
					Get: gdtkube.NewResourceIdentifier(
						"pods", "name", nil,
					),
				},
			},
		},
		&gdtkube.Spec{
			Spec: api.Spec{
				Index:    5,
				Name:     "fetch a pod via long-form kube:get",
				Defaults: &api.Defaults{},
			},
			Kube: &gdtkube.KubeSpec{
				Action: gdtkube.Action{
					Get: gdtkube.NewResourceIdentifier(
						"pods", "name", nil,
					),
				},
			},
		},
		&gdtkube.Spec{
			Spec: api.Spec{
				Index:    6,
				Name:     "fetch a pod via kube.get shortcut to long-form resource identifier with labels",
				Defaults: &api.Defaults{},
			},
			Kube: &gdtkube.KubeSpec{
				Action: gdtkube.Action{
					Get: gdtkube.NewResourceIdentifier(
						"pods", "", map[string]string{
							"app": "nginx",
						},
					),
				},
			},
		},
		&gdtkube.Spec{
			Spec: api.Spec{
				Index:    7,
				Name:     "fetch a pod via kube:get long-form resource identifier with labels",
				Defaults: &api.Defaults{},
			},
			Kube: &gdtkube.KubeSpec{
				Action: gdtkube.Action{
					Get: gdtkube.NewResourceIdentifier(
						"pods", "", map[string]string{
							"app": "nginx",
						},
					),
				},
			},
		},
		&gdtkube.Spec{
			Spec: api.Spec{
				Index:    8,
				Name:     "fetch a pod with envvar substitution",
				Defaults: &api.Defaults{},
			},
			Kube: &gdtkube.KubeSpec{
				Action: gdtkube.Action{
					Get: gdtkube.NewResourceIdentifier(
						"pods", "foo", nil,
					),
				},
			},
			Assert: &gdtkube.Expect{
				Len: &zero,
			},
		},
		&gdtkube.Spec{
			Spec: api.Spec{
				Index:    9,
				Name:     "define a gdt variable",
				Defaults: &api.Defaults{},
			},
			Kube: &gdtkube.KubeSpec{
				Action: gdtkube.Action{
					Get: gdtkube.NewResourceIdentifier(
						"pods", "name", nil,
					),
				},
			},
			Var: gdtkube.Variables{
				"POD": gdtkube.VarEntry{
					From: "$.metadata.name",
				},
			},
		},
		&gdtkube.Spec{
			Spec: api.Spec{
				Index:    10,
				Name:     "fetch a pod with gdt variable system substitution",
				Defaults: &api.Defaults{},
			},
			Kube: &gdtkube.KubeSpec{
				Action: gdtkube.Action{
					Get: gdtkube.NewResourceIdentifier(
						"pods",
						"$POD", // $$POD is replaced with $POD after envvar substitution...
						nil,
					),
				},
			},
		},
	}
	su := s.(*suite.Suite)
	require.Len(su.Scenarios, 1)
	sc := su.Scenarios[0]
	require.Len(sc.Tests, len(expTests))
	for x, st := range sc.Tests {
		exp := expTests[x].(*gdtkube.Spec)
		stk := st.(*gdtkube.Spec)
		assert.Equal(exp.Kube, stk.Kube)
		assert.Equal(exp.Var, stk.Var)
		assert.Equal(exp.Assert, stk.Assert)
	}
}
