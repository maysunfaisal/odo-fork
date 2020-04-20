package utils

import (
	"fmt"
	"reflect"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/openshift/odo/pkg/devfile/parser/data/common"
	"github.com/openshift/odo/pkg/lclient"
	"github.com/openshift/odo/pkg/log"
	"github.com/pkg/errors"
)

// ComponentExists checks if Docker containers labeled with the specified component name exists
func ComponentExists(client lclient.Client, name string) bool {
	containers := GetComponentContainers(client, name)
	return len(containers) != 0
}

// GetComponentContainers returns a list of the running component containers
func GetComponentContainers(client lclient.Client, name string) (containers []types.Container) {
	containerList, err := client.GetContainerList()
	if err != nil {
		return
	}
	containers = client.GetContainersByComponent(name, containerList)

	return
}

// CreateAndInitVolume creates a docker volume and sets permission to the volume
// as volume of type volume, does not have client capability to set file mode on creation
func CreateAndInitVolume(client lclient.Client, volumeMount string, labels map[string]string) (types.Volume, error) {
	// Create the volume with the specified labels
	volume, err := client.CreateVolume(labels)
	if err != nil {
		return volume, err
	}

	VolumeInitContainerEntrypoint := []string{"/bin/sh"}
	VolumeInitContainerArgs := []string{"-c", "chmod -vR 777 " + volumeMount + " && ls -la " + volumeMount}

	// Initialize the volume by starting an init container that updates the volume file mode
	_, err = PullAndStartContainer(client,
		lclient.VolumeInitContainerImage,
		volume.Name,
		volumeMount,
		VolumeInitContainerEntrypoint,
		VolumeInitContainerArgs,
		nil,
		labels)
	if err != nil {
		return volume, err
	}

	// err = client.RemoveContainer(containerID)
	// if err != nil {
	// 	return volume, errors.Wrapf(err, "Unable to remove container %s for volume %s", containerID, volume.Name)
	// }

	return volume, nil
}

// PullAndStartContainer pulls and starts a container with the given image, volume and label
func PullAndStartContainer(client lclient.Client, image, volumeName, volumeMount string, entrypoint []string, cmd []string, envVars []string, labels map[string]string) (string, error) {
	// Container doesn't exist, so need to pull its image (to be safe) and start a new container
	s := log.Spinner("Pulling image " + image)
	err := PullImage(client, image)
	if err != nil {
		s.End(false)
		return "", errors.Wrapf(err, "Unable to pull %s image", image)
	}
	s.End(true)

	containerConfig := GenerateContainerConfig(client, image, entrypoint, cmd, envVars, labels)
	hostConfig := container.HostConfig{}

	if len(volumeName) > 0 {
		AddVolumeToHostConfig(volumeName, volumeMount, &hostConfig)
	}

	// Create the docker container
	s = log.Spinner("Starting container for " + image)
	defer s.End(false)
	containerID, err := StartContainer(client, &containerConfig, &hostConfig, nil)
	if err != nil {
		return containerID, err
	}
	s.End(true)

	return containerID, nil
}

// PullImage pulls an image
func PullImage(client lclient.Client, image string) error {
	err := client.PullImage(image)
	if err != nil {
		return errors.Wrapf(err, "Unable to pull %s image", image)
	}
	return nil
}

// StartContainer starts the container with the configs
func StartContainer(client lclient.Client, containerConfig *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig) (string, error) {
	containerID, err := client.StartContainer(containerConfig, hostConfig, networkingConfig)
	return containerID, err
}

// GenerateContainerConfig generates container config with the container information
func GenerateContainerConfig(client lclient.Client, image string, entrypoint []string, cmd []string, envVars []string, labels map[string]string) container.Config {
	containerConfig := client.GenerateContainerConfig(image, entrypoint, cmd, envVars, labels)

	return containerConfig
}

// ConvertEnvs converts environment variables from the devfile structure to an array of strings, as expected by Docker
func ConvertEnvs(vars []common.DockerimageEnv) []string {
	dockerVars := []string{}
	for _, env := range vars {
		envString := fmt.Sprintf("%s=%s", *env.Name, *env.Value)
		dockerVars = append(dockerVars, envString)
	}
	return dockerVars
}

// DoesContainerNeedUpdating returns true if a given container needs to be removed and recreated
// This function compares values in the container vs corresponding values in the devfile component.
// If any of the values between the two differ, a restart is required (and this function returns true)
// Unlike Kube, Docker doesn't provide a mechanism to update a container in place only when necesary
// so this function is necessary to prevent having to restart the container on every odo pushs
func DoesContainerNeedUpdating(component common.DevfileComponent, containerConfig *container.Config) bool {
	// If the image was changed in the devfile, the container needs to be updated
	if *component.Image != containerConfig.Image {
		return true
	}

	// Update the container if the env vars were updated in the devfile
	// Need to convert the devfile envvars to the format expected by Docker
	devfileEnvVars := ConvertEnvs(component.Env)
	return !reflect.DeepEqual(devfileEnvVars, containerConfig.Env)
}

// AddVolumeToHostConfig adds the volume to the container host config
func AddVolumeToHostConfig(volumeName, volumeMount string, hostConfig *container.HostConfig) *container.HostConfig {
	mount := mount.Mount{
		Type:   mount.TypeVolume,
		Source: volumeName,
		Target: volumeMount,
	}
	hostConfig.Mounts = append(hostConfig.Mounts, mount)

	return hostConfig
}

// GetProjectVolumeLabels returns the label selectors used to retrieve the project/source volume for a given component
func GetProjectVolumeLabels(componentName string) map[string]string {
	volumeLabels := map[string]string{
		"component": componentName,
		"type":      "projects",
	}
	return volumeLabels
}

// GetComponentContainerLabels returns the label selectors used to retrieve the component container for a given component and alias
func GetComponentContainerLabels(componentName, alias string) map[string]string {
	containerLabels := map[string]string{
		"component": componentName,
		"alias":     alias,
	}
	return containerLabels
}
