// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	gdtexec "github.com/gdt-dev/gdt/plugin/exec"
)

type FailAction struct {
	http *Action
	exec *gdtexec.Action
}

// On describes actions that can be taken upon certain conditions.
type On struct {
	// Fail contains one or more actions to take if any of a Spec's assertions
	// fail.
	//
	// Any output from the Fail action is output into the test's debug output
	// as well as any debug stream the gdt user set up with `gdt.WithDebug()`.
	// Output for get Fail actions will be in YAML format.
	//
	// You can use the `exec` plugin's Action or the `kube` plugin's Action.
	//
	// For example, if you wanted to grep a log file in the event of an error
	// `kube apply`ing a `manifests/nginx-pod.yaml` file you might do this:
	//
	// ```yaml
	// tests:
	//  - kube:
	//      apply: manifests/nginx-pod.yaml
	//    on:
	//      fail:
	//        exec: grep ERROR /var/log/myapp.log
	// ```
	//
	// The kube gdt plugin's `on.fail` field also lets you make Kubernetes API
	// calls in addition to the `exec` action. So, you might want to grab some
	// information about a Pod in the event of a failure, like so:
	//
	// No retries are done for actions that fetch information because no
	// assertions are checked for Fail actions. If a get Fail action returns no
	// records, a "not found" message is printed to the test's debug output.
	//
	// ```yaml
	// tests:
	//  - kube:
	//      apply: manifests/nginx-pod.yaml
	//    on:
	//      fail:
	//        get: pods/nginx
	// ```
	Fail *FailAction `yaml:"fail,omitempty"`
}
