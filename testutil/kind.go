package testutil

import (
	"os"
	"testing"
)

func SkipIfNoKind(t *testing.T) {
	_, found := os.LookupEnv("SKIP_KIND")
	if found {
		t.Skipf("skipping KinD-requiring test")
	}
}
