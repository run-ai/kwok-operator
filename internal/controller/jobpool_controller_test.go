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
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/run-ai/kwok-operator/api/v1beta1"
	kwoksigsv1beta1 "github.com/run-ai/kwok-operator/api/v1beta1"
)

const (
	jobPoolName = "default"
)

var _ = Describe("JobPool Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: jobPoolName,
		}
		jobpool := &kwoksigsv1beta1.JobPool{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind JobPool")
			err := k8sClient.Get(ctx, typeNamespacedName, jobpool)
			if err != nil && errors.IsNotFound(err) {
				resource := &kwoksigsv1beta1.JobPool{

					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: jobPoolName,
						Labels: map[string]string{
							controllerLabel: resourceName,
						},
					},
					Spec: v1beta1.JobPoolSpec{
						JobCount: 5,
						JobTemplate: batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										RestartPolicy: corev1.RestartPolicyNever,
										Containers: []corev1.Container{
											{
												Name:  "test-container",
												Image: "busybox",
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
			resource := &kwoksigsv1beta1.JobPool{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance JobPool")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &JobPoolReconciler{
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

func TestJobPoolController(t *testing.T) {
	// create a fake client to mock API calls
	fakeClient := fake.NewClientBuilder().WithScheme(setupScheme()).WithStatusSubresource(&v1beta1.JobPool{}).Build()
	// create jobpool
	jobpool := &kwoksigsv1beta1.JobPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-jobpool",
			Namespace: "default",
		},
		Spec: v1beta1.JobPoolSpec{
			JobCount: 3,
			JobTemplate: batchv1.Job{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test-container",
									Image: "busybox",
								},
							},
						},
					},
				},
			},
		},
	}
	// Create a Reconcile JobPool object with the scheme and fake client.
	r := &JobPoolReconciler{
		Client: fakeClient,
		Scheme: setupScheme(),
	}

	// create jobpool object for testing
	err := fakeClient.Create(context.Background(), jobpool)
	if err != nil {
		t.Fatalf("Failed to create jobpool: %v", err)
	}

	// reconcile an object to get back a result
	res, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-jobpool",
			Namespace: "default",
		},
	})
	if res != (reconcile.Result{}) {
		t.Fatalf("Reconcile did not return an empty result")
	}

	// reconcile an object to get back a result
	res, err = r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-jobpool",
			Namespace: "default",
		},
	})
	if res != (reconcile.Result{}) {
		t.Fatalf("Reconcile did not return an empty result")
	}
	// get latest jobpool object
	err = fakeClient.Get(context.Background(), types.NamespacedName{
		Name:      "test-jobpool",
		Namespace: "default",
	}, jobpool)
	if err != nil {
		t.Fatalf("Failed to get jobpool: %v", err)
	}
	// scale jobpool jobCount to 5 and update the jobpool object in the fake client
	jobpool.Spec.JobCount = 5
	err = fakeClient.Update(context.Background(), jobpool)
	if err != nil {
		t.Fatalf("Failed to update jobpool: %v", err)
	}
	// reconcile an object to get back a result
	res, err = r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-jobpool",
			Namespace: "default",
		},
	})
	if res != (reconcile.Result{}) {
		t.Fatalf("Reconcile did not return an empty result")
	}
	// delete the jobpool object in the fake client
	err = fakeClient.Delete(context.Background(), jobpool)
	if err != nil {
		t.Fatalf("Failed to delete jobpool: %v", err)
	}
	// reconcile an object to get back a result
	res, err = r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-jobpool",
			Namespace: "default",
		},
	})
	if res != (reconcile.Result{}) {
		t.Fatalf("Reconcile did not return an empty result")
	}
}
