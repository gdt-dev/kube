// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kind

import (
	"context"
	"strings"

	gdtcontext "github.com/gdt-dev/gdt/context"
	"github.com/gdt-dev/gdt/debug"
	gdttypes "github.com/gdt-dev/gdt/types"
	"github.com/samber/lo"
	"sigs.k8s.io/kind/pkg/cluster"
	kindconst "sigs.k8s.io/kind/pkg/cluster/constants"

	gdtkube "github.com/gdt-dev/kube"
)

// KindFixture implements `gdttypes.Fixture` and exposes connection/config
// information about a running KinD cluster.
type KindFixture struct {
	// provider is the KinD cluster provider
	provider *cluster.Provider
	// deleteOnStop indicates that the KinD cluster should be deleted when
	// the fixture is stopped. Fixtures are stopped when test scenarios
	// utilizing the fixture have executed all their test steps.
	//
	// By default, KinD clusters that were already running when the fixture was
	// started are not deleted. This is to prevent the deletion of KinD
	// clusters that were in use outside of a gdt-kube execution. To override
	// this behaviour and always delete the KinD cluster on stop, use the
	// WithDeleteOnStop() modifier.
	deleteOnStop bool
	// retainOnStop indicates that the KinD cluster should *not* be deleted
	// when the fixture is stopped. Fixtures are stopped when test scenarios
	// utilizing the fixture have executed all their test steps.
	//
	// By default, KinD clusters that were *not* already running when the fixture was
	// started are deleted when the fixture stops. This is to clean up KinD
	// clusters that were created and used by the gdt-kube execution. To override
	// this behaviour and always retain the KinD cluster on stop, use the
	// WithRetainOnStop() modifier.
	retainOnStop bool
	// runningBeforeStart is true when the KinD cluster was already running
	// when the fixture was started.
	runningBeforeStart bool
	// ClusterName is the name of the KinD cluster. If not specified, gdt will
	// use the default cluster name that KinD uses, which is just "kind"
	ClusterName string
	// Context is the name of the kubecontext to use. If not specified, gdt
	// will use the default KinD context, which is "kind-{cluster_name}"
	// See https://github.com/kubernetes-sigs/kind/blob/3610f606516ccaa88aa098465d8c13af70937050/pkg/cluster/internal/kubeconfig/internal/kubeconfig/helpers.go#L23-L26
	Context string
	// ConfigPath is a path to the v1alpha4 KinD configuration CR
	ConfigPath string
}

func (f *KindFixture) Start(ctx context.Context) {
	ctx = gdtcontext.PushTrace(ctx, "fixtures.kind.start")
	defer func() {
		ctx = gdtcontext.PopTrace(ctx)
	}()
	if f.ClusterName == "" {
		f.ClusterName = kindconst.DefaultClusterName
	}
	if f.isRunning() {
		debug.Println(ctx, "cluster %s already running", f.ClusterName)
		f.runningBeforeStart = true
		return
	}
	opts := []cluster.CreateOption{}
	if f.ConfigPath != "" {
		debug.Println(
			ctx, "using custom kind config %s for cluster %s",
			f.ConfigPath, f.ClusterName,
		)
		opts = append(opts, cluster.CreateWithConfigFile(f.ConfigPath))
	}
	if err := f.provider.Create(f.ClusterName, opts...); err != nil {
		panic(err)
	}
	debug.Println(ctx, "cluster %s successfully created", f.ClusterName)
	if !f.retainOnStop {
		f.deleteOnStop = true
		debug.Println(ctx, "cluster %s will be deleted on stop", f.ClusterName)
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

func (f *KindFixture) Stop(ctx context.Context) {
	ctx = gdtcontext.PushTrace(ctx, "fixtures.kind.stop")
	defer func() {
		ctx = gdtcontext.PopTrace(ctx)
	}()
	if !f.isRunning() {
		debug.Println(ctx, "cluster %s not running", f.ClusterName)
		return
	}
	if f.runningBeforeStart && !f.deleteOnStop {
		debug.Println(ctx, "cluster %s was running before start and deleteOnStop=false so not deleting", f.ClusterName)
		return
	}
	if f.deleteOnStop {
		if err := f.provider.Delete(f.ClusterName, ""); err != nil {
			panic(err)
		}
		debug.Println(ctx, "cluster %s successfully deleted", f.ClusterName)
	}
}

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

// WithConfigPath configures a path to a KinD configuration CR to use
func WithConfigPath(path string) KindFixtureModifier {
	return func(f *KindFixture) {
		f.ConfigPath = path
	}
}

// WithDeleteOnStop instructs gdt-kube to always delete the KinD cluster when
// the fixture stops. Fixtures are stopped when test scenarios utilizing the
// fixture have executed all their test steps.
//
// By default, KinD clusters that were already running when the fixture was
// started are not deleted. This is to prevent the deletion of KinD
// clusters that were in use outside of a gdt-kube execution. To override
// this behaviour and always delete the KinD cluster on stop, use the
// WithDeleteOnStop() modifier.
func WithDeleteOnStop() KindFixtureModifier {
	return func(f *KindFixture) {
		f.deleteOnStop = true
	}
}

// WithRetainOnStop instructs gdt-kube that the KinD cluster should *not* be
// deleted when the fixture is stopped. Fixtures are stopped when test
// scenarios utilizing the fixture have executed all their test steps.
//
// By default, KinD clusters that were *not* already running when the fixture
// was started are deleted when the fixture stops. This is to clean up KinD
// clusters that were created and used by the gdt-kube execution. To override
// this behaviour and always retain the KinD cluster on stop, use the
// WithRetainOnStop() modifier.
func WithRetainOnStop() KindFixtureModifier {
	return func(f *KindFixture) {
		f.retainOnStop = true
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
