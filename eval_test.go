// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gdt-dev/gdt"
	gdtcontext "github.com/gdt-dev/gdt/context"
	kindfix "github.com/gdt-dev/kube/fixtures/kind"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnknownKubeContextInSpec(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "unknown-context.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	err = s.Run(context.TODO(), t)
	assert.NotNil(err)
	assert.ErrorContains(err, "context \"unknownctx\" does not exist")
}

func TestUnknownKubeContextInDefaults(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	fp := filepath.Join("testdata", "parse", "fail", "unknown-context-in-defaults.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	err = s.Run(context.TODO(), t)
	require.NotNil(err)
	assert.ErrorContains(err, "context \"unknownctx\" does not exist")
}

func TestListPodsEmpty(t *testing.T) {
	skipIfKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "list-pods-empty.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", kindfix.New())

	err = s.Run(ctx, t)
	require.Nil(err, "%s", err)
}

func TestGetPodNotFound(t *testing.T) {
	skipIfKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "get-pod-not-found.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", kindfix.New())

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestCreateUnknownResource(t *testing.T) {
	skipIfKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "create-unknown-resource.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", kindfix.New())

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestDeleteResourceNotFound(t *testing.T) {
	skipIfKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "delete-resource-not-found.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", kindfix.New())

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestDeleteUnknownResource(t *testing.T) {
	skipIfKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "delete-unknown-resource.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", kindfix.New())

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestPodCreateGetDelete(t *testing.T) {
	skipIfKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "create-get-delete-pod.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", kindfix.New())

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestMatches(t *testing.T) {
	skipIfKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "matches.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", kindfix.New())

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestConditions(t *testing.T) {
	skipIfKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "conditions.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", kindfix.New())

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestJSON(t *testing.T) {
	skipIfKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "json.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", kindfix.New())

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestApply(t *testing.T) {
	skipIfKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "apply-deployment.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", kindfix.New())

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestEnvvarSubstitution(t *testing.T) {
	skipIfKind(t)
	require := require.New(t)

	t.Setenv("pod_name", "foo")

	fp := filepath.Join("testdata", "envvar-substitution.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", kindfix.New())

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestWithLabels(t *testing.T) {
	skipIfKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "list-pods-with-labels.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", kindfix.New())

	err = s.Run(ctx, t)
	require.Nil(err)
}

func skipIfKind(t *testing.T) {
	_, found := os.LookupEnv("SKIP_KIND")
	if found {
		t.Skipf("skipping KinD-requiring test")
	}
}
