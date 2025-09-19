// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kind

import (
	"context"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/gdt-dev/core/api"
	gdtcontext "github.com/gdt-dev/core/context"
	"github.com/gdt-dev/core/debug"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/cluster"
	kindconst "sigs.k8s.io/kind/pkg/cluster/constants"

	gdtkube "github.com/gdt-dev/kube"
)

var (
	checkDefaultServiceAccountTimeout = time.Second * 15
)

// KindFixture implements `api.Fixture` and exposes connection/config
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

func (f *KindFixture) Start(ctx context.Context) error {
	ctx = gdtcontext.PushTrace(ctx, "fixtures.kind.start")
	defer func() {
		ctx = gdtcontext.PopTrace(ctx)
	}()
	if f.ClusterName == "" {
		f.ClusterName = kindconst.DefaultClusterName
	}
	if f.isRunning() {
		debug.Printf(ctx, "cluster %s already running", f.ClusterName)
		f.runningBeforeStart = true
		return f.waitForDefaultServiceAccount(ctx)
	}
	opts := []cluster.CreateOption{}
	if f.ConfigPath != "" {
		debug.Printf(
			ctx, "using custom kind config %s for cluster %s",
			f.ConfigPath, f.ClusterName,
		)
		opts = append(opts, cluster.CreateWithConfigFile(f.ConfigPath))
	}
	if err := f.provider.Create(f.ClusterName, opts...); err != nil {
		return err
	}
	debug.Printf(ctx, "cluster %s successfully created", f.ClusterName)
	if !f.retainOnStop {
		f.deleteOnStop = true
		debug.Printf(ctx, "cluster %s will be deleted on stop", f.ClusterName)
	}
	return f.waitForDefaultServiceAccount(ctx)
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

func (f *KindFixture) waitForDefaultServiceAccount(ctx context.Context) error {
	// Sometimes it takes a little while for the default service account to
	// exist for new clusters, and the default service account is required for
	// a lot of testing, so we wait here until the default service account is
	// ready to go...
	cfg, err := f.provider.KubeConfig(f.ClusterName, false)
	if err != nil {
		return err
	}
	cc, err := clientcmd.Load([]byte(cfg))
	if err != nil {
		return err
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, checkDefaultServiceAccountTimeout)
	defer cancel()
	overrides := &clientcmd.ConfigOverrides{}
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	ccfg, err := clientcmd.NewNonInteractiveClientConfig(
		*cc, "", overrides, rules,
	).ClientConfig()
	if err != nil {
		return err
	}
	clientset, err := kubernetes.NewForConfig(ccfg)
	if err != nil {
		return err
	}
	bo := backoff.WithContext(
		backoff.NewExponentialBackOff(),
		ctx,
	)
	ticker := backoff.NewTicker(bo)
	attempts := 1
	for range ticker.C {
		found := true
		_, err = clientset.CoreV1().ServiceAccounts("default").Get(context.TODO(), "default", metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
			found = false
		}
		debug.Printf(
			ctx, "check for default service account: attempt %d, found: %v",
			attempts, found,
		)
		attempts++
		if found {
			ticker.Stop()
			break
		}
	}
	return nil
}

func (f *KindFixture) Stop(ctx context.Context) {
	ctx = gdtcontext.PushTrace(ctx, "fixtures.kind.stop")
	defer func() {
		ctx = gdtcontext.PopTrace(ctx)
	}()
	if !f.isRunning() {
		debug.Printf(ctx, "cluster %s not running", f.ClusterName)
		return
	}
	if f.runningBeforeStart && !f.deleteOnStop {
		debug.Printf(ctx, "cluster %s was running before start and deleteOnStop=false so not deleting", f.ClusterName)
		return
	}
	if f.deleteOnStop {
		if err := f.provider.Delete(f.ClusterName, ""); err != nil {
			panic(err)
		}
		debug.Printf(ctx, "cluster %s successfully deleted", f.ClusterName)
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

func (f *KindFixture) State(key string) any {
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
func New(mods ...KindFixtureModifier) api.Fixture {
	f := &KindFixture{
		provider: cluster.NewProvider(),
	}
	for _, mod := range mods {
		mod(f)
	}
	return f
}
