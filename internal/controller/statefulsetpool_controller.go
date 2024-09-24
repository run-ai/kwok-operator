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
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	kwoksigsv1beta1 "github.com/run-ai/kwok-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// StatefulsetPoolReconciler reconciles a StatefulsetPool object
type StatefulsetPoolReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=statefulsetpools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=statefulsetpools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kwok.sigs.run-ai.com,resources=statefulsetpools/finalizers,verbs=update

func (r *StatefulsetPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling StatefulsetPool")
	statefulsetPool := &kwoksigsv1beta1.StatefulsetPool{}
	err := r.Get(ctx, req.NamespacedName, statefulsetPool)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("StatefulsetPool resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch StatefulsetPool")
		return ctrl.Result{}, err
	}
	log.Info("StatefulsetPool resource found", "StatefulsetPool", statefulsetPool)

	if statefulsetPool.Status.Conditions == nil || len(statefulsetPool.Status.Conditions) == 0 {
		err = r.statusConditionController(ctx, statefulsetPool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionUnknown,
			Reason:  "StatefulsetPoolCreated",
			Message: "Starting to reconcile Statefulset",
		})
		if err != nil {
			log.Error(err, "unable to update StatefulsetPool status")
			return ctrl.Result{}, err
		}
		err = r.Get(ctx, req.NamespacedName, statefulsetPool)
		if err != nil {
			log.Error(err, "unable to fetch statefulsetPool")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	// Add finalizer
	if !controllerutil.ContainsFinalizer(statefulsetPool, controllerFinalizer) {
		log.Info("Adding finalizer to StatefulsetPool")
		err = r.addFinalizer(ctx, statefulsetPool)
		if err != nil {
			log.Error(err, "unable to add finalizer to StatefulsetPool")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	// Get Statefulset from the cluster with the owner reference of statefulsetPool
	Statefulset, err := r.getStatefulset(ctx, statefulsetPool)
	if err != nil {
		log.Error(err, "unable to get Statefulset")
		return ctrl.Result{}, err
	}
	// check if statefulset is lower than the expected replicas
	if len(Statefulset) == 0 {
		for i := 0; i < int(statefulsetPool.Spec.StatefulsetCount); i++ {
			err = r.createStatefulset(ctx, statefulsetPool, i)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		err = r.updateObservedGeneration(ctx, statefulsetPool)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	err = r.statusConditionController(ctx, statefulsetPool, metav1.Condition{
		Type:    "Available",
		Status:  metav1.ConditionTrue,
		Reason:  "StatefulsetPoolReconciled",
		Message: "statefulsetPool reconciled successfully",
	})
	if err != nil {
		log.Error(err, "unable to update StatefulsetPool status")
		return ctrl.Result{}, err
	}
	// Update Statefulset
	if statefulsetPool.Status.ObservedGeneration != statefulsetPool.Generation {
		log.Info("StatefulsetPool resource changed. Updating Statefulset")
		err = r.Get(ctx, req.NamespacedName, statefulsetPool)
		if err != nil {
			log.Error(err, "unable to fetch StatefulsetPool")
			return ctrl.Result{}, err
		}
		err = r.statusConditionController(ctx, statefulsetPool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionFalse,
			Reason:  "StatefulsetPoolUpdated",
			Message: "Updating Statefulset",
		})
		log.Info("Updating StatefulsetPool")
		if err != nil {
			log.Error(err, "unable to update StatefulsetPool status")
			return ctrl.Result{}, err
		}
		forrceRequeue := false
		forrceRequeue, err = r.updateStatefulset(ctx, statefulsetPool)
		if err != nil {
			return ctrl.Result{}, err
		}
		if forrceRequeue {
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, nil
	}
	if !statefulsetPool.DeletionTimestamp.IsZero() {
		log.Info("Deleting StatefulsetPool")
		err = r.statusConditionController(ctx, statefulsetPool, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionFalse,
			Reason:  "Deleting",
			Message: "Deleting the statefulsetPool",
		})
		if err != nil {
			log.Error(err, "unable to update statefulsetPool status")
			return ctrl.Result{}, err
		}
		for _, Statefulset := range Statefulset {
			err = r.Delete(ctx, &Statefulset)
			if err != nil {
				log.Error(err, "unable to delete Statefulset")
				return ctrl.Result{}, err
			}
		}
		err = r.deleteFinalizer(ctx, statefulsetPool)
		if err != nil {
			log.Error(err, "unable to delete Finalizer")
			return ctrl.Result{}, err
		}
		// delete all PVC and PV created by the StatefulsetPool
		pvcList := &corev1.PersistentVolumeClaimList{}
		err = r.List(ctx, pvcList, client.MatchingLabels{controllerLabel: statefulsetPool.Name})
		if err != nil {
			return ctrl.Result{}, err
		}
		for _, pvc := range pvcList.Items {
			err = r.Delete(ctx, &pvc)
			if err != nil {
				log.Error(err, "unable to delete PVC")
			}
		}
		pvList := &corev1.PersistentVolumeList{}
		err = r.List(ctx, pvList, client.MatchingLabels{controllerLabel: statefulsetPool.Name})
		if err != nil {
			log.Error(err, "unable to list PV")
		}
		for _, pv := range pvList.Items {
			err = r.Delete(ctx, &pv)
			if err != nil {
				log.Error(err, "unable to delete PV")
			}
		}

		return ctrl.Result{}, nil
	}

	log.Info("Reconciliation completed successfully")
	return ctrl.Result{RequeueAfter: time.Duration(60 * time.Second)}, nil
}

func (r *StatefulsetPoolReconciler) statusConditionController(ctx context.Context, statefulsetPool *kwoksigsv1beta1.StatefulsetPool, condition metav1.Condition) error {
	meta.SetStatusCondition(&statefulsetPool.Status.Conditions, condition)
	return r.Status().Update(ctx, statefulsetPool)
}

func (r *StatefulsetPoolReconciler) addFinalizer(ctx context.Context, statefulsetPool *kwoksigsv1beta1.StatefulsetPool) error {
	controllerutil.AddFinalizer(statefulsetPool, controllerFinalizer)
	return r.Update(ctx, statefulsetPool)
}

func (r *StatefulsetPoolReconciler) deleteFinalizer(ctx context.Context, statefulsetPool *kwoksigsv1beta1.StatefulsetPool) error {
	controllerutil.RemoveFinalizer(statefulsetPool, controllerFinalizer)
	return r.Update(ctx, statefulsetPool)
}

func (r *StatefulsetPoolReconciler) getStatefulset(ctx context.Context, statefulsetPool *kwoksigsv1beta1.StatefulsetPool) ([]appsv1.StatefulSet, error) {
	// get all the Statefulset in the Namespace witt the label of the statefulsetPool
	Statefulset := &appsv1.StatefulSetList{}
	err := r.List(ctx, Statefulset, client.InNamespace(statefulsetPool.Namespace), client.MatchingLabels{controllerLabel: statefulsetPool.Name})
	if err != nil && strings.Contains(err.Error(), "does not exist") {
		return []appsv1.StatefulSet{}, nil
	} else if err != nil {
		return nil, err
	}
	return Statefulset.Items, nil
}

func (r *StatefulsetPoolReconciler) getPVCList(ctx context.Context, statefulsetPoolName string, statefulsetIndex int) ([]corev1.PersistentVolumeClaim, error) {
	pvcList := &corev1.PersistentVolumeClaimList{}
	err := r.List(ctx, pvcList, client.MatchingLabels{"statefulesetName": fmt.Sprintf("%s-%d", statefulsetPoolName, statefulsetIndex)})
	if err != nil {
		return nil, err
	}
	return pvcList.Items, nil
}

func (r *StatefulsetPoolReconciler) getPVList(ctx context.Context, statefulsetPoolName string, statefulsetIndex int) ([]corev1.PersistentVolume, error) {
	pvList := &corev1.PersistentVolumeList{}
	err := r.List(ctx, pvList, client.MatchingLabels{"statefulesetName": fmt.Sprintf("%s-%d", statefulsetPoolName, statefulsetIndex)})
	if err != nil {
		return nil, err
	}
	return pvList.Items, nil
}

// update Statefulset
func (r *StatefulsetPoolReconciler) updateStatefulset(ctx context.Context, statefulsetPool *kwoksigsv1beta1.StatefulsetPool) (bool, error) {
	// get the Statefulset spec from the cluster
	forceRequeue := false
	Statefulsets, err := r.getStatefulset(ctx, statefulsetPool)
	log.Log.Info("Updating Statefulset", "Statefulset", Statefulsets)
	if err != nil {
		return forceRequeue, err
	}
	storageClassName := statefulsetPool.Spec.StatefulsetTemplate.Spec.VolumeClaimTemplates[0].Spec.StorageClassName
	if storageClassName == nil {
		storageClassName, err = r.getDefaultStorageClassName(ctx)
		if err != nil {
			return forceRequeue, err
		}
	}
	if len(Statefulsets) < int(statefulsetPool.Spec.StatefulsetCount) {
		for i := len(Statefulsets); i < int(statefulsetPool.Spec.StatefulsetCount); i++ {
			err = r.createStatefulset(ctx, statefulsetPool, i)
			if err != nil {
				return forceRequeue, err
			}
		}
		forceRequeue = true
		return forceRequeue, nil
	} else if len(Statefulsets) > int(statefulsetPool.Spec.StatefulsetCount) {
		sortedStsList := make([]appsv1.StatefulSet, len(Statefulsets))
		copy(sortedStsList, Statefulsets)
		log.Log.Info("the length of the sts is", "length", len(sortedStsList))
		sort.Slice(sortedStsList, func(i, j int) bool {
			// Split the strings to extract the numeric part
			partsI := strings.Split(sortedStsList[i].Name, "-")
			partsJ := strings.Split(sortedStsList[j].Name, "-")
			// Convert the last part to integers for comparison
			numI, _ := strconv.Atoi(partsI[len(partsI)-1])
			numJ, _ := strconv.Atoi(partsJ[len(partsJ)-1])
			// Compare the numeric parts
			return numI < numJ
		})
		for i := len(sortedStsList); i > int(statefulsetPool.Spec.StatefulsetCount); i-- {
			log.Log.Info("the index is", "index", i-1)
			err = r.Delete(ctx, &sortedStsList[i-1])
			if err != nil {
				return forceRequeue, err
			}
			log.Log.Info("Deleting Statefulset", "Statefulset", sortedStsList[i-1])
			pvcList, err := r.getPVCList(ctx, statefulsetPool.Name, i-1)
			if err != nil {
				return forceRequeue, err
			}
			for _, pvc := range pvcList {
				log.Log.Info("Deleting PVC", "PVC", pvc)
				err = r.Delete(ctx, &pvc)
				if err != nil {
					return forceRequeue, err
				}
			}
			pvList, err := r.getPVList(ctx, statefulsetPool.Name, i-1)
			if err != nil {
				return forceRequeue, err
			}
			for _, pv := range pvList {
				log.Log.Info("Deleting PV", "PV", pv)
				err = r.Delete(ctx, &pv)
				if err != nil {
					return forceRequeue, err
				}
			}
		}
		forceRequeue = true
		return forceRequeue, nil
	} else {
		replicas := statefulsetPool.Spec.StatefulsetTemplate.Spec.Replicas
		for i := 0; i < len(Statefulsets); i++ {
			Statefulsets[i].Spec.Replicas = replicas
			Statefulsets[i].Spec.Template.Spec.Containers = statefulsetPool.Spec.StatefulsetTemplate.Spec.Template.Spec.Containers
			pvList, err := r.getPVList(ctx, statefulsetPool.Name, i)
			if err != nil {
				return forceRequeue, err
			}
			// scale up pv in case of replicas change
			if statefulsetPool.Spec.CreatePV && statefulsetPool.Spec.StatefulsetTemplate.Spec.VolumeClaimTemplates != nil && int(*replicas) > int(len(pvList)) {
				nameSuffix := statefulsetPool.Spec.StatefulsetTemplate.Spec.VolumeClaimTemplates[0].Name
				if len(pvList) < int(*replicas) {
					for j := int(len(pvList)); j < int(*replicas); j++ {
						pvName := fmt.Sprintf("%s-%s-%d", nameSuffix, fmt.Sprintf("%s-%d", statefulsetPool.Name, i), j)
						err = r.createPV(ctx, statefulsetPool, storageClassName, &pvName, i)
						if err != nil {
							return forceRequeue, err
						}
					}
				}
			}
			err = r.Update(ctx, &Statefulsets[i])
			if err != nil {
				return forceRequeue, err
			}
			// scale down pv in case of replicas change
			if statefulsetPool.Spec.CreatePV && statefulsetPool.Spec.StatefulsetTemplate.Spec.VolumeClaimTemplates != nil && int(*replicas) < int(len(pvList)) {
				log.Log.Info("scaling down PV and PVC")
				err = r.scaleDownPVandPVC(ctx, statefulsetPool, i)
				if err != nil {
					return forceRequeue, err
				}
			}
		}
	}
	err = r.updateObservedGeneration(ctx, statefulsetPool)
	if err != nil {
		return forceRequeue, err
	}
	return forceRequeue, nil
}

// create Statefulset
func (r *StatefulsetPoolReconciler) scaleDownPVandPVC(ctx context.Context, statefulsetPool *kwoksigsv1beta1.StatefulsetPool, statefulsetIndex int) error {
	pvcList, err := r.getPVCList(ctx, statefulsetPool.Name, statefulsetIndex)
	if err != nil {
		return err
	}
	pvList, err := r.getPVList(ctx, statefulsetPool.Name, statefulsetIndex)
	if err != nil {
		return err
	}
	replicas := statefulsetPool.Spec.StatefulsetTemplate.Spec.Replicas
	sortedPvList := make([]corev1.PersistentVolume, len(pvList))
	copy(sortedPvList, pvList)
	sort.Slice(sortedPvList, func(i, j int) bool {
		// Split the strings to extract the numeric part
		partsI := strings.Split(sortedPvList[i].Name, "-")
		partsJ := strings.Split(sortedPvList[j].Name, "-")
		// Convert the last part to integers for comparison
		numI, _ := strconv.Atoi(partsI[len(partsI)-1])
		numJ, _ := strconv.Atoi(partsJ[len(partsJ)-1])
		// Compare the numeric parts
		return numI < numJ
	})
	sortedPvcList := make([]corev1.PersistentVolumeClaim, len(pvcList))
	copy(sortedPvcList, pvcList)
	sort.Slice(sortedPvcList, func(i, j int) bool {
		// Split the strings to extract the numeric part
		partsI := strings.Split(sortedPvcList[i].Name, "-")
		partsJ := strings.Split(sortedPvcList[j].Name, "-")

		// Convert the last part to integers for comparison
		numI, _ := strconv.Atoi(partsI[len(partsI)-1])
		numJ, _ := strconv.Atoi(partsJ[len(partsJ)-1])

		// Compare the numeric parts
		return numI < numJ
	})
	for i := int(len(sortedPvcList)); i > int(*replicas); i-- {
		// sort PVC list by name to delete the biggest PVC number
		err = r.Delete(ctx, &sortedPvcList[i-1])
		if err != nil {
			return err
		}
	}
	for i := int(len(sortedPvList)); i > int(*replicas); i-- {
		err = r.Delete(ctx, &sortedPvList[i-1])
		if err != nil {
			return err
		}
	}
	if len(sortedPvList) == int(*replicas) {
		//delete all the pv and pvc
		for _, pvc := range sortedPvcList {
			log.Log.Info("Deleting PVC", "PVC", pvc)
			err = r.Delete(ctx, &pvc)
			if err != nil {
				return err
			}
		}
		for _, pv := range sortedPvList {
			log.Log.Info("Deleting PV", "PV", pv)
			err = r.Delete(ctx, &pv)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *StatefulsetPoolReconciler) createStatefulset(ctx context.Context, statefulsetPool *kwoksigsv1beta1.StatefulsetPool, statefulsetIndex int) error {
	appendSelector := statefulsetPool.Spec.StatefulsetTemplate.Spec.Selector
	appendSelector.MatchLabels[controllerLabel] = statefulsetPool.Name
	appendSelector.MatchLabels["statefulesetName"] = fmt.Sprintf("%s-%d", statefulsetPool.Name, statefulsetIndex)

	overrideLabels := statefulsetPool.Spec.StatefulsetTemplate.Spec.Template.Labels
	if overrideLabels == nil {
		overrideLabels = map[string]string{
			controllerLabel:    statefulsetPool.Name,
			"statefulesetName": fmt.Sprintf("%s-%d", statefulsetPool.Name, statefulsetIndex),
		}
	} else {
		overrideLabels[controllerLabel] = statefulsetPool.Name
		overrideLabels["statefulesetName"] = fmt.Sprintf("%s-%d", statefulsetPool.Name, statefulsetIndex)
	}

	StatefulsetToleration := statefulsetPool.Spec.StatefulsetTemplate.Spec.Template.Spec.Tolerations
	if StatefulsetToleration == nil {
		StatefulsetToleration = make([]corev1.Toleration, 0)
	}

	StatefulsetToleration = append(StatefulsetToleration, corev1.Toleration{
		Key:      controllerAnnotation,
		Operator: corev1.TolerationOpExists,
		Effect:   corev1.TaintEffectNoSchedule,
	})

	Statefulset := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(statefulsetPool, kwoksigsv1beta1.GroupVersion.WithKind("StatefulsetPool")),
			},
			Name:      fmt.Sprintf("%s-%d", statefulsetPool.Name, statefulsetIndex),
			Namespace: statefulsetPool.Namespace,
			Labels:    overrideLabels,
		},
		Spec: statefulsetPool.Spec.StatefulsetTemplate.Spec,
	}
	if statefulsetPool.Spec.CreatePV && statefulsetPool.Spec.StatefulsetTemplate.Spec.VolumeClaimTemplates != nil {
		pvCount := 0
		err := error(nil)
		replicas := statefulsetPool.Spec.StatefulsetTemplate.Spec.Replicas
		storageClassName := statefulsetPool.Spec.StatefulsetTemplate.Spec.VolumeClaimTemplates[0].Spec.StorageClassName
		if storageClassName == nil {
			storageClassName, err = r.getDefaultStorageClassName(ctx)
			if err != nil {
				return err
			}
		}
		pvList, err := r.getPVList(ctx, statefulsetPool.Name, statefulsetIndex)
		if err != nil {
			return err
		}
		if len(pvList) < int(*replicas) || len(pvList) == int(*replicas) {
			pvCount = len(pvList)
		}
		if int32(pvCount) != int32(*replicas) && int32(pvCount) < int32(*replicas) {
			nameSuffix := statefulsetPool.Spec.StatefulsetTemplate.Spec.VolumeClaimTemplates[0].Name
			for i := int32(pvCount); i < *replicas; i++ {
				pvName := fmt.Sprintf("%s-%s-%d", nameSuffix, fmt.Sprintf("%s-%d", statefulsetPool.Name, statefulsetIndex), i)
				println("the pv name is", pvName)
				err = r.createPV(ctx, statefulsetPool, storageClassName, &pvName, statefulsetIndex)
				if err != nil {
					return err
				}
			}
		}
	}
	Statefulset.Spec.Template.ObjectMeta.Labels = overrideLabels
	Statefulset.Spec.Selector = appendSelector
	Statefulset.Spec.Template.Spec.Tolerations = StatefulsetToleration

	err := r.Create(ctx, Statefulset)
	if err != nil {
		return err
	}
	return nil
}

// Get default storage class name
func (r *StatefulsetPoolReconciler) getDefaultStorageClassName(ctx context.Context) (*string, error) {
	storageClassList := &v1.StorageClassList{}
	err := r.List(ctx, storageClassList)
	if err != nil {
		return nil, err
	}
	for _, storageClass := range storageClassList.Items {
		if storageClass.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" || len(storageClassList.Items) == 1 {
			return &storageClass.Name, nil
		}
	}
	return nil, nil
}

// create PV
func (r *StatefulsetPoolReconciler) createPV(ctx context.Context, statefulsetPool *kwoksigsv1beta1.StatefulsetPool, storageClassName *string, pvName *string, statefulsetIndex int) error {
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: *pvName,
			Labels: map[string]string{
				controllerLabel:    statefulsetPool.Name,
				"type":             "Local",
				"statefulesetName": fmt.Sprintf("%s-%d", statefulsetPool.Name, statefulsetIndex),
			},
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: statefulsetPool.Spec.StatefulsetTemplate.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests[corev1.ResourceStorage],
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			StorageClassName: *storageClassName,
			ClaimRef: &corev1.ObjectReference{
				Namespace: statefulsetPool.Namespace,
				Name:      *pvName,
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				Local: &corev1.LocalVolumeSource{
					Path: fmt.Sprintf("/mnt/data/%s", *pvName),
				},
			},
			NodeAffinity: &corev1.VolumeNodeAffinity{
				Required: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "type",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"kwok"},
								},
							},
						},
					},
				},
			},
		},
	}
	if len(*storageClassName) == 0 {
		err := errors.New("storageClassName is nil, please provide a storageClassName in your sts or create a default storageClassName")
		return err
	}
	err := r.Create(ctx, pv)
	if err != nil {
		return err
	}
	return nil
}

// updateObservedGeneration updates the observed generation of the StatefulsetPool
func (r *StatefulsetPoolReconciler) updateObservedGeneration(ctx context.Context, statefulsetPool *kwoksigsv1beta1.StatefulsetPool) error {
	statefulsetPool.Status.ObservedGeneration = statefulsetPool.Generation
	return r.Status().Update(ctx, statefulsetPool)
}

// SetupWithManager sets up the controller with the Manager.
func (r *StatefulsetPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kwoksigsv1beta1.StatefulsetPool{}).
		Complete(r)
}
