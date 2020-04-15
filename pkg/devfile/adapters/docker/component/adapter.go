package component

import (
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/openshift/odo/pkg/devfile/adapters/common"
	"github.com/openshift/odo/pkg/devfile/adapters/docker/utils"
	"github.com/openshift/odo/pkg/exec"
	"github.com/openshift/odo/pkg/lclient"
)

// New instantiantes a component adapter
func New(adapterContext common.AdapterContext, client lclient.Client) Adapter {
	return Adapter{
		Client:         client,
		AdapterContext: adapterContext,
	}
}

// Adapter is a component adapter implementation for Kubernetes
type Adapter struct {
	Client lclient.Client
	common.AdapterContext
}

// Push updates the component if a matching component exists or creates one if it doesn't exist
func (a Adapter) Push(parameters common.PushParameters) (err error) {
	componentExists := utils.ComponentExists(a.Client, a.ComponentName)

	if componentExists {
		err = a.updateComponent()
	} else {
		err = a.createComponent()
	}

	if err != nil {
		return errors.Wrap(err, "unable to create or update component")
	}

	cmd := []string{"/bin/sh", "-c", "./loop.sh"}
	glog.V(3).Infof("MJF hola %v", cmd)
	// err = a.Client.ExecCMDInContainer("", "ac758bf7fb60", cmd, nil, nil, nil, false)
	err = exec.ExecuteCommand(&a.Client, "", "ac758bf7fb60", cmd, false)
	if err != nil {
		return errors.Wrap(err, "unable to exec component")
	}

	return nil
}

// DoesComponentExist returns true if a component with the specified name exists, false otherwise
func (a Adapter) DoesComponentExist(cmpName string) bool {
	return utils.ComponentExists(a.Client, cmpName)
}

// Delete attempts to delete the component with the specified labels, returning an error if it fails
// Stub function until the functionality is added as part of https://github.com/openshift/odo/issues/2581
func (d Adapter) Delete(labels map[string]string) error {
	return nil
}
