// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"github.com/gdt-dev/core/api"
	gdtplugin "github.com/gdt-dev/core/plugin"
	"gopkg.in/yaml.v3"
)

var (
	// DefaultTimeout is the default timeout used for each individual test
	// spec. Note that gdt's top-level Scenario.Run handles all timeout and
	// retry behaviour.
	DefaultTimeout = "5s"
)

func init() {
	gdtplugin.Register(Plugin())
}

const (
	pluginName = "kube"
)

type plugin struct{}

func (p *plugin) Info() api.PluginInfo {
	return api.PluginInfo{
		Name: pluginName,
		Retry: &api.Retry{
			Exponential: true,
		},
		Timeout: &api.Timeout{
			After: DefaultTimeout,
		},
	}
}

func (p *plugin) Defaults() yaml.Unmarshaler {
	return &Defaults{}
}

func (p *plugin) Specs() []api.Evaluable {
	return []api.Evaluable{&Spec{}}
}

// Plugin returns the Kubernetes gdt plugin
func Plugin() api.Plugin {
	return &plugin{}
}
