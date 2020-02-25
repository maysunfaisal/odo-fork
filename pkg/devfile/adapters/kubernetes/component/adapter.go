package component

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/glog"

	adapterCommon "github.com/openshift/odo/pkg/devfile/adapters/common"
	"github.com/openshift/odo/pkg/devfile/versions/common"
	"github.com/openshift/odo/pkg/kclient"
)

// New instantiantes a component adapter
func New(adapterContext adapterCommon.AdapterContext, client kclient.Client) Adapter {
	return Adapter{
		Client:         client,
		AdapterContext: adapterContext,
	}
}

// Adapter is a component adapter implementation for Kubernetes
type Adapter struct {
	Client kclient.Client
	adapterCommon.AdapterContext
}

// GetContainers iterates through the components in the devfile and returns a slice of the corresponding containers
func (a *Adapter) GetContainers() []corev1.Container {
	var containers []corev1.Container
	// Only components with aliases are considered because without an alias commands cannot reference them
	for _, comp := range a.Devfile.Data.GetAliasedComponents() {
		if comp.Type == common.DevfileComponentTypeDockerimage {
			glog.V(3).Infof("Found component %v with alias %v\n", comp.Type, *comp.Alias)
			envVars := convertEnvs(comp.Env)
			resourceReqs := getResourceReqs(comp)
			container := kclient.GenerateContainer(*comp.Alias, *comp.Image, false, comp.Command, comp.Args, envVars, resourceReqs)
			containers = append(containers, *container)
		}
	}
	return containers
}

// GetVolumes iterates through the components in the devfile and returns a slice of the corresponding containers
func (a *Adapter) GetVolumes() map[string][]common.DockerimageVolume {
	// componentAliasToVolumes is a map of the Devfile Component Alias to the Devfile Component Volumes
	componentAliasToVolumes := make(map[string][]common.DockerimageVolume)
	// Only components with aliases are considered because without an alias commands cannot reference them
	for _, comp := range a.Devfile.Data.GetAliasedComponents() {
		if comp.Type == common.DevfileComponentTypeDockerimage {
			if comp.Volumes != nil {
				for _, volume := range comp.Volumes {
					componentAliasToVolumes[*comp.Alias] = append(componentAliasToVolumes[*comp.Alias], volume)
				}
			}
		}
	}
	return componentAliasToVolumes
}

// Start updates the component if a matching component exists or creates one if it doesn't exist
func (a Adapter) Start() (err error) {
	componentName := a.ComponentName

	labels := map[string]string{
		"component": componentName,
	}

	containers := a.GetContainers()
	if len(containers) == 0 {
		return fmt.Errorf("No valid components found in the devfile")
	}

	componentAliasToVolumes := a.GetVolumes()

	// Get a list of all the unique volume names
	var uniqueVolumes []string
	processedVolumes := make(map[string]bool)
	for _, volumes := range componentAliasToVolumes {
		for _, vol := range volumes {
			if _, ok := processedVolumes[*vol.Name]; !ok {
				processedVolumes[*vol.Name] = true
				uniqueVolumes = append(uniqueVolumes, *vol.Name)
			}
		}
	}

	// createComponentStorage creates PVC from the unique Devfile volumes and returns a map of volume name to the PVC created
	volumeNameToPVC, err := createComponentStorage(&a.Client, uniqueVolumes, componentName)
	if err != nil {
		return err
	}

	objectMeta := kclient.CreateObjectMeta(componentName, a.Client.Namespace, labels, nil)
	podTemplateSpec := kclient.GeneratePodTemplateSpec(objectMeta, containers)

	// Add PVC to the podTemplateSpec
	err = kclient.AddPVCAndVolumeMount(podTemplateSpec, volumeNameToPVC, componentAliasToVolumes)
	if err != nil {
		return err
	}

	deploymentSpec := kclient.GenerateDeploymentSpec(*podTemplateSpec)

	glog.V(3).Infof("Creating deployment %v", deploymentSpec.Template.GetName())
	glog.V(3).Infof("The component name is %v", componentName)

	if componentExists(a.Client, componentName) {
		glog.V(3).Info("The component already exists, attempting to update it")
		_, err = a.Client.UpdateDeployment(*deploymentSpec)
		if err != nil {
			return err
		}
		glog.V(3).Infof("Successfully updated component %v", componentName)
	} else {
		_, err = a.Client.CreateDeployment(*deploymentSpec)
		if err != nil {
			return err
		}
		glog.V(3).Infof("Successfully created component %v", componentName)
	}

	podSelector := fmt.Sprintf("component=%s", componentName)
	watchOptions := metav1.ListOptions{
		LabelSelector: podSelector,
	}

	_, err = a.Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Waiting for component to start")
	return err
}
