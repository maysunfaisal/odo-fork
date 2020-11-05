package generator

import (
	devfileParser "github.com/openshift/odo/pkg/devfile/parser"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Generator is an interface that defines the function definition for Kubernetes resource generation
type Generator interface {
	GetObjectMeta(name, namespace string, labels, annotations map[string]string) metav1.ObjectMeta
	GetContainers(devfileObj devfileParser.DevfileObj) ([]corev1.Container, error)
	GetPodTemplateSpec(podTemplateSpecParams PodTemplateSpecParams) *corev1.PodTemplateSpec
	GetDeploymentSpec(deployParams DeploymentSpecParams) *appsv1.DeploymentSpec
	// GetPVCSpec(quantity resource.Quantity) *corev1.PersistentVolumeClaimSpec
	GetService(devfileObj devfileParser.DevfileObj, selectorLabels map[string]string) (*corev1.ServiceSpec, error)
	// GetIngressSpec(ingressParams IngressParams) *extensionsv1.IngressSpec
	// GetRouteSpec(routeParams RouteParams) *routev1.RouteSpec
	GetOwnerReference(deployment *appsv1.Deployment) metav1.OwnerReference
	// GetBuildConfig(buildConfigParams BuildConfigParams) buildv1.BuildConfig
	// GetSourceBuildStrategy(imageName, imageNamespace string) buildv1.BuildStrategy
}
