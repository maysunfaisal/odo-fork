package generator

import (
	devfileParser "github.com/openshift/odo/pkg/devfile/parser"
	corev1 "k8s.io/api/core/v1"
)

// GetContainers iterates through the components in the devfile and returns a slice of the corresponding containers
func (g *GeneratorFakeData) GetContainers(devfileObj devfileParser.DevfileObj) ([]corev1.Container, error) {
	var containers []corev1.Container

	return containers, nil
}
