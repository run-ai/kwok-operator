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
	"fmt"
	"strings"

	kwoksigsv1beta1 "github.com/run-ai/kwok-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// kwokControllerOwnedByLabel is applied to namespace-scoped resources (ServiceAccount,
	// Deployment) created by KwokControllerReconciler. Owner references cannot cross
	// cluster/namespace scope boundaries, so we track ownership with this label instead.
	kwokControllerOwnedByLabel = "kwok.sigs.run-ai.com/owned-by-kwokcontroller"

	kwokImage              = "registry.k8s.io/kwok/kwok"
	kwokServiceAccountName = "kwok-controller"
	kwokClusterRoleName    = "kwok-operator-kwok-controller"
	kwokDeploymentName     = "kwok-controller"
)

// KwokControllerReconciler reconciles a KwokController object
type KwokControllerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=kwokcontrollers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=kwokcontrollers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=kwokcontrollers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kwok.x-k8s.io,resources="*",verbs=get;list;watch;create;update;patch;delete

func (r *KwokControllerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling KwokController")

	kwokCtrl := &kwoksigsv1beta1.KwokController{}
	if err := r.Get(ctx, req.NamespacedName, kwokCtrl); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("KwokController resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get KwokController")
		return ctrl.Result{}, err
	}

	// Init status condition on first reconcile
	if len(kwokCtrl.Status.Conditions) == 0 {
		if err := r.setCondition(ctx, kwokCtrl, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionUnknown,
			Reason:  "Reconciling",
			Message: "Starting to reconcile the KwokController",
		}); err != nil {
			log.Error(err, "Failed to update KwokController status")
			return ctrl.Result{}, err
		}
		if err := r.Get(ctx, req.NamespacedName, kwokCtrl); err != nil {
			log.Error(err, "Failed to re-fetch KwokController")
			return ctrl.Result{}, err
		}
	}

	// Add finalizer
	if !controllerutil.ContainsFinalizer(kwokCtrl, controllerFinalizer) {
		log.Info("Adding Finalizer for KwokController")
		controllerutil.AddFinalizer(kwokCtrl, controllerFinalizer)
		if err := r.Update(ctx, kwokCtrl); err != nil {
			log.Error(err, "Failed to add finalizer to KwokController")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Handle deletion
	if !kwokCtrl.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, kwokCtrl)
	}

	// Reconcile all child resources
	if err := r.reconcileResources(ctx, kwokCtrl); err != nil {
		log.Error(err, "Failed to reconcile KwokController resources")
		_ = r.setCondition(ctx, kwokCtrl, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionFalse,
			Reason:  "ReconcileError",
			Message: err.Error(),
		})
		return ctrl.Result{}, err
	}

	// Sync status from the live Deployment
	if err := r.syncStatus(ctx, kwokCtrl); err != nil {
		log.Error(err, "Failed to sync KwokController status")
		return ctrl.Result{}, err
	}

	log.Info("Reconciliation completed successfully")
	return ctrl.Result{RequeueAfter: DefaultIdleRequeue}, nil
}

// reconcileResources ensures the ServiceAccount, ClusterRole, ClusterRoleBinding,
// and Deployment exist and match the desired spec.
func (r *KwokControllerReconciler) reconcileResources(ctx context.Context, kwokCtrl *kwoksigsv1beta1.KwokController) error {
	if err := r.reconcileServiceAccount(ctx, kwokCtrl); err != nil {
		return fmt.Errorf("serviceaccount: %w", err)
	}
	if err := r.reconcileClusterRole(ctx, kwokCtrl); err != nil {
		return fmt.Errorf("clusterrole: %w", err)
	}
	if err := r.reconcileClusterRoleBinding(ctx, kwokCtrl); err != nil {
		return fmt.Errorf("clusterrolebinding: %w", err)
	}
	if err := r.reconcileDeployment(ctx, kwokCtrl); err != nil {
		return fmt.Errorf("deployment: %w", err)
	}
	return nil
}

func (r *KwokControllerReconciler) reconcileServiceAccount(ctx context.Context, kwokCtrl *kwoksigsv1beta1.KwokController) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kwokServiceAccountName,
			Namespace: kwokCtrl.Spec.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, sa, func() error {
		if sa.Labels == nil {
			sa.Labels = make(map[string]string)
		}
		sa.Labels[kwokControllerOwnedByLabel] = kwokCtrl.Name
		return nil
	})
	return err
}

