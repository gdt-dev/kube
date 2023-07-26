// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

const (
	// StateKeyConfig holds a file path to a kubeconfig
	StateKeyConfig = "kube.config"
	// StateKeyConfigBytes holds a KUBECONFIG object in a bytearray
	StateKeyConfigBytes = "kube.config.bytes"
	// StateKeyContext holds a string kubecontext name
	StateKeyContext = "kube.context"
)
