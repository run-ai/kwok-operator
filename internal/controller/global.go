package controller

import (
	"time"

	v1beta1 "github.com/run-ai/kwok-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	controllerFinalizer  = "kwok.sigs.run-ai.com/finalizer"
	controllerLabel      = "kwok.x-k8s.io/controller"
	controllerAnnotation = "kwok.x-k8s.io/node"
	fakeString           = "fake"
	// poolChurnerLabel selects Pods owned by a PoolChurner (distinct from controllerLabel to avoid clashes with PodPool same name).
	poolChurnerLabel = "kwok.sigs.run-ai.com/pool-churner"
)

// DefaultIdleRequeue is used when a pool is steady state; periodic reconcile catches drift.
const DefaultIdleRequeue = 60 * time.Second

// setupScheme sets up the scheme for the tests
func setupScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = v1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	return scheme
}
