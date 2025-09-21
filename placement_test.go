// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube_test

import (
	"bufio"
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	gdtcontext "github.com/gdt-dev/core/context"
	"github.com/gdt-dev/gdt"
	"github.com/stretchr/testify/require"

	kindfix "github.com/gdt-dev/kube/fixtures/kind"
	"github.com/gdt-dev/kube/testutil"
)

func TestPlacementSpread(t *testing.T) {
	testutil.SkipIfNoKind(t)
	require := require.New(t)

	fp := filepath.Join("testdata", "placement-spread.yaml")

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	kindCfgPath := "kind-config-three-workers-three-zones.yaml"

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	ctx := gdtcontext.New(gdtcontext.WithDebug(w))

	ctx = gdtcontext.RegisterFixture(
		ctx, "kind-three-workers-three-zones",
		kindfix.New(
			kindfix.WithClusterName("kind-three-workers-three-zones"),
			kindfix.WithConfigPath(kindCfgPath),
		),
	)

	err = s.Run(ctx, t)
	require.Nil(err)

	w.Flush()
	fmt.Println(b.String())
}
