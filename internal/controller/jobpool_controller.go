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

	batchv1 "k8s.io/api/batch/v1"
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

type JobPoolReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=jobpools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=jobpools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=jobpools/finalizers,verbs=update

func (r *JobPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling JobPool")
	jobPool := &kwoksigsv1beta1.JobPool{}
	err := r.Get(ctx, req.NamespacedName, jobPool)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("JobPool resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch JobPool")
		return ctrl.Result{}, err
	}
	if jobPool.Status.Conditions == nil || len(jobPool.Status.Conditions) == 0 {
		err := r.statusConditionController(ctx, jobPool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionUnknown,
			Reason:  "Reconciling",
			Message: "Starting to reconcile the JobPool",
		})
		if err != nil {
			log.Error(err, "Failed to update JobPool status")
			return ctrl.Result{}, err
		}
		err = r.Get(ctx, req.NamespacedName, jobPool)
		if err != nil {
			log.Error(err, "Failed to get jobPool")
			return ctrl.Result{}, err
		}
	}
	log.Info("JobPool resource found")
	// Set the finalizer for the JobPool
	if !controllerutil.ContainsFinalizer(jobPool, controllerFinalizer) {
		log.Info("Adding Finalizer for the JobPool")
		err = r.addFinalizer(ctx, jobPool)
		if err != nil {
			log.Error(err, "Failed to add finalizer for the JobPool")
			return ctrl.Result{}, err
		}
	}
	// Get the jobs for the JobPool
	jobs, err := r.getJobs(ctx, jobPool)
	if err != nil {
		log.Error(err, "Failed to get jobs")
		return ctrl.Result{}, err
	}
	// Check if the JobPool is in the desired state
	log.Info("Checking if the JobPool is in the desired state")
	println("jobs: ", len(jobs), "jobPool.Spec.JobCount: ", jobPool.Spec.JobCount)
	if int32(len(jobs)) != jobPool.Spec.JobCount {
		if int32(len(jobs)) < jobPool.Spec.JobCount {
			log.Info("Creating jobs")
			err := r.statusConditionController(ctx, jobPool, metav1.Condition{
				Type:    "Available",
				Status:  metav1.ConditionFalse,
				Reason:  "ScalingUp",
				Message: "Scalling up the jobPool",
			})
			if err != nil {
				log.Error(err, "Failed to update jobPool status")
				return ctrl.Result{}, err
			}
			log.Info("Scalling up the jobPool... creating jobs!")
			err = r.createJobs(ctx, jobPool, jobs)
			if err != nil {
				log.Error(err, "Failed to create jobs")
				return ctrl.Result{}, err
			}
		} else {
			log.Info("Deleting jobs")
			err := r.statusConditionController(ctx, jobPool, metav1.Condition{
				Type:    "Available",
				Status:  metav1.ConditionFalse,
				Reason:  "ScalingDown",
				Message: "Scalling down the jobPool",
			})
			if err != nil {
				log.Error(err, "Failed to update jobPool status")
				return ctrl.Result{}, err
			}
			log.Info("Scalling down the jobPool... deleting jobs!")
			err = r.deleteJobs(ctx, jobPool, jobs)
			if err != nil {
				log.Error(err, "Failed to delete jobs")
				return ctrl.Result{}, err
			}
		}
	}
	// Update the status of the jobPool
	err = r.statusConditionController(ctx, jobPool, metav1.Condition{
		Type:    "Available",
		Status:  metav1.ConditionTrue,
		Reason:  "jobPoolReconciled",
		Message: "jobPool reconciled successfully",
	})
	if err != nil {
		log.Error(err, "Failed to update jobPool status")
		return ctrl.Result{}, err
	}
	// Update the observed generation of the jobPool
	if jobPool.Status.ObservedGeneration != jobPool.Generation {
		log.Info("jobTemplate has changed")
		err := r.statusConditionController(ctx, jobPool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionFalse,
			Reason:  "Updating",
			Message: "Updating the jobPool",
		})
		if err != nil {
			log.Error(err, "Failed to update jobPool status")
			return ctrl.Result{}, err
		}
		emptyJobPool := &kwoksigsv1beta1.JobPool{
			ObjectMeta: metav1.ObjectMeta{
				Name: jobPool.Name,
			},
		}
		err = r.deleteJobs(ctx, emptyJobPool, jobs)
		if err != nil {
			log.Error(err, "Failed to delete jobs")
			return ctrl.Result{}, err
		}
		err = r.createJobs(ctx, jobPool, jobs)
		if err != nil {
			log.Error(err, "Failed to create jobs")
			return ctrl.Result{}, err
		}
		err = r.statusConditionController(ctx, jobPool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "Ready",
			Message: "jobPool is ready",
		})
		if err != nil {
			log.Error(err, "Failed to update jobPool status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	if !jobPool.DeletionTimestamp.IsZero() {
		log.Info("Deleting jobPool")
		err = r.statusConditionController(ctx, jobPool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionFalse,
			Reason:  "Deleting",
			Message: "Deleting the jobPool",
		})
		if err != nil {
			log.Error(err, "Failed to update jobPool status")
			return ctrl.Result{}, nil
		}
		err := r.deleteJobs(ctx, jobPool, jobs)
		if err != nil {
			log.Error(err, "Failed to delete jobs")
			return ctrl.Result{}, err
		}
		err = r.deleteFinalizer(ctx, jobPool)
		if err != nil {
			log.Error(err, "Failed to delete finalizer from jobPool")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	log.Info("Reconciliation finished")
	return ctrl.Result{RequeueAfter: time.Duration(60 * time.Second)}, nil
}

// delete the finalizer from the jobPool
func (r *JobPoolReconciler) deleteFinalizer(ctx context.Context, jobPool *kwoksigsv1beta1.JobPool) error {
	controllerutil.RemoveFinalizer(jobPool, controllerFinalizer)
	return r.Update(ctx, jobPool)
}

// delede jobs for the JobPool
func (r *JobPoolReconciler) deleteJobs(ctx context.Context, jobPool *kwoksigsv1beta1.JobPool, jobs []batchv1.Job) error {
	for i := int32(len(jobs)); i > jobPool.Spec.JobCount; i-- {
		// Delete a job
		err := r.Delete(ctx, &jobs[i-1])
		if err != nil {
			return err
		}
	}
	err := r.updateObservedGeneration(ctx, jobPool)
	if err != nil {
		return err
	}
	return nil
}

// updateObservedGeneration updates the observedGeneration of the JobPool
func (r *JobPoolReconciler) updateObservedGeneration(ctx context.Context, jobPool *kwoksigsv1beta1.JobPool) error {
	jobPool.Status.ObservedGeneration = jobPool.Generation
	return r.Status().Update(ctx, jobPool)
}

// create jobs for the JobPool
func (r *JobPoolReconciler) createJobs(ctx context.Context, jobPool *kwoksigsv1beta1.JobPool, jobs []batchv1.Job) error {
	manualSelector := true
	jobLabels := jobPool.Spec.JobTemplate.Spec.Template.Labels
	if jobLabels == nil {
		jobLabels = make((map[string]string), 0)
	}
	jobLabels[controllerLabel] = jobPool.Name
	jobToleration := jobPool.Spec.JobTemplate.Spec.Template.Spec.Tolerations
	if jobToleration == nil {
		jobToleration = make([]corev1.Toleration, 0)
	}
	jobToleration = append(jobToleration, corev1.Toleration{
		Key:      controllerAnnotation,
		Operator: corev1.TolerationOpExists,
		Effect:   corev1.TaintEffectNoSchedule,
	})

	jobAnnotation := jobPool.Spec.JobTemplate.Spec.Template.Annotations
	if jobAnnotation == nil {
		jobAnnotation = make(map[string]string)
	}
	jobAnnotation[controllerAnnotation] = fakeString
	for i := int32(len(jobs)); i < jobPool.Spec.JobCount; i++ {
		// Create a new jobs
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: jobPool.Name + "-",
				Namespace:    jobPool.Namespace,
				Labels:       jobLabels,
				Annotations:  jobAnnotation,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(jobPool, kwoksigsv1beta1.GroupVersion.WithKind("JobPool")),
				},
			},
			Spec: jobPool.Spec.JobTemplate.Spec,
		}
		job.Spec.Template.Spec.Tolerations = jobToleration
		job.Spec.ManualSelector = &manualSelector

		job.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				controllerLabel: jobPool.Name,
			},
		}
		job.Spec.Template.ObjectMeta.Labels = jobLabels
		err := r.Create(ctx, job)
		if err != nil {
			return err
		}
		err = r.updateObservedGeneration(ctx, jobPool)
		if err != nil {
			return err
		}
	}
	return nil
}

