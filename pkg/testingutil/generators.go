package testingutil

import (
	"fmt"

	"github.com/openshift/odo/pkg/kclient/generator"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	devfileParser "github.com/openshift/odo/pkg/devfile/parser"
)

// GeneratorMockData is a mock struct
type GeneratorMockData struct {
}

// NewMockGenerator returns an instance of Generator Interface
func NewMockGenerator() generator.Generator {
	return &GeneratorMockData{}
}

// GetObjectMeta creates a common object meta
func (g *GeneratorMockData) GetObjectMeta(name, namespace string, labels, annotations map[string]string) metav1.ObjectMeta {

	return metav1.ObjectMeta{
		Name: name,
	}
}

// GetContainers is a mock function
func (g *GeneratorMockData) GetContainers(devfileObj devfileParser.DevfileObj) ([]corev1.Container, error) {
	return []corev1.Container{
		{
			Name: "dummy",
		},
	}, nil
}

// GetPodTemplateSpec creates a pod template spec that can be used to create a deployment spec
func (g *GeneratorMockData) GetPodTemplateSpec(podTemplateSpecParams generator.PodTemplateSpecParams) *corev1.PodTemplateSpec {
	podTemplateSpec := &corev1.PodTemplateSpec{
		ObjectMeta: podTemplateSpecParams.ObjectMeta,
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "volume1",
				},
			},
		},
	}

	return podTemplateSpec
}

// GetDeploymentSpec creates a deployment spec
func (g *GeneratorMockData) GetDeploymentSpec(deployParams generator.DeploymentSpecParams) *appsv1.DeploymentSpec {
	deploymentSpec := &appsv1.DeploymentSpec{
		Strategy: appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		},
		Selector: &metav1.LabelSelector{
			MatchLabels: deployParams.PodSelectorLabels,
		},
		Template: deployParams.PodTemplateSpec,
	}

	return deploymentSpec
}

// GetService iterates through the components in the devfile and returns a ServiceSpec
func (g *GeneratorMockData) GetService(devfileObj devfileParser.DevfileObj, selectorLabels map[string]string) (*corev1.ServiceSpec, error) {

	return &corev1.ServiceSpec{}, nil
}

func (g *GeneratorMockData) GetOwnerReference(deployment *appsv1.Deployment) metav1.OwnerReference {
	return metav1.OwnerReference{}
}

// GeneratorMockErrorData is a mock error struct
type GeneratorMockErrorData struct {
}

// NewMockErrorGenerator returns an instance of Generator Interface
func NewMockErrorGenerator() generator.Generator {
	return &GeneratorMockErrorData{}
}

// GetContainers is a mock function
func (g *GeneratorMockErrorData) GetContainers(devfileObj devfileParser.DevfileObj) ([]corev1.Container, error) {
	return []corev1.Container{}, nil
}

// GetObjectMeta creates a common object meta
func (g *GeneratorMockErrorData) GetObjectMeta(name, namespace string, labels, annotations map[string]string) metav1.ObjectMeta {

	return metav1.ObjectMeta{}
}

// GetPodTemplateSpec creates a pod template spec that can be used to create a deployment spec
func (g *GeneratorMockErrorData) GetPodTemplateSpec(podTemplateSpecParams generator.PodTemplateSpecParams) *corev1.PodTemplateSpec {
	return &corev1.PodTemplateSpec{}
}

// GetDeploymentSpec creates a deployment spec
func (g *GeneratorMockErrorData) GetDeploymentSpec(deployParams generator.DeploymentSpecParams) *appsv1.DeploymentSpec {
	return &appsv1.DeploymentSpec{}

}

// GetService iterates through the components in the devfile and returns a ServiceSpec
func (g *GeneratorMockErrorData) GetService(devfileObj devfileParser.DevfileObj, selectorLabels map[string]string) (*corev1.ServiceSpec, error) {
	return &corev1.ServiceSpec{}, fmt.Errorf("GetService() error")
}

func (g *GeneratorMockErrorData) GetOwnerReference(deployment *appsv1.Deployment) metav1.OwnerReference {
	return metav1.OwnerReference{}
}
