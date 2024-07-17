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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	//"k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kwoksigsv1beta1 "github.com/run-ai/kwok-operator/api/v1beta1"
)

// DaemonsetPoolReconciler reconciles a DaemonsetPool object
type DaemonsetPoolReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=daemonsetpools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=daemonsetpools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=daemonsetpools/finalizers,verbs=update

func (r *DaemonsetPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling DaemonsetPool")
	daemonsetPool := &kwoksigsv1beta1.DaemonsetPool{}
	err := r.Get(ctx, req.NamespacedName, daemonsetPool)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("DaemonsetPool resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch DaemonsetPool")
		return ctrl.Result{}, err
	}
	log.Info("DaemonsetPool resource found")
	if daemonsetPool.Status.Conditions == nil || len(daemonsetPool.Status.Conditions) == 0 {
		err = r.statusConditionController(ctx, daemonsetPool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionUnknown,
			Reason:  "DaemonsetPoolCreated",
			Message: "Stating to reconcile DaemonsetPool",
		})
		if err != nil {
			log.Error(err, "unable to update DaemonsetPool status")
			return ctrl.Result{}, err
		}
		err = r.Get(ctx, req.NamespacedName, daemonsetPool)
		if err != nil {
			log.Error(err, "unable to fetch DaemonsetPool")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	// add finalizer to the daemonset pool
	if !controllerutil.ContainsFinalizer(daemonsetPool, controllerFinalizer) {
		log.Info("Adding Finalizer for the DaemonsetPool")
		err = r.addFinalizer(ctx, daemonsetPool)
		if err != nil {
			log.Error(err, "unable to update DaemonsetPool")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	// Get Daemonset in the cluster with owner reference of the DaemonsetPool
	daemonset, err := r.getDaemonset(ctx, daemonsetPool)
	if err != nil {
		log.Error(err, "unable to get Daemonset")
		return ctrl.Result{}, err
	}
	if daemonset == nil {
		log.Info("Daemonset resource not found. Creating a new one")
		err = r.createDaemonset(ctx, daemonsetPool)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	err = r.statusConditionController(ctx, daemonsetPool, metav1.Condition{
		Type:    "Available",
		Status:  metav1.ConditionTrue,
		Reason:  "DaemonsetPoolReconciled",
		Message: "DaemonsetPool reconciled successfully",
	})
	if err != nil {
		log.Error(err, "unable to update DaemonsetPool status")
		return ctrl.Result{}, err
	}
	// update Daemonset if the DaemonsetPool is updated
	if daemonsetPool.Status.ObservedGeneration != daemonsetPool.Generation {
		log.Info("DaemonsetPool updated. Updating Daemonset")
		err = r.Get(ctx, req.NamespacedName, daemonsetPool)
		if err != nil {
			log.Error(err, "unable to fetch DaemonsetPool")
			return ctrl.Result{}, err
		}
		err = r.statusConditionController(ctx, daemonsetPool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionFalse,
			Reason:  "DaemonsetPoolPoolReconciling",
			Message: "Updating DaemonsetPool",
		})
		if err != nil {
			log.Error(err, "unable to update DaemonsetPool status")
			return ctrl.Result{}, err
		}
		log.Info("updating Daemonset")
		err = r.updateDaemonset(ctx, daemonsetPool)
		if err != nil {
			log.Error(err, "unable to update Daemonset")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, nil
	}
	if !daemonsetPool.DeletionTimestamp.IsZero() {
		log.Info("DaemonsetPool is being deleted. Deleting Daemonset")
		err = r.statusConditionController(ctx, daemonsetPool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionFalse,
			Reason:  "Deleting",
			Message: "Deleting the daemonsetPool",
		})
		if err != nil {
			log.Error(err, "unable to update DaemonsetPool status")
			return ctrl.Result{}, err
		}
		err = r.Delete(ctx, daemonset)
		if err != nil {
			log.Error(err, "unable to delete Daemonset")
			return ctrl.Result{}, err
		}
		err = r.deleteFinalizer(ctx, daemonsetPool)
		if err != nil {
			log.Error(err, "unable to delete Finalizer")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, nil
	}
	log.Info("Reconciliation completed successfully")
	return ctrl.Result{RequeueAfter: time.Duration(60 * time.Second)}, nil
}

// delete finalizer from the DaemonsetPool
func (r *DaemonsetPoolReconciler) deleteFinalizer(ctx context.Context, daemonsetPool *kwoksigsv1beta1.DaemonsetPool) error {
	controllerutil.RemoveFinalizer(daemonsetPool, controllerFinalizer)
	return r.Update(ctx, daemonsetPool)
}

// update the Daemonset in the cluster with owner reference to the DaemonsetPool
func (r *DaemonsetPoolReconciler) updateDaemonset(ctx context.Context, daemonsetPool *kwoksigsv1beta1.DaemonsetPool) error {
	// get the Daemonset spec from the cluster
	daemonset, err := r.getDaemonset(ctx, daemonsetPool)
	log.Log.Info("Updating daemonset", "daemonset", daemonset)
	if err != nil {
		return err
	}
	daemonset.Spec.Template.Spec.Containers = daemonsetPool.Spec.DaemonsetTemplate.Spec.Template.Spec.Containers
	err = r.Update(ctx, daemonset)
	if err != nil {
		return err
	}
	err = r.updateObservedGeneration(ctx, daemonsetPool)
	if err != nil {
		return err
	}
	return nil
}

// create the Daemonset in the cluster with owner reference to the DaemonsetPool
func (r *DaemonsetPoolReconciler) createDaemonset(ctx context.Context, daemonsetPool *kwoksigsv1beta1.DaemonsetPool) error {
	daemonsetToleration := daemonsetPool.Spec.DaemonsetTemplate.Spec.Template.Spec.Tolerations
	if daemonsetToleration == nil {
		daemonsetToleration = make([]corev1.Toleration, 0)
	}
	daemonsetToleration = append(daemonsetToleration, corev1.Toleration{
		Key:      controllerAnnotation,
		Operator: corev1.TolerationOpExists,
		Effect:   corev1.TaintEffectNoSchedule,
	})
	daemonset := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(daemonsetPool, kwoksigsv1beta1.GroupVersion.WithKind("DaemonsetPool")),
			},
			Name:      daemonsetPool.Name,
			Namespace: daemonsetPool.Namespace,
		},
		Spec: daemonsetPool.Spec.DaemonsetTemplate.Spec,
	}
	daemonset.Spec.Template.ObjectMeta.Labels = map[string]string{
		controllerLabel: daemonsetPool.Name,
	}
	daemonset.Spec.Template.Spec.Tolerations = daemonsetToleration
	daemonset.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			controllerLabel: daemonsetPool.Name,
		},
	}
	err := r.Create(ctx, daemonset)
	if err != nil {
		return err
	}
	err = r.updateObservedGeneration(ctx, daemonsetPool)
	if err != nil {
		return err
	}
	return nil
}

