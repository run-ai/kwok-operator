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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kwoksigsv1beta1 "github.com/run-ai/kwok-operator/api/v1beta1"
)

const (
	deploymentpoolName      = "test-deploymentpool"
	deploymentpoolNamespace = "default"
	deploymentName          = "test-deployment"
)

var (
	typeNamespacedName = types.NamespacedName{
		Name:      deploymentpoolName,
		Namespace: deploymentpoolNamespace, // TODO(user):Modify as needed
	}
	ctx = context.Background()
)

var _ = Describe("DeploymentPool Controller", func() {
	Context("When reconciling a resource", func() {

		deploymentpool := &kwoksigsv1beta1.DeploymentPool{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind DeploymentPool")
			err := k8sClient.Get(ctx, typeNamespacedName, deploymentpool)
			if err != nil && errors.IsNotFound(err) {
				resource := &kwoksigsv1beta1.DeploymentPool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      deploymentpoolName,
						Namespace: deploymentpoolNamespace,
					},
					Spec: kwoksigsv1beta1.DeploymentPoolSpec{
						DeploymentTemplate: appsv1.Deployment{
							ObjectMeta: metav1.ObjectMeta{
								Name:      deploymentName,
								Namespace: deploymentpoolNamespace,
							},
							Spec: appsv1.DeploymentSpec{
								Replicas: func() *int32 { i := int32(1); return &i }(),
								Selector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"app": deploymentName,
									},
								},
								Template: corev1.PodTemplateSpec{
									ObjectMeta: metav1.ObjectMeta{
										Labels: map[string]string{
											"app": deploymentName,
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
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &kwoksigsv1beta1.DeploymentPool{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance DeploymentPool")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &DeploymentPoolReconciler{
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

func TestReconcileDeploymentPool(t *testing.T) {
	// Create a fake client with the scheme and status subresource with all v1 objects.
	fakeClient := fake.NewClientBuilder().WithScheme(setupScheme()).WithStatusSubresource(&kwoksigsv1beta1.DeploymentPool{}).Build()

	// Create a ReconcileDeploymentPool object with the scheme and fake client.
	r := &DeploymentPoolReconciler{
		Client: fakeClient,
		Scheme: setupScheme(),
	}
	// Create a DeploymentPool object and Deployment object to use in test
	dp := &kwoksigsv1beta1.DeploymentPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentpoolName,
			Namespace: deploymentpoolNamespace,
		},
		Spec: kwoksigsv1beta1.DeploymentPoolSpec{
			DeploymentTemplate: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      deploymentName,
					Namespace: deploymentpoolNamespace,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: func() *int32 { i := int32(1); return &i }(),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": deploymentName,
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": deploymentName,
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
	// Create the DeploymentPool object in the fake client.
	err := fakeClient.Create(ctx, dp)
	if err != nil {
		t.Fatalf("create DeploymentPool: (%v)", err)
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
	// list deployment and check if the deployment has been created
	depList := &appsv1.DeploymentList{}
	err = fakeClient.List(ctx, depList)
	if err != nil {
		t.Fatalf("list Deployment: (%v)", err)
	}
	// Check to make sure the DeploymentPool has been reconciled.
	err = fakeClient.Get(ctx, typeNamespacedName, dp)
	if err != nil {
		t.Fatalf("get DeploymentPool: (%v)", err)
	}
	if dp.Status.ObservedGeneration != dp.Generation {
		t.Fatalf("observedGeneration not updated")
	}
	// update replicas to 8
	replicas := int32(8)
	dp.Spec.DeploymentTemplate.Spec.Replicas = &replicas
	err = fakeClient.Update(ctx, dp)
	if err != nil {
		t.Fatalf("update DeploymentPool: (%v)", err)
	}
	// Reconcile the updated object.
	res, err = r.Reconcile(ctx, reconcile.Request{
		NamespacedName: typeNamespacedName,
	})
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	// delete deploymentPool and check if the deployment has been deleted
	err = fakeClient.Delete(ctx, dp)
	if err != nil {
		t.Fatalf("delete DeploymentPool: (%v)", err)
	}
	// Reconcile the deleted object.
	res, err = r.Reconcile(ctx, reconcile.Request{
		NamespacedName: typeNamespacedName,
	})
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	// Check to make sure the DeploymentPool has been deleted.
	dpList := &kwoksigsv1beta1.DeploymentPoolList{}
	err = fakeClient.List(ctx, dpList)
	if err != nil {
		t.Fatalf("list DeploymentPool: (%v)", err)
	}
	err = fakeClient.List(ctx, depList)
	if err != nil {
		t.Fatalf("list Deployment: (%v)", err)
	}
}
