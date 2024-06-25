// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"os"

	"github.com/gdt-dev/gdt/api"
	"gopkg.in/yaml.v3"
)

type kubeDefaults struct {
	// Config is the path of the kubeconfig to use in executing Kubernetes
	// client calls. If empty, typical kubeconfig path-finding is used, meaning
	// that the following precedence is used:
	//
	// 1) KUBECONFIG environment variable pointing at a file.
	// 2) In-cluster config if running in cluster.
	// 3) $HOME/.kube/config if exists.
	//
	// This value can be overridden with the `Spec.Kube.Config` field.
	Config string `yaml:"config,omitempty"`
	// Context is the name of the kubecontext to use. If empty, the kubecontext
	// marked default in the kubeconfig is used. This can be overridden with
	// the `Spec.Kube.Context` field.
	Context string `yaml:"context,omitempty"`
	// Namespace is the name of the Kubernetes namespace to use by default.
	// This can be overridden with the `Spec.Kube.Namespace` field.
	Namespace string `yaml:"namespace,omitempty"`
}

// Defaults is the known HTTP plugin defaults collection
type Defaults struct {
	kubeDefaults
}

func (d *Defaults) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return api.ExpectedMapAt(node)
	}
	// maps/structs are stored in a top-level Node.Content field which is a
	// concatenated slice of Node pointers in pairs of key/values.
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode.Kind != yaml.ScalarNode {
			return api.ExpectedScalarAt(keyNode)
		}
		key := keyNode.Value
		valNode := node.Content[i+1]
		switch key {
		case "kube":
			if valNode.Kind != yaml.MappingNode {
				return api.ExpectedMapAt(valNode)
			}
			hd := kubeDefaults{}
			if err := valNode.Decode(&hd); err != nil {
				return err
			}
			d.kubeDefaults = hd
		default:
			continue
		}
	}
	return d.validate()
}

// validate determines if any specified defaults are valid.
func (d *Defaults) validate() error {
	if d.Config != "" {
		f, err := os.Open(d.Config)
		if err != nil {
			if os.IsNotExist(err) {
				return KubeConfigNotFound(d.Config)
			}
			return err
		}
		_, err = f.Stat()
		if err != nil {
			return err
		}
	}
	return nil
}

// fromBaseDefaults returns an gdt-kube plugin-specific Defaults from a Spec
func fromBaseDefaults(base *api.Defaults) *Defaults {
	if base == nil {
		return nil
	}
	d := base.For(pluginName)
	if d == nil {
		return nil
	}
	return d.(*Defaults)
}
