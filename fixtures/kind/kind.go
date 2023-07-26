// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kind

import (
	"bytes"
	"strings"

	gdttypes "github.com/gdt-dev/gdt/types"
	"github.com/samber/lo"
	"sigs.k8s.io/kind/pkg/cluster"
	kindconst "sigs.k8s.io/kind/pkg/cluster/constants"
	kubeyaml "sigs.k8s.io/yaml"

	gdtkube "github.com/gdt-dev/kube"
)

const (
	workdirNamePattern = "gdt-kube.kindfix.*"
)

// KindFixture implements `gdttypes.Fixture` and exposes connection/config
// information about a running KinD cluster.
type KindFixture struct {
	// provider is the KinD cluster provider
	provider *cluster.Provider
	// cfgStr contains the stringified KUBECONFIG that KinD returns in its
	// Provider.KubeConfig() call
	cfgStr string
	// ClusterName is the name of the KinD cluster. If not specified, gdt will
	// use the default cluster name that KinD uses, which is just "kind"
	ClusterName string
	// Context is the name of the kubecontext to use. If not specified, gdt
	// will use the default KinD context, which is "kind-{cluster_name}"
	// See https://github.com/kubernetes-sigs/kind/blob/3610f606516ccaa88aa098465d8c13af70937050/pkg/cluster/internal/kubeconfig/internal/kubeconfig/helpers.go#L23-L26
	Context string
}

func (f *KindFixture) Start() {
	if f.ClusterName == "" {
		f.ClusterName = kindconst.DefaultClusterName
	}
	if f.isRunning() {
		return
	}
	if err := f.provider.Create(f.ClusterName); err != nil {
		panic(err)
	}
}

func (f *KindFixture) isRunning() bool {
	if f.provider == nil || f.ClusterName == "" {
		return false
	}
	clusterNames, err := f.provider.List()
	if err != nil {
		return false
	}
	return lo.Contains(clusterNames, f.ClusterName)
}

func (f *KindFixture) Stop() {}

func (f *KindFixture) HasState(key string) bool {
	lkey := strings.ToLower(key)
	switch lkey {
	case gdtkube.StateKeyConfigBytes, gdtkube.StateKeyContext:
		return true
	}
	return false
}

func (f *KindFixture) State(key string) interface{} {
	key = strings.ToLower(key)
	switch key {
	case gdtkube.StateKeyConfigBytes:
		if f.provider == nil {
			return []byte{}
		}
		cfg, err := f.provider.KubeConfig(f.ClusterName, false)
		if err != nil {
			panic(err)
		}
		return []byte(cfg)
	case gdtkube.StateKeyContext:
		if f.Context != "" {
			return f.Context
		}
		if f.ClusterName == "" {
			return ""
		}
		return "kind-" + f.ClusterName
	}
	return ""
}

// normYAML round trips yaml bytes through sigs.k8s.io/yaml to normalize them
// versus other kubernetes ecosystem yaml output
func normYAML(y []byte) ([]byte, error) {
	var unstructured interface{}
	if err := kubeyaml.Unmarshal(y, &unstructured); err != nil {
		return nil, err
	}
	encoded, err := kubeyaml.Marshal(&unstructured)
	if err != nil {
		return nil, err
	}
	// special case: don't write anything when empty
	if bytes.Equal(encoded, []byte("{}\n")) {
		return []byte{}, nil
	}
	return encoded, nil
}

type KindFixtureModifier func(*KindFixture)

// WithClusterName modifies the KindFixture's cluster name
func WithClusterName(name string) KindFixtureModifier {
	return func(f *KindFixture) {
		f.ClusterName = name
	}
}

// WithContext modifies the KindFixture's kubecontext
func WithContext(name string) KindFixtureModifier {
	return func(f *KindFixture) {
		f.Context = name
	}
}

// New returns a fixture that exposes Kubernetes configuration/context
// information about a KinD cluster. If no such KinD cluster exists, one will
// be created. If the KinD cluster is created, it is destroyed at the end of
// the fixture's scope (when Fixture.Stop() is called).  The returned fixture
// exposes some state keys:
//
//   - "kube.config" returns the path of the kubeconfig file to use with this
//     KinD cluster
//   - "kube.context" returns the kubecontext to use with this KinD cluster
func New(mods ...KindFixtureModifier) gdttypes.Fixture {
	f := &KindFixture{
		provider: cluster.NewProvider(),
	}
	for _, mod := range mods {
		mod(f)
	}
	return f
}
