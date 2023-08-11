// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

// With houses one or more selectors that the Get and Delete fields may use to
// select the resources to operate against.
type With struct {
	// Labels is a map, keyed by metadata Label, of Label values to select a
	// resource by
	Labels map[string]string `yaml:"labels,omitempty"`
}

// Action describes the the Kubernetes-specific action that is performed by the
// test.
type Action struct {
	// Create is a string containing a file path or raw YAML content describing
	// a Kubernetes resource to call `kubectl create` with.
	Create string `yaml:"create,omitempty"`
	// Apply is a string containing a file path or raw YAML content describing
	// a Kubernetes resource to call `kubectl apply` with.
	Apply string `yaml:"apply,omitempty"`
	// Delete is a string containing an argument to `kubectl delete` and must
	// be one of the following:
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
	Delete string `yaml:"delete,omitempty"`
	// Get is a string containing an argument to `kubectl get` and must be one
	// of the following:
	//
	// - a file path to a manifest that will be read and the resources within
	//   retrieved via `kubectl get`
	// - a resource kind or kind alias, e.g. "pods", "po", followed by one of
	//   the following:
	//   * a space or `/` character followed by the resource name to get only a
	//     resource with that name.
	//   * a space followed by `-l ` followed by a label to get resources
	//     having such a label.
	Get string `yaml:"get,omitempty"`
	// With houses one or more selectors that the Get and Delete fields may use
	// to select the resources to operate against.
	//
	// Use in conjunction with Get and Delete to filter resources:
	//
	// ```yaml
	// tests:
	//  - name: delete pods with app:nginx label
	//    kube:
	//      delete: pods
	//      with:
	//        labels:
	//          app: nginx
	// ```
	With *With `yaml:"with,omitempty"`
}
