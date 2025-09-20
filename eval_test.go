// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube_test

import (
	"bufio"
	"bytes"
	"path/filepath"
	"testing"

	gdtcontext "github.com/gdt-dev/core/context"
	"github.com/gdt-dev/gdt"
	"github.com/stretchr/testify/require"

	kindfix "github.com/gdt-dev/kube/fixtures/kind"
	"github.com/gdt-dev/kube/testutil"
)

var stdKindFix = kindfix.New(
	kindfix.WithRetainOnStop(),
)

func TestKindListPodsEmpty(t *testing.T) {
	testutil.SkipIfNoKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "kind", "list-pods-empty.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", stdKindFix)

	err = s.Run(ctx, t)
	require.Nil(err, "%s", err)
}

func TestKindGetPodNotFound(t *testing.T) {
	testutil.SkipIfNoKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "kind", "get-pod-not-found.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", stdKindFix)

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestKindCreateUnknownResource(t *testing.T) {
	testutil.SkipIfNoKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "kind", "create-unknown-resource.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", stdKindFix)

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestKindSameNamedKind(t *testing.T) {
	testutil.SkipIfNoKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "kind", "same-named-kind.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", stdKindFix)

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestKindDeleteResourceNotFound(t *testing.T) {
	testutil.SkipIfNoKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "kind", "delete-resource-not-found.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", stdKindFix)

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestKindDeleteUnknownResource(t *testing.T) {
	testutil.SkipIfNoKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "kind", "delete-unknown-resource.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", stdKindFix)

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestKindPodCreateGetDelete(t *testing.T) {
	testutil.SkipIfNoKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "kind", "create-get-delete-pod.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", stdKindFix)

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestKindMatches(t *testing.T) {
	testutil.SkipIfNoKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "kind", "matches.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", stdKindFix)

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestKindConditions(t *testing.T) {
	testutil.SkipIfNoKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "kind", "conditions.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", stdKindFix)

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestKindJSON(t *testing.T) {
	testutil.SkipIfNoKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "kind", "json.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", stdKindFix)

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestKindApply(t *testing.T) {
	testutil.SkipIfNoKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "kind", "apply-deployment.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", stdKindFix)

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestKindEnvvarSubstitution(t *testing.T) {
	testutil.SkipIfNoKind(t)
	require := require.New(t)

	t.Setenv("pod_name", "foo")

	fp := filepath.Join("testdata", "kind", "envvar-substitution.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", stdKindFix)

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestKindWithLabels(t *testing.T) {
	testutil.SkipIfNoKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "kind", "list-pods-with-labels.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", stdKindFix)

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestKindVarSaveRestore(t *testing.T) {
	testutil.SkipIfNoKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "kind", "var-save-restore.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	ctx := gdtcontext.New(gdtcontext.WithDebug(w))
	ctx = gdtcontext.RegisterFixture(ctx, "kind", stdKindFix)

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestKindCurlPodIP(t *testing.T) {
	testutil.SkipIfNoKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "kind", "curl-pod-ip.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	ctx := gdtcontext.New(gdtcontext.WithDebug(w))
	ctx = gdtcontext.RegisterFixture(ctx, "kind", stdKindFix)

	err = s.Run(ctx, t)
	require.Nil(err)
}
