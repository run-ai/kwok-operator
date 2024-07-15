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
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kwoksigsv1beta1 "github.com/run-ai/kwok-operator/api/v1beta1"
)

// PodPoolReconciler reconciles a PodPool object
type PodPoolReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=podpools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=podpools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=podpools/finalizers,verbs=update

func (r *PodPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling PodPool")
	podPool := &kwoksigsv1beta1.PodPool{}
	err := r.Get(ctx, req.NamespacedName, podPool)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then, it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("podPool resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request CR.
		log.Error(err, "Failed to get podPool")
		return ctrl.Result{}, err
	}
	// Set reconciling status condition in the podPool status
	if podPool.Status.Conditions == nil || len(podPool.Status.Conditions) == 0 {
		err := r.statusConditionController(ctx, podPool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionUnknown,
			Reason:  "Reconciling",
			Message: "Starting to reconcile the PodPool",
		})
		if err != nil {
			log.Error(err, "Failed to update PodPool status")
			return ctrl.Result{}, err
		}
		err = r.Get(ctx, req.NamespacedName, podPool)
		if err != nil {
			log.Error(err, "Failed to get PodPool")
			return ctrl.Result{}, err
		}
	}
	// Set the finalizer for the podPool
	if !controllerutil.ContainsFinalizer(podPool, controllerFinalizer) {
		log.Info("Adding Finalizer for the PodPool")
		err := r.addFinalizer(ctx, podPool)
		if err != nil {
			log.Error(err, "Failed to add finalizer to PodPool")
			return ctrl.Result{}, err
		}
	}
	// Get pods in the cluster with owner reference to the podPool
	pods, err := r.getPods(ctx, podPool)
	if err != nil {
		log.Error(err, "Failed to get pods")
		return ctrl.Result{}, err
	}
	// Check if the number of pods in the cluster is equal to the desired number of pods in the podPool
	if int32(len(pods)) != podPool.Spec.PodCount {
		if int32(len(pods)) < podPool.Spec.PodCount {
			log.Info("Creating pods")
			err := r.statusConditionController(ctx, podPool, metav1.Condition{
				Type:    "Available",
				Status:  metav1.ConditionFalse,
				Reason:  "ScalingUp",
				Message: "Scalling up the podPool",
			})
			if err != nil {
				log.Error(err, "Failed to update podPool status")
				return ctrl.Result{}, err
			}
			log.Info("Scalling up the podPool... creating pods!")
			err = r.createPods(ctx, podPool, pods)
			if err != nil {
				log.Error(err, "Failed to create pods")
				return ctrl.Result{}, err
			}
		} else {
			log.Info("Deleting pods")
			err := r.statusConditionController(ctx, podPool, metav1.Condition{
				Type:    "Available",
				Status:  metav1.ConditionFalse,
				Reason:  "ScalingDown",
				Message: "Scalling down the podPool",
			})
			if err != nil {
				log.Error(err, "Failed to update podPool status")
				return ctrl.Result{}, err
			}
			log.Info("Scalling down the podPool... deleting pods!")
			err = r.deletePods(ctx, podPool, pods)
			if err != nil {
				log.Error(err, "Failed to delete pods")
				return ctrl.Result{}, err
			}
		}
	}
	// Update the status of the podPool
	err = r.statusConditionController(ctx, podPool, metav1.Condition{
		Type:    "Available",
		Status:  metav1.ConditionTrue,
		Reason:  "PodPoolReconciled",
		Message: "PodPool reconciled successfully",
	})
	if err != nil {
		log.Error(err, "Failed to update podPool status")
		return ctrl.Result{}, err
	}
	// Update the observed generation of the podPool
	if podPool.Status.ObservedGeneration != podPool.Generation {
		log.Info("podTemplate has changed")
		err := r.statusConditionController(ctx, podPool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionFalse,
			Reason:  "Updating",
			Message: "Updating the podPool",
		})
		if err != nil {
			log.Error(err, "Failed to update podPool status")
			return ctrl.Result{}, err
		}
		emptyPodPool := &kwoksigsv1beta1.PodPool{
			ObjectMeta: metav1.ObjectMeta{
				Name: podPool.Name,
			},
		}
		err = r.deletePods(ctx, emptyPodPool, pods)
		if err != nil {
			log.Error(err, "Failed to delete pods")
			return ctrl.Result{}, err
		}
		err = r.createPods(ctx, podPool, pods)
		if err != nil {
			log.Error(err, "Failed to create pods")
			return ctrl.Result{}, err
		}
		err = r.statusConditionController(ctx, podPool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "Ready",
			Message: "podPool is ready",
		})
		if err != nil {
			log.Error(err, "Failed to update PodPool status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if !podPool.DeletionTimestamp.IsZero() {
		log.Info("Deleting PodPool")
		err = r.statusConditionController(ctx, podPool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionFalse,
			Reason:  "Deleting",
			Message: "Deleting the podPool",
		})
		if err != nil {
			log.Error(err, "Failed to update podPool status")
			return ctrl.Result{}, nil
		}
		err := r.deletePods(ctx, podPool, pods)
		if err != nil {
			log.Error(err, "Failed to delete nodes")
			return ctrl.Result{}, err
		}
		err = r.deleteFinalizer(ctx, podPool)
		if err != nil {
			log.Error(err, "Failed to delete finalizer from PodPool")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	log.Info("Reconciliation finished")
	return ctrl.Result{RequeueAfter: time.Duration(60 * time.Second)}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kwoksigsv1beta1.PodPool{}).
		Complete(r)
}

