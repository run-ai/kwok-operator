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
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/run-ai/kwok-operator/api/v1beta1"
	kwoksigsv1beta1 "github.com/run-ai/kwok-operator/api/v1beta1"
)

var _ = Describe("PodPool Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		podpool := &kwoksigsv1beta1.PodPool{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind PodPool")
			err := k8sClient.Get(ctx, typeNamespacedName, podpool)
			if err != nil && errors.IsNotFound(err) {
				resource := &kwoksigsv1beta1.PodPool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: v1beta1.PodPoolSpec{
						PodCount: 1,
						PodTemplate: corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-pod",
							},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "test-container",
										Image: "nginx",
									},
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &kwoksigsv1beta1.PodPool{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance PodPool")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &PodPoolReconciler{
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

//test for the controller of PodPool resource provision and deletion with fake client

func TestReconcilepodPool(t *testing.T) {
	// Create a fake client
	fakeClient := fake.NewClientBuilder().WithScheme(setupScheme()).WithStatusSubresource(&v1beta1.PodPool{}).Build()
	// Create a NodePool object for testing
	podPool := &v1beta1.PodPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-podpool",
			Namespace: "default",
		},
		Spec: v1beta1.PodPoolSpec{
			PodCount: 1,
			PodTemplate: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "nginx",
						},
					},
				},
			},
		},
	}
	// Create a ReconcilepodPool object with the scheme and fake client.
	r := &PodPoolReconciler{
		Client: fakeClient,
		Scheme: setupScheme(),
	}
	// Create the PodPool object in the fake client.
	err := fakeClient.Create(ctx, podPool)
	if err != nil {
		t.Fatalf("create PodPool: (%v)", err)
	}
	// Reconcile an object to get back a result.
	res, err := r.Reconcile(ctx, reconcile.Request{
		NamespacedName: typeNamespacedName,
	})
	if res != (reconcile.Result{}) {
		t.Fatalf("reconcile did not return an empty result")
	}
	// Check to make sure the reconcile was successful and that it should requeue the request.
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	//update podPool podCount to 5 and update the podPool object in the fake client
	podPool.Spec.PodCount = 5
	err = fakeClient.Update(ctx, podPool)
	if err != nil {
		t.Fatalf("update PodPool: (%v)", err)
	}
	// Reconcile an object to get back a result.
	res, err = r.Reconcile(ctx, reconcile.Request{
		NamespacedName: typeNamespacedName,
	})
	if res != (reconcile.Result{}) {
		t.Fatalf("reconcile did not return an empty result")
	}
	// Check to make sure the reconcile was successful and that it should requeue the request.
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	//delete the podPool object in the fake client
	err = fakeClient.Delete(ctx, podPool)
	if err != nil {
		t.Fatalf("delete PodPool: (%v)", err)
	}
	// Reconcile an object to get back a result.
	res, err = r.Reconcile(ctx, reconcile.Request{
		NamespacedName: typeNamespacedName,
	})
	if res != (reconcile.Result{}) {
		t.Fatalf("reconcile did not return an empty result")
	}
	// Check to make sure the reconcile was successful and that it should requeue the request.
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

}
