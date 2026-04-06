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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/run-ai/kwok-operator/api/v1beta1"
)

func TestReconcilePoolChurnerCreatesPods(t *testing.T) {
	ctx := context.Background()
	sch := setupScheme()
	cl := fake.NewClientBuilder().WithScheme(sch).WithStatusSubresource(&v1beta1.PoolChurner{}).Build()
	pc := &v1beta1.PoolChurner{
		ObjectMeta: metav1.ObjectMeta{Name: "churn", Namespace: "default"},
		Spec: v1beta1.PoolChurnerSpec{
			PodCount:        2,
			IntervalSeconds: 3600,
			ChurnCount:      1,
			PodTemplate: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "c", Image: "pause"}},
				},
			},
		},
	}
	if err := cl.Create(ctx, pc); err != nil {
		t.Fatal(err)
	}
	r := &PoolChurnerReconciler{Client: cl, Scheme: sch}
	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: "churn", Namespace: "default"}}

	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatalf("reconcile 1: %v", err)
	}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatalf("reconcile 2: %v", err)
	}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatalf("reconcile 3: %v", err)
	}

	pods := &corev1.PodList{}
	if err := cl.List(ctx, pods, client.InNamespace("default")); err != nil {
		t.Fatal(err)
	}
	if len(pods.Items) != 2 {
		t.Fatalf("expected 2 pods, got %d", len(pods.Items))
	}
	for _, p := range pods.Items {
		if p.Labels[poolChurnerLabel] != "churn" {
			t.Fatalf("pod %q missing pool churner label", p.Name)
		}
	}
}
