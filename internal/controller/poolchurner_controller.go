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
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kwoksigsv1beta1 "github.com/run-ai/kwok-operator/api/v1beta1"
)

// PoolChurnerReconciler reconciles a PoolChurner object.
type PoolChurnerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=poolchurners,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=poolchurners/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=poolchurners/finalizers,verbs=update

func poolChurnerKey(pc *kwoksigsv1beta1.PoolChurner) client.ObjectKey {
	return client.ObjectKey{Namespace: pc.Namespace, Name: pc.Name}
}

// updatePoolChurnerStatus loads the latest PoolChurner and writes status (retries on RV conflict).
func (r *PoolChurnerReconciler) updatePoolChurnerStatus(ctx context.Context, key client.ObjectKey, mutate func(*kwoksigsv1beta1.PoolChurner)) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		pc := &kwoksigsv1beta1.PoolChurner{}
		if err := r.Get(ctx, key, pc); err != nil {
			return err
		}
		mutate(pc)
		return r.Status().Update(ctx, pc)
	})
}

func (r *PoolChurnerReconciler) ensureFinalizer(ctx context.Context, key client.ObjectKey) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		pc := &kwoksigsv1beta1.PoolChurner{}
		if err := r.Get(ctx, key, pc); err != nil {
			return err
		}
		if controllerutil.ContainsFinalizer(pc, controllerFinalizer) {
			return nil
		}
		controllerutil.AddFinalizer(pc, controllerFinalizer)
		return r.Update(ctx, pc)
	})
}

func (r *PoolChurnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	pc := &kwoksigsv1beta1.PoolChurner{}
	if err := r.Get(ctx, req.NamespacedName, pc); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	wt := pc.Spec.WorkloadType
	if wt == "" {
		wt = kwoksigsv1beta1.PoolChurnerWorkloadPod
	}
	if wt != kwoksigsv1beta1.PoolChurnerWorkloadPod {
		logger.Info("unsupported workload type for PoolChurner", "workloadType", wt)
		_ = r.setCondition(ctx, req.NamespacedName, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionFalse,
			Reason:  "UnsupportedWorkload",
			Message: "Only workloadType Pod is supported",
		})
		return ctrl.Result{}, nil
	}

	if !pc.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, pc)
	}

	if !controllerutil.ContainsFinalizer(pc, controllerFinalizer) {
		if err := r.ensureFinalizer(ctx, req.NamespacedName); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if pc.Status.Conditions == nil {
		if err := r.setCondition(ctx, req.NamespacedName, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionUnknown,
			Reason:  "Reconciling",
			Message: "Starting PoolChurner reconciliation",
		}); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Get(ctx, req.NamespacedName, pc); err != nil {
			return ctrl.Result{}, err
		}
	}

	pods, err := r.listOwnedPods(ctx, pc)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Any spec change bumps Generation; delete managed Pods and reset churn timers.
	// ObservedGeneration is advanced in createPods once count matches PodCount again.
	if pc.Status.ObservedGeneration != pc.Generation {
		for i := range pods {
			if err := r.Delete(ctx, &pods[i]); err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		}
		if err := r.updatePoolChurnerStatus(ctx, req.NamespacedName, func(latest *kwoksigsv1beta1.PoolChurner) {
			latest.Status.LastChurnTime = nil
			latest.Status.ChurnCycles = 0
		}); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Get(ctx, req.NamespacedName, pc); err != nil {
			return ctrl.Result{}, err
		}
		pods, err = r.listOwnedPods(ctx, pc)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	n := int32(len(pods))
	if n < pc.Spec.PodCount {
		if err := r.createPods(ctx, pc, pods); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.setCondition(ctx, req.NamespacedName, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "Scaling",
			Message: "Creating Pods toward podCount",
		}); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	if n > pc.Spec.PodCount {
		sort.Slice(pods, func(i, j int) bool { return pods[i].Name < pods[j].Name })
		for i := n - 1; i >= pc.Spec.PodCount; i-- {
			if err := r.Delete(ctx, &pods[i]); err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// PodCount 0: nothing to create; still advance ObservedGeneration after a spec bump.
	if pc.Spec.PodCount == 0 && pc.Status.ObservedGeneration != pc.Generation {
		if err := r.updatePoolChurnerStatus(ctx, req.NamespacedName, func(latest *kwoksigsv1beta1.PoolChurner) {
			latest.Status.ObservedGeneration = latest.Generation
		}); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: DefaultIdleRequeue}, nil
	}

	churnOn := pc.Spec.IntervalSeconds > 0 && pc.Spec.ChurnCount > 0
	if !churnOn {
		if err := r.setCondition(ctx, req.NamespacedName, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "Steady",
			Message: "Churn disabled; PodCount maintained",
		}); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: DefaultIdleRequeue}, nil
	}

	now := time.Now()
	interval := time.Duration(pc.Spec.IntervalSeconds) * time.Second
	if err := r.Get(ctx, req.NamespacedName, pc); err != nil {
		return ctrl.Result{}, err
	}
	if pc.Status.LastChurnTime == nil {
		if err := r.updatePoolChurnerStatus(ctx, req.NamespacedName, func(latest *kwoksigsv1beta1.PoolChurner) {
			latest.Status.LastChurnTime = &metav1.Time{Time: now}
			meta.SetStatusCondition(&latest.Status.Conditions, metav1.Condition{
				Type:    "Available",
				Status:  metav1.ConditionTrue,
				Reason:  "ChurnScheduled",
				Message: "Waiting for first churn interval",
			})
		}); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: interval}, nil
	}

	elapsed := now.Sub(pc.Status.LastChurnTime.Time)
	if elapsed < interval {
		return ctrl.Result{RequeueAfter: interval - elapsed}, nil
	}

	sort.Slice(pods, func(i, j int) bool { return pods[i].Name < pods[j].Name })
	churn := int(pc.Spec.ChurnCount)
	if churn > len(pods) {
		churn = len(pods)
	}
	for i := 0; i < churn; i++ {
		if err := r.Delete(ctx, &pods[i]); err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
	}
	if err := r.updatePoolChurnerStatus(ctx, req.NamespacedName, func(latest *kwoksigsv1beta1.PoolChurner) {
		latest.Status.LastChurnTime = &metav1.Time{Time: now}
		latest.Status.ChurnCycles++
		meta.SetStatusCondition(&latest.Status.Conditions, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "Churned",
			Message: "Issued churn deletes; Pods will be recreated",
		})
	}); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true}, nil
}