// deleteFinalizer deletes the finalizer from the podPool
func (r *PodPoolReconciler) deleteFinalizer(ctx context.Context, podPool *kwoksigsv1beta1.PodPool) error {
	controllerutil.RemoveFinalizer(podPool, controllerFinalizer)
	return r.Update(ctx, podPool)
}

// statusConditionController updates the status of the podPool
func (r *PodPoolReconciler) statusConditionController(ctx context.Context, podPool *kwoksigsv1beta1.PodPool, condition metav1.Condition) error {
	meta.SetStatusCondition(&podPool.Status.Conditions, condition)
	return r.Status().Update(ctx, podPool)
}

// addFinalizer adds the finalizer to the podPool
func (r *PodPoolReconciler) addFinalizer(ctx context.Context, podPool *kwoksigsv1beta1.PodPool) error {
	controllerutil.AddFinalizer(podPool, controllerFinalizer)
	return r.Update(ctx, podPool)
}

// get pods in the cluster with owner reference to the podPool
func (r *PodPoolReconciler) getPods(ctx context.Context, podPool *kwoksigsv1beta1.PodPool) ([]corev1.Pod, error) {
	pods := &corev1.PodList{}
	err := r.List(ctx, pods, client.InNamespace(podPool.Namespace), client.MatchingLabels{controllerLabel: podPool.Name})
	if err != nil && strings.Contains(err.Error(), "does not exist") {
		return []corev1.Pod{}, nil
	} else if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

// deletePods deletes the pods in the cluster
func (r *PodPoolReconciler) deletePods(ctx context.Context, podPool *kwoksigsv1beta1.PodPool, pods []corev1.Pod) error {
	for i := int32(len(pods)); i > podPool.Spec.PodCount; i-- {
		// Delete a pod
		err := r.Delete(ctx, &pods[i-1])
		if err != nil {
			return err
		}
	}
	err := r.updateObservedGeneration(ctx, podPool)
	if err != nil {
		return err
	}
	return nil
}

// updateObservedGeneration updates the observed generation of the podPool
func (r *PodPoolReconciler) updateObservedGeneration(ctx context.Context, podPool *kwoksigsv1beta1.PodPool) error {
	podPool.Status.ObservedGeneration = podPool.Generation
	return r.Status().Update(ctx, podPool)
}

func (r *PodPoolReconciler) createPods(ctx context.Context, podPool *kwoksigsv1beta1.PodPool, pods []corev1.Pod) error {
	podLabels := podPool.Spec.PodTemplate.Labels
	if podLabels == nil {
		podLabels = make(map[string]string)
	}
	podLabels[controllerLabel] = podPool.Name
	podToleration := podPool.Spec.PodTemplate.Spec.Tolerations
	if podToleration == nil {
		podToleration = make([]corev1.Toleration, 0)
	}
	podToleration = append(podToleration, corev1.Toleration{
		Key:      controllerAnnotation,
		Operator: corev1.TolerationOpExists,
		Effect:   corev1.TaintEffectNoSchedule,
	})

	podAnnotation := podPool.Spec.PodTemplate.Annotations
	if podAnnotation == nil {
		podAnnotation = make(map[string]string)
	}
	podAnnotation[controllerAnnotation] = fakeString
	for i := int32(len(pods)); i < podPool.Spec.PodCount; i++ {
		// Create a new pod
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: podPool.Name + "-",
				Namespace:    podPool.Namespace,
				Labels:       podLabels,
				Annotations:  podAnnotation,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(podPool, kwoksigsv1beta1.GroupVersion.WithKind("PodPool")),
				},
			},
			Spec: podPool.Spec.PodTemplate.Spec,
		}
		pod.Spec.Tolerations = podToleration

		err := r.Create(ctx, pod)
		if err != nil {
			return err
		}
	}

	err := r.updateObservedGeneration(ctx, podPool)
	if err != nil {
		return err
	}
	return nil
}
