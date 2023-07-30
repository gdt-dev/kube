// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gdt-dev/gdt"
	"github.com/gdt-dev/gdt/errors"
	gdttypes "github.com/gdt-dev/gdt/types"
	gdtkube "github.com/gdt-dev/kube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func currentDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
}

func TestBadDefaults(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "bad-defaults.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorIs(err, errors.ErrExpectedMap)
	require.Nil(s)
}

func TestFailureDefaultsConfigNotFound(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "defaults-config-not-found.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorIs(err, gdtkube.ErrKubeConfigNotFound)
	assert.ErrorIs(err, errors.ErrParse)
	require.Nil(s)
}

func TestFailureBothShortcutAndKubeSpec(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "shortcut-and-long-kube.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorIs(err, gdtkube.ErrEitherShortcutOrKubeSpec)
	assert.ErrorIs(err, errors.ErrParse)
	require.Nil(s)
}

func TestFailureMoreThanOneShortcut(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "more-than-one-shortcut.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorIs(err, gdtkube.ErrMoreThanOneShortcut)
	assert.ErrorIs(err, errors.ErrParse)
	require.Nil(s)
}

func TestFailureMoreThanOneKubeAction(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "more-than-one-kube-action.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorIs(err, gdtkube.ErrMoreThanOneKubeAction)
	assert.ErrorIs(err, errors.ErrParse)
	require.Nil(s)
}

func TestFailureInvalidResourceSpecifierNoMultipleResources(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "invalid-resource-specifier-multiple-resources.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorIs(err, gdtkube.ErrResourceSpecifierInvalid)
	assert.ErrorIs(err, errors.ErrParse)
	require.Nil(s)
}

func TestFailureInvalidResourceSpecifierMutipleForwardSlashes(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "invalid-resource-specifier-multiple-forward-slashes.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorIs(err, gdtkube.ErrResourceSpecifierInvalid)
	assert.ErrorIs(err, errors.ErrParse)
	require.Nil(s)
}

func TestFailureInvalidDeleteNotFilepathOrResourceSpecifier(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "invalid-delete-not-filepath-or-resource-specifier.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorIs(err, gdtkube.ErrResourceSpecifierInvalidOrFilepath)
	assert.ErrorIs(err, errors.ErrParse)
	require.Nil(s)
}

func TestFailureCreateFileNotFound(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "create-file-not-found.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorIs(err, errors.ErrFileNotFound)
	assert.ErrorIs(err, errors.ErrParse)
	require.Nil(s)
}

func TestDeleteFileNotFound(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "delete-file-not-found.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorIs(err, errors.ErrFileNotFound)
	assert.ErrorIs(err, errors.ErrParse)
	require.Nil(s)
}

func TestFailureBadMatchesFileNotFound(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "bad-matches-file-not-found.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorIs(err, errors.ErrFileNotFound)
	assert.ErrorIs(err, errors.ErrParse)
	require.Nil(s)
}

func TestFailureBadMatchesInvalidYAML(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "bad-matches-invalid-yaml.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorIs(err, gdtkube.ErrMatchesInvalid)
	assert.ErrorIs(err, errors.ErrParse)
	require.Nil(s)
}

func TestFailureBadMatchesNotMapAny(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "bad-matches-not-map-any.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorIs(err, gdtkube.ErrMatchesInvalid)
	assert.ErrorIs(err, errors.ErrParse)
	require.Nil(s)
}

func TestWithLabelsOnlyGetDelete(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "with-labels-only-get-delete.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorIs(err, gdtkube.ErrWithLabelsOnlyGetDelete)
	assert.ErrorIs(err, errors.ErrParse)
	require.Nil(s)
}

func TestWithLabelsInvalid(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "with-labels-invalid.yaml")

	s, err := gdt.From(fp)
	require.NotNil(err)
	assert.ErrorIs(err, gdtkube.ErrWithLabelsInvalid)
	assert.ErrorIs(err, errors.ErrParse)
	require.Nil(s)
}

func TestParse(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	t.Setenv("pod_name", "foo")

	fp := filepath.Join("testdata", "parse.yaml")

	suite, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(suite)

	require.Len(suite.Scenarios, 1)
	s := suite.Scenarios[0]

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

	expTests := []gdttypes.Evaluable{
		&gdtkube.Spec{
			Spec: gdttypes.Spec{
				Index:    0,
				Name:     "create a pod from YAML using kube.create shortcut",
				Defaults: &gdttypes.Defaults{},
			},
			KubeCreate: podYAML,
			Kube: &gdtkube.KubeSpec{
				Create: podYAML,
			},
		},
		&gdtkube.Spec{
			Spec: gdttypes.Spec{
				Index:    1,
				Name:     "apply a pod from a file using kube.apply shortcut",
				Defaults: &gdttypes.Defaults{},
			},
			KubeApply: "testdata/manifests/nginx-pod.yaml",
			Kube: &gdtkube.KubeSpec{
				Apply: "testdata/manifests/nginx-pod.yaml",
			},
		},
		&gdtkube.Spec{
			Spec: gdttypes.Spec{
				Index:    2,
				Name:     "create a pod from YAML",
				Defaults: &gdttypes.Defaults{},
			},
			Kube: &gdtkube.KubeSpec{
				Create: podYAML,
			},
		},
		&gdtkube.Spec{
			Spec: gdttypes.Spec{
				Index:    3,
				Name:     "delete a pod from a file",
				Defaults: &gdttypes.Defaults{},
			},
			Kube: &gdtkube.KubeSpec{
				Delete: "testdata/manifests/nginx-pod.yaml",
			},
		},
		&gdtkube.Spec{
			Spec: gdttypes.Spec{
				Index:    4,
				Name:     "fetch a pod via kube.get shortcut",
				Defaults: &gdttypes.Defaults{},
			},
			KubeGet: "pods/name",
			Kube: &gdtkube.KubeSpec{
				Get: "pods/name",
			},
		},
		&gdtkube.Spec{
			Spec: gdttypes.Spec{
				Index:    5,
				Name:     "fetch a pod via long-form kube:get",
				Defaults: &gdttypes.Defaults{},
			},
			Kube: &gdtkube.KubeSpec{
				Get: "pods/name",
			},
		},
		&gdtkube.Spec{
			Spec: gdttypes.Spec{
				Index:    6,
				Name:     "fetch a pod with envvar substitution",
				Defaults: &gdttypes.Defaults{},
			},
			Kube: &gdtkube.KubeSpec{
				Get: "pods/foo",
			},
			Assert: &gdtkube.Expect{
				Len: &zero,
			},
		},
	}
	assert.Equal(expTests, s.Tests)
}
