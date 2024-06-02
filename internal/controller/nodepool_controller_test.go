/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	kwoksigsv1beta1 "github.com/run-ai/kwok-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/stretchr/testify/assert"

	"github.com/run-ai/kwok-operator/api/v1beta1"
)

var _ = Describe("NodePool Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name: resourceName,
		}
		nodepool := &kwoksigsv1beta1.NodePool{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind NodePool")
			err := k8sClient.Get(ctx, typeNamespacedName, nodepool)
			if err != nil && errors.IsNotFound(err) {
				resource := &kwoksigsv1beta1.NodePool{
					ObjectMeta: metav1.ObjectMeta{
						Name: resourceName,
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &kwoksigsv1beta1.NodePool{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance NodePool")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &NodePoolReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})

func TestReconcileNodePool(t *testing.T) {
	// Create a fake client
	fakeClient := fake.NewClientBuilder().WithScheme(setupScheme()).WithStatusSubresource(&v1beta1.NodePool{}).Build()
	// Create a NodePool object for testing
	nodePool := &v1beta1.NodePool{
		ObjectMeta: metav1.ObjectMeta{Name: "single-nodepool"},
		Spec: v1beta1.NodePoolSpec{
			NodeCount: 5, // Set the desired number of nodes
			NodeTemplate: corev1.Node{
				Spec: corev1.NodeSpec{
					// Set node spec fields as needed for testing
				},
			},
		},
		Status: v1beta1.NodePoolStatus{},
	}

	// Create a Reconciler instance
	reconciler := &NodePoolReconciler{
		Client: fakeClient,
		Scheme: setupScheme(),
	}

	// Create a context
	ctx := context.Background()

	// Create the NodePool object in the fake client
	err := fakeClient.Create(ctx, nodePool)
	assert.NoError(t, err, "failed to create NodePool object")

	// Reconcile the NodePool
	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: "single-nodepool"}}
	_, err = reconciler.Reconcile(ctx, req)
	assert.NoError(t, err, "reconciliation failed")

	// Verify that the number of nodes matches the desired count
	nodes := &corev1.NodeList{}
	err = fakeClient.List(ctx, nodes)
	assert.NoError(t, err, "failed to list nodes")
	assert.Equal(t, int(nodePool.Spec.NodeCount), len(nodes.Items), "unexpected number of nodes")

	// Update the NodePool object to have a single node

	err = fakeClient.Get(ctx, types.NamespacedName{Name: "single-nodepool"}, nodePool)
	nodePool.Spec.NodeCount = 2
	err = fakeClient.Update(ctx, nodePool)
	req = reconcile.Request{NamespacedName: types.NamespacedName{Name: "single-nodepool"}}
	_, err = reconciler.Reconcile(ctx, req)
	assert.NoError(t, err, "reconciliation failed")
	err = fakeClient.List(ctx, nodes)
	assert.NoError(t, err, "failed to list nodes")
	assert.Equal(t, int(nodePool.Spec.NodeCount), len(nodes.Items), "expected 1 got %d", len(nodes.Items))

	// delete the NodePool object and check if the nodes are deleted
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "single-nodepool"}, nodePool)
	err = fakeClient.Delete(ctx, nodePool)
	assert.NoError(t, err, "failed to delete NodePool object")
	// Reconcile the NodePool
	req = reconcile.Request{NamespacedName: types.NamespacedName{Name: "single-nodepool"}}
	_, err = reconciler.Reconcile(ctx, req)
	assert.Error(t, err, "single-nodepool not found")
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "single-nodepool"}, nodePool)
	assert.Error(t, err, "single-nodepool not found")

	// validate that the nodes are deleted
	//fakeClient.List(ctx, nodes)
	//assert.Equal(t, 0, len(nodes.Items), "unexpected number of nodes")
}

func TestMultipleNodePools(t *testing.T) {
	// Create a fake client
	fakeClient := fake.NewClientBuilder().WithScheme(setupScheme()).WithStatusSubresource(&v1beta1.NodePool{}).Build()

	// Initial node count for first NodePool
	initialNodeCount1 := int32(2)

	// Create the first NodePool object for testing
	nodePool1 := &v1beta1.NodePool{
		ObjectMeta: metav1.ObjectMeta{Name: "test-nodepool-1"},
		Spec: v1beta1.NodePoolSpec{
			NodeCount: initialNodeCount1,
			NodeTemplate: corev1.Node{
				Spec: corev1.NodeSpec{
					// Set node spec fields as needed for testing
				},
			},
		},
	}

	// Initial node count for second NodePool
	initialNodeCount2 := int32(3)

	// Create the second NodePool object for testing
	nodePool2 := &v1beta1.NodePool{
		ObjectMeta: metav1.ObjectMeta{Name: "test-nodepool-2"},
		Spec: v1beta1.NodePoolSpec{
			NodeCount: initialNodeCount2,
			NodeTemplate: corev1.Node{
				Spec: corev1.NodeSpec{
					// Set node spec fields as needed for testing
				},
			},
		},
	}

	// Create a Reconciler instance
	reconciler := &NodePoolReconciler{
		Client: fakeClient,
		Scheme: setupScheme(),
	}

	// Create a context
	ctx := context.Background()

	// Create the first NodePool object in the fake client
	err := fakeClient.Create(ctx, nodePool1)
	assert.NoError(t, err, "failed to create NodePool 1")

	// Create the second NodePool object in the fake client
	err = fakeClient.Create(ctx, nodePool2)
	assert.NoError(t, err, "failed to create NodePool 2")

	// Reconcile the first NodePool
	req1 := reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-nodepool-1"}}
	_, err = reconciler.Reconcile(ctx, req1)
	assert.NoError(t, err, "reconciliation for NodePool 1 failed")

	// Reconcile the second NodePool
	req2 := reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-nodepool-2"}}
	_, err = reconciler.Reconcile(ctx, req2)
	assert.NoError(t, err, "reconciliation for NodePool 2 failed")

	// Verify the desired node count for each NodePool
	assertNodeCount(t, fakeClient, "test-nodepool-1", initialNodeCount1)
	assertNodeCount(t, fakeClient, "test-nodepool-2", initialNodeCount2)
}

func assertNodeCount(t *testing.T, c client.Client, nodeName string, expectedNodeCount int32) {
	nodes := &corev1.NodeList{}
	err := c.List(context.Background(), nodes)
	assert.NoError(t, err, "failed to list nodes")
	count := 0
	for _, node := range nodes.Items {
		if node.Labels[nodePoolControllerLabel] == nodeName {
			count++
		}
	}
	assert.Equal(t, int(expectedNodeCount), count, "unexpected node count for NodePool: "+nodeName)
}

// setupScheme sets up the scheme for the tests
func setupScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = v1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	return scheme
}
