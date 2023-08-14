// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

// Action describes the the Kubernetes-specific action that is performed by the
// test.
type Action struct {
	// Create is a string containing a file path or raw YAML content describing
	// a Kubernetes resource to call `kubectl create` with.
	Create string `yaml:"create,omitempty"`
	// Apply is a string containing a file path or raw YAML content describing
	// a Kubernetes resource to call `kubectl apply` with.
	Apply string `yaml:"apply,omitempty"`
	// Delete is a string or object containing arguments to `kubectl delete`.
	//
	// It must be one of the following:
	//
	// - a file path to a manifest that will be read and the resources
	//   described in the manifest will be deleted
	// - a resource kind or kind alias, e.g. "pods", "po", followed by one of
	//   the following:
	//   * a space or `/` character followed by the resource name to delete
	//     only a resource with that name.
	// - an object with a `type` and optional `labels` field containing a label
	//   selector that should be used to select that `type` of resource.
	Delete *ResourceIdentifierOrFile `yaml:"delete,omitempty"`
	// Get is a string or object containing arguments to `kubectl get`.
	//
	// It must be one of the following:
	//
	// - a string with a resource kind or kind alias, e.g. "pods", "po",
	//   followed by one of the following:
	//   * a space or `/` character followed by the resource name to get only a
	//     resource with that name.
	// - an object with a `type` and optional `labels` field containing a label
	//   selector that should be used to select that `type` of resource.
	Get *ResourceIdentifier `yaml:"get,omitempty"`
}
