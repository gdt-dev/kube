// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kind_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gdt-dev/gdt"
	gdtcontext "github.com/gdt-dev/gdt/context"
	kindfix "github.com/gdt-dev/kube/fixtures/kind"
	"github.com/stretchr/testify/require"
)

func TestDefaultSingleControlPlane(t *testing.T) {
	skipKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "default-single-control-plane.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(ctx, "kind", kindfix.New())

	err = s.Run(ctx, t)
	require.Nil(err)
}

func TestOneControlPlaneOneWorker(t *testing.T) {
	skipKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "one-control-plane-one-worker.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	kindCfgPath := filepath.Join("testdata", "kind-config-one-cp-one-worker.yaml")

	ctx := gdtcontext.New()
	ctx = gdtcontext.RegisterFixture(
		ctx, "kind-one-cp-one-worker",
		kindfix.New(
			kindfix.WithClusterName("kind-one-cp-one-worker"),
			kindfix.WithConfigPath(kindCfgPath),
		),
	)

	err = s.Run(ctx, t)
	require.Nil(err)
}

func skipKind(t *testing.T) {
	_, found := os.LookupEnv("SKIP_KIND")
	if found {
		t.Skipf("skipping KinD-requiring test")
	}
}
