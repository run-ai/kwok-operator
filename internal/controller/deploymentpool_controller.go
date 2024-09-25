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

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kwoksigsv1beta1 "github.com/run-ai/kwok-operator/api/v1beta1"
)

// DeploymentPoolReconciler reconciles a DeploymentPool object
type DeploymentPoolReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=deploymentpools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=deploymentpools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=deploymentpools/finalizers,verbs=update

func (r *DeploymentPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling DeploymentPool")
	deploymentPool := &kwoksigsv1beta1.DeploymentPool{}
	err := r.Get(ctx, req.NamespacedName, deploymentPool)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("DeploymentPool resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch DeploymentPool")
		return ctrl.Result{}, err
	}
	log.Info("DeploymentPool resource found")

	if deploymentPool.Status.Conditions == nil || len(deploymentPool.Status.Conditions) == 0 {
		err = r.statusConditionController(ctx, deploymentPool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionUnknown,
			Reason:  "DeploymentPoolCreated",
			Message: "Starting to reconcile DeploymentPool",
		})
		if err != nil {
			log.Error(err, "unable to update DeploymentPool status")
			return ctrl.Result{}, err
		}
		err = r.Get(ctx, req.NamespacedName, deploymentPool)
		if err != nil {
			log.Error(err, "unable to fetch DeploymentPool")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}
	// add finalizer to the deployment pool
	if !controllerutil.ContainsFinalizer(deploymentPool, controllerFinalizer) {
		log.Info("Adding Finalizer for the DeploymentPool")
		err = r.addFinalizer(ctx, deploymentPool)
		if err != nil {
			log.Error(err, "unable to add Finalizer for the DeploymentPool")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Get Deployment in the cluster with owner reference to the DeploymentPool
	deployments, err := r.getDeployments(ctx, deploymentPool)
	if err != nil {
		//log.Error(err, "unable to get Deployment", deploymentPool)
		return ctrl.Result{}, err
	}
	// Create Deployment if it does not exist
	if len(deployments) == 0 {
		//log.Info("Creating %v Deployments", deploymentPool.Spec.DeploymentCount)
		for i := 0; i < int(deploymentPool.Spec.DeploymentCount); i++ {
			err = r.createDeployment(ctx, deploymentPool)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		err = r.updateObservedGeneration(ctx, deploymentPool)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	// update DeploymentPool status to condition true
	err = r.statusConditionController(ctx, deploymentPool, metav1.Condition{
		Type:    "Available",
		Status:  metav1.ConditionTrue,
		Reason:  "DeploymentPoolReconciled",
		Message: "DeploymentPool reconciled successfully",
	})
	if err != nil {
		log.Error(err, "unable to update DeploymentPool status")
		return ctrl.Result{Requeue: true}, nil
	}
	// update status of the deployment pool

	if deploymentPool.Status.ObservedGeneration != deploymentPool.Generation {
		log.Info("DeploymentPool generation has changed, requeuing")
		err = r.Get(ctx, req.NamespacedName, deploymentPool)
		if err != nil {
			log.Error(err, "unable to fetch DeploymentPool")
			return ctrl.Result{}, err
		}
		err = r.statusConditionController(ctx, deploymentPool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionFalse,
			Reason:  "DeploymentPoolReconciling",
			Message: "Updating DeploymentPool",
		})
		log.Info("Updating DeploymentPool")
		if err != nil {
			log.Error(err, "unable to update DeploymentPool status")
			return ctrl.Result{}, err
		}
		forceRequeue := false
		forceRequeue, err = r.updateDeployment(ctx, deploymentPool)
		if err != nil {
			log.Error(err, "unable to update Deployment")
			return ctrl.Result{}, err
		}
		if forceRequeue {
			println("Requeueing the deployment")
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, nil
	}
	if !deploymentPool.DeletionTimestamp.IsZero() {
		log.Info("Deleting DeploymentPool")
		err = r.statusConditionController(ctx, deploymentPool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionFalse,
			Reason:  "Deleting",
			Message: "Deleting the deploymentPool",
		})
		if err != nil {
			log.Error(err, "unable to update DeploymentPool status")
			return ctrl.Result{}, nil
		}
		for _, deployment := range deployments {
			err = r.Delete(ctx, &deployment)
			if err != nil {
				log.Error(err, "unable to delete Deployment")
				return ctrl.Result{}, nil
			}
		}
		err = r.deleteFinalizer(ctx, deploymentPool)
		if err != nil {
			log.Error(err, "unable to delete Finalizer")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, nil
	}
	log.Info("Reconciliation completed successfully")
	return ctrl.Result{RequeueAfter: time.Duration(60 * time.Second)}, nil
}

func (r *DeploymentPoolReconciler) statusConditionController(ctx context.Context, deploymentPool *kwoksigsv1beta1.DeploymentPool, condition metav1.Condition) error {
	meta.SetStatusCondition(&deploymentPool.Status.Conditions, condition)
	return r.Status().Update(ctx, deploymentPool)
}

func (r *DeploymentPoolReconciler) addFinalizer(ctx context.Context, deploymentPool *kwoksigsv1beta1.DeploymentPool) error {
	controllerutil.AddFinalizer(deploymentPool, controllerFinalizer)
	return r.Update(ctx, deploymentPool)
}

func (r *DeploymentPoolReconciler) deleteFinalizer(ctx context.Context, deploymentPool *kwoksigsv1beta1.DeploymentPool) error {
	controllerutil.RemoveFinalizer(deploymentPool, controllerFinalizer)
	return r.Update(ctx, deploymentPool)
}

func (r *DeploymentPoolReconciler) getDeployments(ctx context.Context, deploymentPool *kwoksigsv1beta1.DeploymentPool) ([]appsv1.Deployment, error) {

	deployment := &appsv1.DeploymentList{}
	err := r.List(ctx, deployment, client.InNamespace(deploymentPool.Namespace), client.MatchingLabels{controllerLabel: deploymentPool.Name})
	if err != nil && errors.IsNotFound(err) {
		return []appsv1.Deployment{}, nil
	} else if err != nil {
		return nil, err
	}
	return deployment.Items, nil
}

// update deployment
func (r *DeploymentPoolReconciler) updateDeployment(ctx context.Context, deploymentPool *kwoksigsv1beta1.DeploymentPool) (bool, error) {
	// get the deployment spec from the cluster
	forceRequeue := false
	deployments, err := r.getDeployments(ctx, deploymentPool)
	if err != nil {
		return forceRequeue, err
	}
	if len(deployments) < int(deploymentPool.Spec.DeploymentCount) {
		for i := int32(len(deployments)); i < deploymentPool.Spec.DeploymentCount; i++ {
			err = r.createDeployment(ctx, deploymentPool)
			if err != nil {
				log.Log.Error(err, "unable to create Deployment")
				return forceRequeue, err
			}
		}
		forceRequeue = true
		return forceRequeue, nil
	} else if len(deployments) > int(deploymentPool.Spec.DeploymentCount) {
		for i := int32(len(deployments)); i > deploymentPool.Spec.DeploymentCount; i-- {
			err = r.Delete(ctx, &deployments[i-1])
			if err != nil {
				log.Log.Error(err, "unable to delete Deployment")
				return forceRequeue, err
			}
		}
		forceRequeue = true
		return forceRequeue, nil
	} else {
		for i := 0; i < len(deployments); i++ {
			deployment := &deployments[i]
			deployment.Spec.Replicas = deploymentPool.Spec.DeploymentTemplate.Spec.Replicas
			deployment.Spec.Template.Spec.Containers = deploymentPool.Spec.DeploymentTemplate.Spec.Template.Spec.Containers
			err = r.Update(ctx, deployment)
			if err != nil {
				log.Log.Error(err, "unable to update Deployment")
				return forceRequeue, err
			}
		}
	}
	err = r.updateObservedGeneration(ctx, deploymentPool)
	if err != nil {
		log.Log.Error(err, "unable to update DeploymentPool")
		return forceRequeue, err
	}
	return forceRequeue, nil
}

// create deployment
func (r *DeploymentPoolReconciler) createDeployment(ctx context.Context, deploymentPool *kwoksigsv1beta1.DeploymentPool) error {
	appendSelector := deploymentPool.Spec.DeploymentTemplate.Spec.Selector
	appendSelector.MatchLabels[controllerLabel] = deploymentPool.Name

	overrideLabels := deploymentPool.Spec.DeploymentTemplate.Spec.Template.Labels
	if overrideLabels == nil {
		overrideLabels = map[string]string{
			controllerLabel: deploymentPool.Name,
		}
	} else {
		overrideLabels[controllerLabel] = deploymentPool.Name
	}

	deploymentToleration := deploymentPool.Spec.DeploymentTemplate.Spec.Template.Spec.Tolerations
	if deploymentToleration == nil {
		deploymentToleration = make([]corev1.Toleration, 0)
	}

	deploymentToleration = append(deploymentToleration, corev1.Toleration{
		Key:      controllerAnnotation,
		Operator: corev1.TolerationOpExists,
		Effect:   corev1.TaintEffectNoSchedule,
	})

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(deploymentPool, kwoksigsv1beta1.GroupVersion.WithKind("DeploymentPool")),
			},
			GenerateName: deploymentPool.Name + "-",
			Namespace:    deploymentPool.Namespace,
			Labels:       overrideLabels,
		},
		Spec: deploymentPool.Spec.DeploymentTemplate.Spec,
	}
	deployment.Spec.Template.ObjectMeta.Labels = overrideLabels
	deployment.Spec.Selector = appendSelector
	deployment.Spec.Template.Spec.Tolerations = deploymentToleration

	err := r.Create(ctx, deployment)
	if err != nil {
		return err
	}
	return nil
}

// updateObservedGeneration updates the observed generation of the NodePool
func (r *DeploymentPoolReconciler) updateObservedGeneration(ctx context.Context, deploymentPool *kwoksigsv1beta1.DeploymentPool) error {
	deploymentPool.Status.ObservedGeneration = deploymentPool.Generation
	return r.Status().Update(ctx, deploymentPool)
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kwoksigsv1beta1.DeploymentPool{}).
		Complete(r)
}