func (r *KwokControllerReconciler) reconcileClusterRole(ctx context.Context, kwokCtrl *kwoksigsv1beta1.KwokController) error {
	cr := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: kwokClusterRoleName,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cr, func() error {
		// ClusterRole is cluster-scoped; owner reference is valid from cluster-scoped KwokController.
		if err := controllerutil.SetControllerReference(kwokCtrl, cr, r.Scheme); err != nil {
			return err
		}
		cr.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"nodes/status"},
				Verbs:     []string{"get", "patch", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods/status"},
				Verbs:     []string{"get", "patch", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"create", "patch", "update"},
			},
			{
				APIGroups: []string{"coordination.k8s.io"},
				Resources: []string{"leases"},
				Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch"},
			},
			{
				APIGroups: []string{"kwok.x-k8s.io"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
		}
		return nil
	})
	return err
}

func (r *KwokControllerReconciler) reconcileClusterRoleBinding(ctx context.Context, kwokCtrl *kwoksigsv1beta1.KwokController) error {
	crbName := kwokClusterRoleName
	existing := &rbacv1.ClusterRoleBinding{}
	err := r.Get(ctx, types.NamespacedName{Name: crbName}, existing)

	if apierrors.IsNotFound(err) {
		crb := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: crbName,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     kwokClusterRoleName,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      kwokServiceAccountName,
					Namespace: kwokCtrl.Spec.Namespace,
				},
			},
		}
		if err := controllerutil.SetControllerReference(kwokCtrl, crb, r.Scheme); err != nil {
			return err
		}
		return r.Create(ctx, crb)
	}
	if err != nil {
		return err
	}

	// ClusterRoleBinding.RoleRef is immutable; only update subjects if namespace changed.
	updated := existing.DeepCopy()
	updated.Subjects = []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      kwokServiceAccountName,
			Namespace: kwokCtrl.Spec.Namespace,
		},
	}
	if err := controllerutil.SetControllerReference(kwokCtrl, updated, r.Scheme); err != nil {
		return err
	}
	return r.Update(ctx, updated)
}

func (r *KwokControllerReconciler) reconcileDeployment(ctx context.Context, kwokCtrl *kwoksigsv1beta1.KwokController) error {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kwokDeploymentName,
			Namespace: kwokCtrl.Spec.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, dep, func() error {
		if dep.Labels == nil {
			dep.Labels = make(map[string]string)
		}
		dep.Labels[kwokControllerOwnedByLabel] = kwokCtrl.Name

		replicas := kwokCtrl.Spec.Replicas
		if replicas == nil {
			one := int32(1)
			replicas = &one
		}

		image := fmt.Sprintf("%s:%s", kwokImage, kwokCtrl.Spec.Version)
		args := r.buildContainerArgs(kwokCtrl)

		podLabels := map[string]string{
			"app":                      "kwok-controller",
			kwokControllerOwnedByLabel: kwokCtrl.Name,
		}

		dep.Spec = appsv1.DeploymentSpec{
			Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: podLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: kwokServiceAccountName,
					Containers: []corev1.Container{
						{
							Name:      "kwok-controller",
							Image:     image,
							Args:      args,
							Resources: kwokCtrl.Spec.Resources,
						},
					},
				},
			},
		}
		return nil
	})
	return err
}

// buildContainerArgs translates KwokControllerSpec fields into kwok controller CLI flags.
// Key memory-saving flags: --enable-crds limits informer cache scope.
func (r *KwokControllerReconciler) buildContainerArgs(kwokCtrl *kwoksigsv1beta1.KwokController) []string {
	args := []string{
		fmt.Sprintf("--manage-all-nodes=%v", kwokCtrl.Spec.ManageAllNodes),
		"--v=2",
	}

	if kwokCtrl.Spec.NodeAnnotationSelector != "" && !kwokCtrl.Spec.ManageAllNodes {
		args = append(args, fmt.Sprintf("--node-annotation-selector=%s", kwokCtrl.Spec.NodeAnnotationSelector))
	}

	if len(kwokCtrl.Spec.EnableCRDs) > 0 {
		args = append(args, fmt.Sprintf("--enable-crds=%s", strings.Join(kwokCtrl.Spec.EnableCRDs, ",")))
	}

	args = append(args, kwokCtrl.Spec.AdditionalFlags...)
	return args
}

