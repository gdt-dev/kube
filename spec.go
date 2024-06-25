// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"path/filepath"
	"strings"

	"github.com/gdt-dev/gdt/api"
)

// KubeSpec is the complex type containing all of the Kubernetes-specific
// actions. Most users will use the `kube.create`, `kube.apply` and
// `kube.describe` shortcut fields.
type KubeSpec struct {
	Action
	// Config is the path of the kubeconfig to use in executing Kubernetes
	// client calls for this Spec. If empty, the `kube` defaults' `config`
	// value will be used. If that is empty, the following precedence is used:
	//
	// 1) KUBECONFIG environment variable pointing at a file.
	// 2) In-cluster config if running in cluster.
	// 3) $HOME/.kube/config if exists.
	Config string `yaml:"config,omitempty"`
	// Context is the name of the kubecontext to use for this Spec. If empty,
	// the `kube` defaults' `context` value will be used. If that is empty, the
	// kubecontext marked default in the kubeconfig is used.
	Context string `yaml:"context,omitempty"`
	// Namespace is a string indicating the Kubernetes namespace to use when
	// calling the Kubernetes API. If empty, any namespace specified in the
	// Defaults is used and then the string "default" is used.
	Namespace string `yaml:"namespace,omitempty"`
}

// Spec describes a test of a *single* Kubernetes API request and response.
type Spec struct {
	api.Spec
	// Kube is the complex type containing all of the Kubernetes-specific
	// actions and assertions. Most users will use the `kube.create`,
	// `kube.apply` and `kube.describe` shortcut fields.
	Kube *KubeSpec `yaml:"kube,omitempty"`
	// KubeCreate is a shortcut for the `KubeSpec.Create`. It can contain
	// either a file path or raw YAML content describing a Kubernetes resource
	// to call `kubectl create` with.
	KubeCreate string `yaml:"kube.create,omitempty"`
	// KubeGet is a string containing an argument to `kubectl get` and must be
	// one of the following:
	//
	// - a file path to a manifest that will be read and the resources within
	//   retrieved via `kubectl get`
	// - a resource kind or kind alias, e.g. "pods", "po", followed by one of
	//   the following:
	//   * a space or `/` character followed by the resource name to get only a
	//     resource with that name.
	//   * a space followed by `-l ` followed by a label to get resources
	//     having such a label.
	KubeGet string `yaml:"kube.get,omitempty"`
	// KubeApply is a shortcut for the `KubeSpec.Apply`. It is a string
	// containing a file path or raw YAML content describing a Kubernetes
	// resource to call `kubectl apply` with.
	KubeApply string `yaml:"kube.apply,omitempty"`
	// KubeDelete is a shortcut for the `KubeSpec.Delete`. It is a string
	// containing an argument to `kubectl delete` and must be one of the
	// following:
	//
	// - a file path to a manifest that will be read and the resources
	//   described in the manifest will be deleted
	// - a resource kind or kind alias, e.g. "pods", "po", followed by one of
	//   the following:
	//   * a space or `/` character followed by the resource name to delete
	//     only a resource with that name.
	//   * a space followed by `-l ` followed by a label to delete resources
	//     having such a label.
	//   * the string `--all` to delete all resources of that kind.
	KubeDelete string `yaml:"kube.delete,omitempty"`
	// Assert houses the various assertions to be made about the kube client
	// call (Create, Apply, Get, etc)
	// TODO(jaypipes): Make this polymorphic to be either a single assertion
	// struct or a list of assertion structs
	Assert *Expect `yaml:"assert,omitempty"`
}

func (s *Spec) Retry() *api.Retry {
	if s.Spec.Retry != nil {
		// The user may have overridden in the test spec file...
		return s.Spec.Retry
	}
	if s.Kube.Action.Get != nil {
		// returning nil here means the plugin's default will be used...
		return nil
	}
	// for apply/create/delete, we don't want to retry...
	return api.NoRetry
}

func (s *Spec) Timeout() *api.Timeout {
	// returning nil here means the plugin's default will be used...
	return nil
}

// Title returns a good name for the Spec
func (s *Spec) Title() string {
	// If the user did not specify a name for the test spec, just default
	// it to the method and URL
	if s.Name != "" {
		return s.Name
	}
	if s.Kube == nil {
		// Shouldn't happen because of parsing, but you never know...
		return ""
	}
	if s.Kube.Get != nil {
		return "kube.get:" + s.Kube.Get.Title()
	}
	if s.Kube.Create != "" {
		create := s.Kube.Create
		if probablyFilePath(create) {
			return "kube.create:" + filepath.Base(create)
		}
	}
	if s.Kube.Apply != "" {
		apply := s.Kube.Apply
		if probablyFilePath(apply) {
			return "kube.apply:" + filepath.Base(apply)
		}
	}
	if s.Kube.Delete != nil {
		return "kube.delete:" + s.Kube.Delete.Title()
	}
	return ""
}

// probablyFilePath returns true if the supplied string looks to be a file
// path, false otherwise
func probablyFilePath(subject string) bool {
	if strings.ContainsAny(subject, " :\n\r\t") {
		return false
	}
	return strings.ContainsRune(subject, '.')
}

func (s *Spec) SetBase(b api.Spec) {
	s.Spec = b
}

func (s *Spec) Base() *api.Spec {
	return &s.Spec
}

// Namespace returns the Kubernetes namespace to use when calling the
// Kubernetes API server. We evaluate which namespace to use by looking at the
// following things, in this order:
//
// 1) The Spec.Kube.Namespace value
// 2) The Defaults.Namespace value
// 3) Use the string "default"
func (s *Spec) Namespace() string {
	if s.Kube.Namespace != "" {
		return s.Kube.Namespace
	}
	d := fromBaseDefaults(s.Defaults)
	if d != nil && d.Namespace != "" {
		return d.Namespace
	}
	return "default"
}