func (r *PoolChurnerReconciler) reconcileDelete(ctx context.Context, pc *kwoksigsv1beta1.PoolChurner) (ctrl.Result, error) {
	pods, err := r.listOwnedPods(ctx, pc)
	if err != nil {
		return ctrl.Result{}, err
	}
	for i := range pods {
		if err := r.Delete(ctx, &pods[i]); err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
	}
	key := poolChurnerKey(pc)
	return ctrl.Result{}, retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &kwoksigsv1beta1.PoolChurner{}
		if err := r.Get(ctx, key, latest); err != nil {
			return err
		}
		controllerutil.RemoveFinalizer(latest, controllerFinalizer)
		return r.Update(ctx, latest)
	})
}

func (r *PoolChurnerReconciler) listOwnedPods(ctx context.Context, pc *kwoksigsv1beta1.PoolChurner) ([]corev1.Pod, error) {
	list := &corev1.PodList{}
	err := r.List(ctx, list, client.InNamespace(pc.Namespace), client.MatchingLabels{poolChurnerLabel: pc.Name})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *PoolChurnerReconciler) createPods(ctx context.Context, pc *kwoksigsv1beta1.PoolChurner, existing []corev1.Pod) error {
	podLabels := pc.Spec.PodTemplate.Labels
	if podLabels == nil {
		podLabels = make(map[string]string)
	}
	podLabels[poolChurnerLabel] = pc.Name
	podToleration := pc.Spec.PodTemplate.Spec.Tolerations
	if podToleration == nil {
		podToleration = make([]corev1.Toleration, 0)
	}
	podToleration = append(podToleration, corev1.Toleration{
		Key:      controllerAnnotation,
		Operator: corev1.TolerationOpExists,
		Effect:   corev1.TaintEffectNoSchedule,
	})
	podAnnotation := pc.Spec.PodTemplate.Annotations
	if podAnnotation == nil {
		podAnnotation = make(map[string]string)
	}
	podAnnotation[controllerAnnotation] = fakeString

	for i := int32(len(existing)); i < pc.Spec.PodCount; i++ {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: pc.Name + "-",
				Namespace:    pc.Namespace,
				Labels:       podLabels,
				Annotations:  podAnnotation,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(pc, kwoksigsv1beta1.GroupVersion.WithKind("PoolChurner")),
				},
			},
			Spec: pc.Spec.PodTemplate.Spec,
		}
		pod.Spec.Tolerations = podToleration
		if err := r.Create(ctx, pod); err != nil {
			return err
		}
	}
	return r.updatePoolChurnerStatus(ctx, poolChurnerKey(pc), func(latest *kwoksigsv1beta1.PoolChurner) {
		latest.Status.ObservedGeneration = latest.Generation
	})
}

func (r *PoolChurnerReconciler) setCondition(ctx context.Context, key client.ObjectKey, c metav1.Condition) error {
	return r.updatePoolChurnerStatus(ctx, key, func(pc *kwoksigsv1beta1.PoolChurner) {
		meta.SetStatusCondition(&pc.Status.Conditions, c)
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *PoolChurnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kwoksigsv1beta1.PoolChurner{}).
		Complete(r)
}