// syncStatus reads the live Deployment and updates KwokController status.
func (r *KwokControllerReconciler) syncStatus(ctx context.Context, kwokCtrl *kwoksigsv1beta1.KwokController) error {
	dep := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      kwokDeploymentName,
		Namespace: kwokCtrl.Spec.Namespace,
	}, dep); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	kwokCtrl.Status.ReadyReplicas = dep.Status.ReadyReplicas
	kwokCtrl.Status.DeployedVersion = kwokCtrl.Spec.Version
	kwokCtrl.Status.ObservedGeneration = kwokCtrl.Generation

	desired := int32(1)
	if kwokCtrl.Spec.Replicas != nil {
		desired = *kwokCtrl.Spec.Replicas
	}

	condStatus := metav1.ConditionFalse
	condReason := "DeploymentNotReady"
	condMsg := fmt.Sprintf("Ready replicas: %d/%d", dep.Status.ReadyReplicas, desired)
	if dep.Status.ReadyReplicas >= desired {
		condStatus = metav1.ConditionTrue
		condReason = "Available"
		condMsg = fmt.Sprintf("KWOK controller %s is running with %d ready replica(s)", kwokCtrl.Spec.Version, dep.Status.ReadyReplicas)
	}

	apimeta.SetStatusCondition(&kwokCtrl.Status.Conditions, metav1.Condition{
		Type:    "Available",
		Status:  condStatus,
		Reason:  condReason,
		Message: condMsg,
	})
	return r.Status().Update(ctx, kwokCtrl)
}

// reconcileDelete cleans up all resources owned by this KwokController and removes the finalizer.
func (r *KwokControllerReconciler) reconcileDelete(ctx context.Context, kwokCtrl *kwoksigsv1beta1.KwokController) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Deleting KwokController resources")

	_ = r.setCondition(ctx, kwokCtrl, metav1.Condition{
		Type:    "Available",
		Status:  metav1.ConditionFalse,
		Reason:  "Deleting",
		Message: "Deleting the KwokController",
	})

	// Delete Deployment (namespace-scoped, tracked by label)
	depList := &appsv1.DeploymentList{}
	if err := r.List(ctx, depList,
		client.InNamespace(kwokCtrl.Spec.Namespace),
		client.MatchingLabels{kwokControllerOwnedByLabel: kwokCtrl.Name},
	); err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("list deployments: %w", err)
	}
	for i := range depList.Items {
		if err := r.Delete(ctx, &depList.Items[i]); err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("delete deployment: %w", err)
		}
	}

	// Delete ServiceAccount (namespace-scoped, tracked by label)
	saList := &corev1.ServiceAccountList{}
	if err := r.List(ctx, saList,
		client.InNamespace(kwokCtrl.Spec.Namespace),
		client.MatchingLabels{kwokControllerOwnedByLabel: kwokCtrl.Name},
	); err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("list serviceaccounts: %w", err)
	}
	for i := range saList.Items {
		if err := r.Delete(ctx, &saList.Items[i]); err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("delete serviceaccount: %w", err)
		}
	}

	// ClusterRoleBinding and ClusterRole are cluster-scoped with owner references;
	// Kubernetes GC will cascade-delete them. Explicit deletion here avoids GC latency.
	crb := &rbacv1.ClusterRoleBinding{}
	if err := r.Get(ctx, types.NamespacedName{Name: kwokClusterRoleName}, crb); err == nil {
		if err := r.Delete(ctx, crb); err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("delete clusterrolebinding: %w", err)
		}
	}

	cr := &rbacv1.ClusterRole{}
	if err := r.Get(ctx, types.NamespacedName{Name: kwokClusterRoleName}, cr); err == nil {
		if err := r.Delete(ctx, cr); err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("delete clusterrole: %w", err)
		}
	}

	controllerutil.RemoveFinalizer(kwokCtrl, controllerFinalizer)
	if err := r.Update(ctx, kwokCtrl); err != nil {
		return ctrl.Result{}, fmt.Errorf("remove finalizer: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *KwokControllerReconciler) setCondition(ctx context.Context, kwokCtrl *kwoksigsv1beta1.KwokController, condition metav1.Condition) error {
	apimeta.SetStatusCondition(&kwokCtrl.Status.Conditions, condition)
	return r.Status().Update(ctx, kwokCtrl)
}

// SetupWithManager sets up the controller with the Manager.
// ClusterRole and ClusterRoleBinding carry owner references so Owns() triggers re-reconcile on changes.
// Deployment and ServiceAccount are tracked by label; the DefaultIdleRequeue provides convergence.
func (r *KwokControllerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kwoksigsv1beta1.KwokController{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Complete(r)
}