// get the jobs for the JobPool
func (r *JobPoolReconciler) getJobs(ctx context.Context, jobPool *kwoksigsv1beta1.JobPool) ([]batchv1.Job, error) {
	jobs := &batchv1.JobList{}
	err := r.List(ctx, jobs, client.InNamespace(jobPool.Namespace), client.MatchingLabels{controllerLabel: jobPool.Name})
	if err != nil && strings.Contains(err.Error(), "does not exist") {
		return []batchv1.Job{}, nil
	} else if err != nil {
		return nil, err
	}
	return jobs.Items, nil
}

// addFinalizer adds the finalizer to the JobPool
func (r *JobPoolReconciler) addFinalizer(ctx context.Context, jobPool *kwoksigsv1beta1.JobPool) error {
	controllerutil.AddFinalizer(jobPool, controllerFinalizer)
	return r.Update(ctx, jobPool)
}

// statusConditionController updates the status of the jobPool
func (r *JobPoolReconciler) statusConditionController(ctx context.Context, jodPool *kwoksigsv1beta1.JobPool, condition metav1.Condition) error {
	meta.SetStatusCondition(&jodPool.Status.Conditions, condition)
	return r.Status().Update(ctx, jodPool)
}

// SetupWithManager sets up the controller with the Manager.
func (r *JobPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kwoksigsv1beta1.JobPool{}).
		Complete(r)
}
