package lclient

import (
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

// GetContainersByComponent returns the list of Docker containers that matches the specified component label
// If no container with that component exists, it returns an empty list
func (dc *Client) GetContainersByComponent(componentName string, containers []types.Container) []types.Container {
	var containerList []types.Container

	for _, container := range containers {
		if container.Labels["component"] == componentName {
			containerList = append(containerList, container)
		}
	}
	return containerList
}

// GetContainersByComponentAndAlias returns the list of Docker containers that have the same component and alias labeled
func (dc *Client) GetContainersByComponentAndAlias(componentName string, alias string) ([]types.Container, error) {
	containerList, err := dc.GetContainerList()
	if err != nil {
		return nil, err
	}
	var labeledContainers []types.Container
	for _, container := range containerList {
		if container.Labels["component"] == componentName && container.Labels["alias"] == alias {
			labeledContainers = append(labeledContainers, container)
		}
	}
	return labeledContainers, nil
}

// GetContainerList returns a list of all of the running containers on the user's system
func (dc *Client) GetContainerList() ([]types.Container, error) {
	containers, err := dc.Client.ContainerList(dc.Context, types.ContainerListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to retrieve Docker containers")
	}
	return containers, nil
}

// StartContainer takes in a Docker container object and starts it.
// containerConfig - configurations for the container itself (image name, command, ports, etc) (if needed)
// hostConfig - configurations related to the host (volume mounts, exposed ports, etc) (if needed)
// networkingConfig - endpoints to expose (if needed)
// Returns an error if the container couldn't be started.
func (dc *Client) StartContainer(containerConfig *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig) error {
	resp, err := dc.Client.ContainerCreate(dc.Context, containerConfig, hostConfig, networkingConfig, "")
	if err != nil {
		return err
	}

	// Start the container
	if err := dc.Client.ContainerStart(dc.Context, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	return nil
}

// RemoveContainer takes in a given container ID and kills it, then removes it.
func (dc *Client) RemoveContainer(containerID string) error {
	err := dc.Client.ContainerRemove(dc.Context, containerID, types.ContainerRemoveOptions{
		Force: true,
	})
	if err != nil {
		return errors.Wrapf(err, "unable to remove container %s", containerID)
	}
	return nil
}

// GetContainerConfig takes in a given container ID and retrieves its corresponding container config
func (dc *Client) GetContainerConfig(containerID string) (*container.Config, error) {
	containerJSON, err := dc.Client.ContainerInspect(dc.Context, containerID)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to inspect container %s", containerID)
	}

	return containerJSON.Config, nil
}

//ExecCMDInContainer executes
func (dc *Client) ExecCMDInContainer(podName string, containerID string, cmd []string, stdout io.Writer, stderr io.Writer, stdin io.Reader, tty bool) error {

	execConfig := types.ExecConfig{
		AttachStdin:  stdin != nil,
		AttachStdout: stdout != nil,
		AttachStderr: stderr != nil,
		Cmd:          cmd,
		WorkingDir:   "/tmp",
	}

	resp, err := dc.Client.ContainerExecCreate(dc.Context, containerID, execConfig)
	if err != nil {
		glog.V(3).Infof("MJF err 1 %v", err)
		return err
	}
	// glog.V(3).Infof("MJF resp 1 %v", resp)

	// execStartCheck := types.ExecStartCheck{
	// 	Detach: true,
	// 	Tty:    tty,
	// }

	// err = dc.Client.ContainerExecStart(dc.Context, resp.ID, execStartCheck)
	// glog.V(3).Infof("MJF err 2 %v", err)

	aresp, err := dc.Client.ContainerExecAttach(dc.Context, resp.ID, types.ExecStartCheck{})
	if err != nil {
		glog.V(3).Infof("MJF err 2 %v", err)
		return err
	}
	defer aresp.Close()

	// read the output
	// var outBuf, errBuf bytes.Buffer
	outputDone := make(chan error)

	go func() {
		// StdCopy demultiplexes the stream into two buffers
		_, err = stdcopy.StdCopy(stdout, stderr, aresp.Reader)
		outputDone <- err
	}()

	err = <-outputDone
	if err != nil {
		glog.V(3).Infof("MJF err 3 %v", err)
		return err
	}

	// glog.V(3).Infof("MJF &stdout %v", stdout)
	// glog.V(3).Infof("MJF &stderr %v", stderr)

	return nil
}
