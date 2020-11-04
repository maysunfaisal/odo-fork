package generator

import (
	devfileParser "github.com/openshift/odo/pkg/devfile/parser"
	corev1 "k8s.io/api/core/v1"
)

// Generator is an interface that defines the function definition for Kubernetes resources
type Generator interface {
	GetContainers(devfileObj devfileParser.DevfileObj) ([]corev1.Container, error)
}
