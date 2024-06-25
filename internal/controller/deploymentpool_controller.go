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
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	//"k8s.io/client-go/tools/clientcmd/api"
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

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DeploymentPool object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *DeploymentPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
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
			log.Error(err, "unable to update DeploymentPool")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Get Deployment in the cluster with owner reference to the DeploymentPool
	deployment, err := r.getDeployment(ctx, deploymentPool)
	log.Info("Deployment is here", "Deployment", deployment)
	if err != nil {
		log.Info("Deployment not found", "DeploymentPool", deploymentPool)
		log.Error(err, "unable to fetch Deployment name", "DeploymentPool", deployment)
		return ctrl.Result{}, err
	}
	// Create Deployment if it does not exist
	if deployment == nil {
		log.Info("Creating Deployment", "DeploymentPool", deploymentPool)
		err = r.createDeployment(ctx, deploymentPool)
		if err != nil {
			log.Error(err, "unable to create Deployment", "DeploymentPool", deploymentPool)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
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

func (r *DeploymentPoolReconciler) getDeployment(ctx context.Context, deploymentPool *kwoksigsv1beta1.DeploymentPool) (*appsv1.Deployment, error) {
	deployment := &appsv1.DeploymentList{}
	err := r.List(ctx, deployment, client.InNamespace(deploymentPool.Namespace), client.MatchingLabels{controllerLabel: deploymentPool.Name})
	if err != nil && strings.Contains(err.Error(), "does not exist") {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	if len(deployment.Items) == 0 {
		return nil, nil
	}
	return &deployment.Items[0], nil
}

// create deployment
func (r *DeploymentPoolReconciler) createDeployment(ctx context.Context, deploymentPool *kwoksigsv1beta1.DeploymentPool) error {
	deploymentToleration := deploymentPool.Spec.DeploymentTemplate.Spec.Template.Spec.Tolerations
	if deploymentToleration == nil {
		log.Log.Info("Pod tolerations is nil")
		deploymentToleration = make([]corev1.Toleration, 0)
	}
	deploymentToleration = append(deploymentToleration, corev1.Toleration{
		Key:      controllerAnnotation,
		Value:    fakeString,
		Operator: corev1.TolerationOpExists,
		Effect:   corev1.TaintEffectNoSchedule,
	})
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: deploymentPool.Name + "-",
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(deploymentPool, kwoksigsv1beta1.GroupVersion.WithKind("DeploymentPool")),
			},
			Name:      deploymentPool.Name,
			Namespace: deploymentPool.Namespace,
		},
		Spec: deploymentPool.Spec.DeploymentTemplate.Spec,
	}
	deployment.Spec.Template.Spec.Tolerations = deploymentToleration

	log.Log.Info("present entire object", "DeploymentPool", deploymentPool)
	err := r.Create(ctx, deployment)
	if err != nil {
		return err
	}
	err = r.updateObservedGeneration(ctx, deploymentPool)
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
