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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kwoksigsv1beta1 "github.com/run-ai/kwok-operator/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("DaemonsetPool Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		daemonsetpool := &kwoksigsv1beta1.DaemonsetPool{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind DaemonsetPool")
			err := k8sClient.Get(ctx, typeNamespacedName, daemonsetpool)
			if err != nil && errors.IsNotFound(err) {
				resource := &kwoksigsv1beta1.DaemonsetPool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: kwoksigsv1beta1.DaemonsetPoolSpec{
						DaemonsetTemplate: appsv1.DaemonSet{
							ObjectMeta: metav1.ObjectMeta{
								Name:      resourceName,
								Namespace: "default",
							},
							Spec: appsv1.DaemonSetSpec{
								Selector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"app": resourceName,
									},
								},
								Template: corev1.PodTemplateSpec{
									ObjectMeta: metav1.ObjectMeta{
										Labels: map[string]string{
											"app": resourceName,
										},
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
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &kwoksigsv1beta1.DaemonsetPool{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance DaemonsetPool")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &DaemonsetPoolReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

func TestReconcileDaemonsetPool(t *testing.T) {
	//create a fake client to mock API calls
	fakeClient := fake.NewClientBuilder().WithScheme(setupScheme()).WithStatusSubresource(&kwoksigsv1beta1.DaemonsetPool{}).Build()

	// Create a ReconcileDaemonsetPool object with the scheme and fake client.
	r := &DaemonsetPoolReconciler{
		Client: fakeClient,
		Scheme: setupScheme(),
	}

	// Create a DaemonsetPool object with the scheme and fake client.
	daemonsetpool := &kwoksigsv1beta1.DaemonsetPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-daemonsetpool",
			Namespace: "default",
		},
		Spec: kwoksigsv1beta1.DaemonsetPoolSpec{
			DaemonsetTemplate: appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-daemonset",
					Namespace: "default",
				},
				Spec: appsv1.DaemonSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "test-daemonset",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": "test-daemonset",
							},
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
			},
		},
	}
	err := fakeClient.Create(ctx, daemonsetpool)
	if err != nil {
		t.Fatalf("create DaemonsetPool: (%v)", err)
	}
	// Reconcile an object to get back a result
	res, err := r.Reconcile(ctx, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-daemonsetpool",
			Namespace: "default",
		},
	})
	if res != (reconcile.Result{}) {
		t.Fatalf("Reconcile did not return an empty result")
	}
	// Check to make sure the reconcile was successful and that it should requeue the request.
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	// get latest DaemonsetPool object
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-daemonsetpool",
		Namespace: "default",
	}, daemonsetpool)
	if err != nil {
		t.Fatalf("get DaemonsetPool: (%v)", err)
	}
	// change image of container
	daemonsetpool.Spec.DaemonsetTemplate.Spec.Template.Spec.Containers[0].Image = "busybox"
	err = fakeClient.Update(ctx, daemonsetpool)
	if err != nil {
		t.Fatalf("update DaemonsetPool: (%v)", err)
	}
	// Reconcile an object to get back a result
	res, err = r.Reconcile(ctx, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-daemonsetpool",
			Namespace: "default",
		},
	})
	if res != (reconcile.Result{}) {
		t.Fatalf("Reconcile did not return an empty result")
	}
	// Check to make sure the reconcile was successful and that it should requeue the request.
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	// get latest DaemonsetPool object
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-daemonsetpool",
		Namespace: "default",
	}, daemonsetpool)
	if err != nil {
		t.Fatalf("get DaemonsetPool: (%v)", err)
	}
	//delete DaemonsetPool object
	err = fakeClient.Delete(ctx, daemonsetpool)
	if err != nil {
		t.Fatalf("delete DaemonsetPool: (%v)", err)
	}
	// Reconcile an object to get back a result
	res, err = r.Reconcile(ctx, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-daemonsetpool",
			Namespace: "default",
		},
	})
	// Check to make sure the reconcile was successful and that it should requeue the request.
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	// validate the daemosetpool deleted
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-daemonsetpool",
		Namespace: "default",
	}, daemonsetpool)

	// list DaemonsetPool and check if the DaemonsetPool has been created
	daemonsetpoolList := &kwoksigsv1beta1.DaemonsetPoolList{
		Items: []kwoksigsv1beta1.DaemonsetPool{},
	}
	err = fakeClient.List(ctx, daemonsetpoolList)
	if err != nil {
		t.Fatalf("list DaemonsetPool: (%v)", err)
	}

}
