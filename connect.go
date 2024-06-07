// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"context"
	"fmt"
	"os"

	gdtcontext "github.com/gdt-dev/gdt/context"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	discocached "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// Config returns a Kubernetes client-go rest.Config to use for this Spec. We
// evaluate where to retrieve the Kubernetes config from by looking at the
// following things, in this order:
//
// 1) The Spec.Kube.Config value
// 2) Any Fixtures that return a `kube.config` or `kube.config.bytes` state key
// 3) The Defaults.Config value
// 4) KUBECONFIG environment variable pointing at a file.
// 5) In-cluster config if running in cluster.
// 6) $HOME/.kube/config if exists.
func (s *Spec) Config(ctx context.Context) (*rest.Config, error) {
	d := fromBaseDefaults(s.Defaults)
	fixtures := gdtcontext.Fixtures(ctx)
	kctx := ""
	fixkctx := ""
	kcfgPath := ""
	fixkcfgPath := ""
	fixkcfgBytes := []byte{}

	for _, f := range fixtures {
		if f.HasState(StateKeyConfigBytes) {
			cfgBytesUntyped := f.State(StateKeyConfigBytes)
			fixkcfgBytes = cfgBytesUntyped.([]byte)
		}
		if f.HasState(StateKeyConfig) {
			cfgUntyped := f.State(StateKeyConfig)
			fixkcfgPath = cfgUntyped.(string)
		}
		if f.HasState(StateKeyContext) {
			ctxUntyped := f.State(StateKeyContext)
			fixkctx = ctxUntyped.(string)
		}
	}
	if s.Kube.Config != "" {
		kcfgPath = s.Kube.Config
	} else if fixkcfgPath != "" {
		kcfgPath = fixkcfgPath
	} else if d != nil && d.Config != "" {
		kcfgPath = d.Config
	}
	if s.Kube.Context != "" {
		kctx = s.Kube.Context
	} else if fixkctx != "" {
		kctx = fixkctx
	} else if d != nil && d.Context != "" {
		kctx = d.Context
	}
	overrides := &clientcmd.ConfigOverrides{}
	if kctx != "" {
		overrides.CurrentContext = kctx
	}
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kcfgPath != "" {
		rules.ExplicitPath = kcfgPath
	}
	if len(fixkcfgBytes) > 0 {
		cc, err := clientcmd.Load(fixkcfgBytes)
		if err != nil {
			return nil, err
		}
		return clientcmd.NewNonInteractiveClientConfig(
			*cc, "", overrides, rules,
		).ClientConfig()
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		rules, overrides,
	).ClientConfig()
}

// connection is a struct containing a discovery client and a dynamic client
// that the Spec uses to communicate with Kubernetes.
type connection struct {
	mapper meta.RESTMapper
	disco  discovery.CachedDiscoveryInterface
	client dynamic.Interface
}

// mappingFor returns a RESTMapper for a given resource type or kind
func (c *connection) mappingFor(typeOrKind string) (*meta.RESTMapping, error) {
	fullySpecifiedGVR, groupResource := schema.ParseResourceArg(typeOrKind)
	gvk := schema.GroupVersionKind{}

	if fullySpecifiedGVR != nil {
		gvk, _ = c.mapper.KindFor(*fullySpecifiedGVR)
	}
	if gvk.Empty() {
		gvk, _ = c.mapper.KindFor(groupResource.WithVersion(""))
	}
	if !gvk.Empty() {
		return c.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	}

	fullySpecifiedGVK, groupKind := schema.ParseKindArg(typeOrKind)
	if fullySpecifiedGVK == nil {
		gvk := groupKind.WithVersion("")
		fullySpecifiedGVK = &gvk
	}

	if !fullySpecifiedGVK.Empty() {
		if mapping, err := c.mapper.RESTMapping(fullySpecifiedGVK.GroupKind(), fullySpecifiedGVK.Version); err == nil {
			return mapping, nil
		}
	}

	mapping, err := c.mapper.RESTMapping(groupKind, gvk.Version)
	if err != nil {
		// if we error out here, it is because we could not match a resource or a kind
		// for the given argument. To maintain consistency with previous behavior,
		// announce that a resource type could not be found.
		// if the error is _not_ a *meta.NoKindMatchError, then we had trouble doing discovery,
		// so we should return the original error since it may help a user diagnose what is actually wrong
		if meta.IsNoMatchError(err) {
			return nil, fmt.Errorf("the server doesn't have a resource type %q", groupResource.Resource)
		}
		return nil, err
	}

	return mapping, nil
}

// gvrFromGVK returns a GroupVersionResource from a GroupVersionKind, using the
// discovery client to look up the resource name (the plural of the kind). The
// returned GroupVersionResource will have the proper Group and Version filled
// in (as opposed to an APIResource which has empty Group and Version strings
// because it "inherits" its APIResourceList's GroupVersion ... ugh.)
func (c *connection) gvrFromGVK(
	gvk schema.GroupVersionKind,
) (schema.GroupVersionResource, error) {
	empty := schema.GroupVersionResource{}
	r, err := c.mappingFor(gvk.Kind)
	if err != nil {
		return empty, ResourceUnknown(gvk)
	}

	return r.Resource, nil
}

// resourceNamespaces returns true if the supplied schema.GroupVersionResource
// is namespaced, false otherwise
func (c *connection) resourceNamespaced(gvr schema.GroupVersionResource) bool {
	apiResources, err := c.disco.ServerResourcesForGroupVersion(
		gvr.GroupVersion().String(),
	)
	if err != nil {
		panic("expected to find APIResource for GroupVersion " + gvr.GroupVersion().String())
	}
	for _, apiResource := range apiResources.APIResources {
		if apiResource.Name == gvr.Resource {
			return apiResource.Namespaced
		}
	}
	panic("expected to find APIResource for GroupVersionResource " + gvr.Resource)
}

// connect returns a connection with a discovery client and a Kubernetes
// client-go DynamicClient to use in communicating with the Kubernetes API
// server configured for this Spec
func (s *Spec) connect(ctx context.Context) (*connection, error) {
	cfg, err := s.Config(ctx)
	if err != nil {
		return nil, err
	}
	c, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	discoverer, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}
	disco := discocached.NewMemCacheClient(discoverer)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(disco)
	expander := restmapper.NewShortcutExpander(mapper, disco, func(s string) { fmt.Fprint(os.Stderr, s) })

	return &connection{
		mapper: expander,
		disco:  disco,
		client: c,
	}, nil
}
