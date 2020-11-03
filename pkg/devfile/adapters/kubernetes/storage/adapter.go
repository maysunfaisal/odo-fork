package storage

import (
	"github.com/openshift/odo/pkg/devfile/adapters/common"
	"github.com/openshift/odo/pkg/kclient"

	corev1 "k8s.io/api/core/v1"
)

// New instantiates a storage adapter
func New(adapterContext common.AdapterContext, client kclient.Client) common.StorageAdapter {
	return &Adapter{
		Client:         client,
		AdapterContext: adapterContext,
	}
}

// Adapter is a storage adapter implementation for Kubernetes
type Adapter struct {
	Client kclient.Client
	common.AdapterContext
}

// Create creates the component pvc storage if it does not exist
func (a *Adapter) Create(volumeMap map[corev1.Volume]*corev1.PersistentVolumeClaimSpec) (err error) {

	// createComponentStorage creates PVC from the unique Devfile volumes if it does not exist
	err = CreateComponentStorage(&a.Client, volumeMap, a.ComponentName)
	if err != nil {
		return err
	}

	return
}
