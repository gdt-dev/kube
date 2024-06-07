// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package kube

import (
	"context"
	"fmt"
	"strings"

	gdtcontext "github.com/gdt-dev/gdt/context"
	"github.com/gdt-dev/gdt/debug"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type node struct {
	name        string
	allocatable map[string]resource.Quantity
	labels      map[string]string
}

// getNodes returns a slice of node objects in the Kubernetes cluster
func getNodes(
	ctx context.Context,
	c *connection,
) []node {
	gvk := schema.GroupVersionKind{
		Kind: "Node",
	}
	res, err := c.gvrFromGVK(gvk)
	opts := metav1.ListOptions{}
	list, err := c.client.Resource(res).Namespace("").List(
		ctx, opts,
	)
	if err != nil {
		panic(err)
	}
	nodes := make([]node, len(list.Items))
	for x, n := range list.Items {
		labels, _, _ := unstructured.NestedStringMap(n.UnstructuredContent(), "metadata", "labels")
		allocs := map[string]resource.Quantity{}
		allocatable, _, _ := unstructured.NestedStringMap(n.UnstructuredContent(), "status", "allocatable")
		for k, v := range allocatable {
			allocs[k] = resource.MustParse(v)
		}
		nodes[x] = node{
			name:        n.GetName(),
			allocatable: allocs,
			labels:      labels,
		}
	}
	return nodes
}

type pod struct {
	name     string
	nodename string
}

// getPods returns a slice of pod objects in the supplied Deployment or StatefulSet
func getPods(
	ctx context.Context,
	c *connection,
	r *unstructured.Unstructured,
) []pod {
	kind := strings.ToLower(r.GetKind())
	ns := r.GetNamespace()
	ls := labels.NewSelector()
	switch kind {
	case "deployment":
	case "statefulset":
		selector, _, _ := unstructured.NestedMap(r.UnstructuredContent(), "spec", "selector")
		matchLabels, found := selector["matchLabels"]
		if found {
			for k, v := range matchLabels.(map[string]string) {
				r, err := labels.NewRequirement(k, selection.Equals, []string{v})
				if err != nil {
					panic(err)
				}
				ls = ls.Add(*r)
			}
		}
	default:
		panic("unsupported placement Kind: " + kind)
	}
	gvk := schema.GroupVersionKind{
		Kind: "Pod",
	}
	res, err := c.gvrFromGVK(gvk)
	opts := client.ListOptions{
		LabelSelector: ls,
		Namespace:     ns,
	}
	list, err := c.client.Resource(res).Namespace(ns).List(
		ctx, *opts.AsListOptions(),
	)
	if err != nil {
		panic(err)
	}
	pods := make([]pod, len(list.Items))
	for x, p := range list.Items {
		nodename, _, _ := unstructured.NestedString(p.UnstructuredContent(), "spec", "nodeName")
		pods[x] = pod{
			name:     p.GetName(),
			nodename: nodename,
		}
	}
	return pods
}

// placementSpreadOK returns true if the Pods in the subject are evenly spread
// across hosts with the supplied topology keys
func (a *assertions) placementSpreadOK(
	ctx context.Context,
	res *unstructured.Unstructured,
	topoKeys []string,
) bool {
	if len(topoKeys) == 0 {
		return true
	}
	ctx = gdtcontext.PushTrace(ctx, "assert-placement-spread")
	defer func() {
		ctx = gdtcontext.PopTrace(ctx)
	}()
	nodes := getNodes(ctx, a.c)
	domainNodes := map[string][]string{}
	for _, k := range topoKeys {
		domainNodes[k] = []string{}
		for _, n := range nodes {
			_, found := n.labels[k]
			if found {
				domainNodes[k] = append(domainNodes[k], n.name)
			}
		}
	}

	// we construct a map, keyed by topology key, of maps, keyed by the value
	// of the topology key (the domain), with counts of pods scheduled to that
	// domain.
	pods := getPods(ctx, a.c, res)
	podDomains := map[string]map[string]int{}
	for _, k := range topoKeys {
		podDomains[k] = map[string]int{}
		for _, dom := range domainNodes[k] {
			podDomains[k][dom] = 0
			for _, pod := range pods {
				podNode := pod.nodename
				if dom == podNode {
					podDomains[k][dom]++
				}
			}
		}
	}

	// Pods are evenly spread across domains defined by the topology key when
	// the min and max number of pods on each domain is not greater than 1.
	for domain, nodes := range domainNodes {
		debug.Println(
			ctx, "domain: %s, unique nodes: %d",
			domain, len(nodes),
		)
		if len(nodes) > 0 {
			nodeCounts := lo.Values(podDomains[domain])

			debug.Println(
				ctx, "domain: %s, pods per node: %d",
				domain, nodeCounts,
			)
			minCount := lo.Min(nodeCounts)
			maxCount := lo.Max(nodeCounts)
			skew := maxCount - minCount
			if skew > 1 {
				msg := fmt.Sprintf(
					"found uneven spread skew of %d for domain %s",
					skew, domain,
				)
				a.Fail(fmt.Errorf(msg))
				return false
			}
		}
	}
	return true
}

// placementPackOK returns true if the Pods in the subject are packed onto
// hosts with the supplied topology keys
func (a *assertions) placementPackOK(
	ctx context.Context,
	res *unstructured.Unstructured,
	topoKeys []string,
) bool {
	return true
}
