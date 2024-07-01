package controller

import (
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
)

// setupScheme sets up the scheme for the tests
func setupScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = v1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	return scheme
}
