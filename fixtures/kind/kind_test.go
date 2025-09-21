// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kind_test

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	gdtcontext "github.com/gdt-dev/core/context"
	"github.com/gdt-dev/core/scenario"
	kindfix "github.com/gdt-dev/kube/fixtures/kind"
	"github.com/stretchr/testify/require"
)

func TestDefaultSingleControlPlane(t *testing.T) {
	skipKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "default-single-control-plane.yaml")
	f, err := os.Open(fp)
	require.Nil(err)
	defer f.Close() // nolint:errcheck

	s, err := scenario.FromReader(f, scenario.WithPath(fp))
	require.Nil(err)
	require.NotNil(s)

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	ctx := gdtcontext.New(gdtcontext.WithDebug(w))
	ctx = gdtcontext.RegisterFixture(
		ctx, "kind",
		kindfix.New(
			kindfix.WithDeleteOnStop(),
		),
	)

	err = s.Run(ctx, t)
	w.Flush()
	fmt.Println(b.String())
	require.Nil(err)
}

func TestOneControlPlaneOneWorker(t *testing.T) {
	skipKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "one-control-plane-one-worker.yaml")
	f, err := os.Open(fp)
	require.Nil(err)
	defer f.Close() // nolint:errcheck

	s, err := scenario.FromReader(f, scenario.WithPath(fp))
	require.Nil(err)
	require.NotNil(s)

	kindCfgPath := "kind-config-one-cp-one-worker.yaml"

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	ctx := gdtcontext.New(gdtcontext.WithDebug(w))
	ctx = gdtcontext.RegisterFixture(
		ctx, "kind-one-cp-one-worker",
		kindfix.New(
			kindfix.WithClusterName("kind-one-cp-one-worker"),
			kindfix.WithConfigPath(kindCfgPath),
		),
	)

	err = s.Run(ctx, t)
	w.Flush()
	fmt.Println(b.String())
	require.Nil(err)
}

func skipKind(t *testing.T) {
	_, found := os.LookupEnv("SKIP_KIND")
	if found {
		t.Skipf("skipping KinD-requiring test")
	}
}