// update the observed generation of the DaemonsetPool
func (r *DaemonsetPoolReconciler) updateObservedGeneration(ctx context.Context, daemonsetPool *kwoksigsv1beta1.DaemonsetPool) error {
	daemonsetPool.Status.ObservedGeneration = daemonsetPool.Generation
	return r.Status().Update(ctx, daemonsetPool)
}

// get the Daemonset in the cluster with owner reference to the DaemonsetPool
func (r *DaemonsetPoolReconciler) getDaemonset(ctx context.Context, daemonsetPool *kwoksigsv1beta1.DaemonsetPool) (*appsv1.DaemonSet, error) {
	daemonset := &appsv1.DaemonSet{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: daemonsetPool.Namespace,
		Name:      daemonsetPool.Name,
	}, daemonset)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return daemonset, nil
}

// adding finalizer to the DaemonsetPool
func (r *DaemonsetPoolReconciler) addFinalizer(ctx context.Context, daemonsetPool *kwoksigsv1beta1.DaemonsetPool) error {
	controllerutil.AddFinalizer(daemonsetPool, controllerFinalizer)
	return r.Update(ctx, daemonsetPool)
}

// update the status of the DaemonsetPool
func (r *DaemonsetPoolReconciler) statusConditionController(ctx context.Context, daemonsetPool *kwoksigsv1beta1.DaemonsetPool, condition metav1.Condition) error {
	meta.SetStatusCondition(&daemonsetPool.Status.Conditions, condition)
	return r.Status().Update(ctx, daemonsetPool)
}

// SetupWithManager sets up the controller with the Manager.
func (r *DaemonsetPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kwoksigsv1beta1.DaemonsetPool{}).
		Complete(r)
}
