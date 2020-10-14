package generic

import (
	"fmt"
	"strings"

	adaptersCommon "github.com/openshift/odo/pkg/devfile/adapters/common"
	"github.com/openshift/odo/pkg/devfile/parser/data/common"
	"k8s.io/apimachinery/pkg/api/resource"
)

// validateComponents validates that the components
// 1. makes sure the container components reference a valid volume component if it uses volume mounts
// 2. makes sure the volume components are unique
func validateComponents(components []common.DevfileComponent) error {

	processedVolumes := make(map[string]bool)
	processedVolumeMounts := make(map[string]bool)

	for _, component := range components {
		if component.Container != nil {
			// Process all the volume mounts in container components to validate them later
			for _, volumeMount := range component.Container.VolumeMounts {
				if _, ok := processedVolumeMounts[volumeMount.Name]; !ok {
					processedVolumeMounts[volumeMount.Name] = true
				}
			}

			// Check if any containers are customizing the reserved PROJECT_SOURCE or PROJECTS_ROOT env
			for _, env := range component.Container.Env {
				if env.Name == adaptersCommon.EnvProjectsSrc {
					return &ReservedEnvError{envName: adaptersCommon.EnvProjectsSrc, componentName: component.Name}
				} else if env.Name == adaptersCommon.EnvProjectsRoot {
					return &ReservedEnvError{envName: adaptersCommon.EnvProjectsRoot, componentName: component.Name}
				}
			}
		}

		if component.Volume != nil {
			if _, ok := processedVolumes[component.Name]; !ok {
				processedVolumes[component.Name] = true
				if len(component.Volume.Size) > 0 {
					// Only validate on Kubernetes since Docker volumes do not use sizes
					// We use the Kube API for validation because there are so many ways to
					// express storage in Kubernetes. For reference, you may check doc
					// https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
					if _, err := resource.ParseQuantity(component.Volume.Size); err != nil {
						return &InvalidVolumeError{name: component.Name, reason: fmt.Sprintf("size %s for volume component is invalid, %v. Example - 2Gi, 1024Mi", component.Volume.Size, err)}
					}
				}
			} else {
				return &InvalidVolumeError{name: component.Name, reason: "duplicate volume components present with the same name"}
			}
		}
	}

	// Check if the volume mounts mentioned in the containers are referenced by a volume component
	var invalidVolumeMounts []string
	for volumeMountName := range processedVolumeMounts {
		if _, ok := processedVolumes[volumeMountName]; !ok {
			invalidVolumeMounts = append(invalidVolumeMounts, volumeMountName)
		}
	}

	if len(invalidVolumeMounts) > 0 {
		return &MissingVolumeMountError{volumeName: strings.Join(invalidVolumeMounts, ",")}
	}

	// Successful
	return nil
}
