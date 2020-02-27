package storage

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/openshift/odo/pkg/kclient"
	"github.com/openshift/odo/pkg/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	size = "1Gi"
)

// CreateComponentStorage creates PVCs with the given list of volume names if it does not exist, else it uses the existing PVC
func CreateComponentStorage(Client *kclient.Client, volumes []string, componentName string) (map[string]*corev1.PersistentVolumeClaim, error) {
	volumeNameToPVC := make(map[string]*corev1.PersistentVolumeClaim)

	for _, vol := range volumes {
		label := "component=" + componentName + ",storage-name=" + vol
		glog.V(3).Infof("Checking for PVC with name %v and label %v\n", vol, label)
		PVCs, err := Client.GetPVCsFromSelector(label)
		if err != nil {
			err = errors.New("Unable to get PVC with selectors " + label + ": " + err.Error())
			return nil, err
		}
		if len(PVCs) == 1 {
			glog.V(3).Infof("Found an existing PVC with name %v and label %v\n", vol, label)
			existingPVC := &PVCs[0]
			volumeNameToPVC[vol] = existingPVC
		} else if len(PVCs) == 0 {
			glog.V(3).Infof("Creating a PVC with name %v and label %v\n", vol, label)
			createdPVC, err := Create(Client, vol, componentName)
			volumeNameToPVC[vol] = createdPVC
			if err != nil {
				err = errors.New("Error creating PVC " + vol + ": " + err.Error())
				return nil, err
			}
		} else {
			err = errors.New("More than 1 PVC found with the label " + label + ": " + err.Error())
			return nil, err
		}
	}

	return volumeNameToPVC, nil
}

// Create creates the pvc for the given pvc name and component name
func Create(Client *kclient.Client, name, componentName string) (*corev1.PersistentVolumeClaim, error) {

	labels := map[string]string{
		"component":    componentName,
		"storage-name": name,
	}

	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse size: %v", size)
	}

	randomChars := util.GenerateRandomString(4)
	namespaceKubernetesObject, err := util.NamespaceOpenShiftObject(name, componentName)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create namespaced name")
	}
	namespaceKubernetesObject = fmt.Sprintf("%v-%v", namespaceKubernetesObject, randomChars)

	objectMeta := kclient.CreateObjectMeta(namespaceKubernetesObject, Client.Namespace, labels, nil)
	pvcSpec := kclient.GeneratePVCSpec(quantity)

	// Create PVC
	glog.V(3).Infof("Creating a PVC with name %v\n", namespaceKubernetesObject)
	pvc, err := Client.CreatePVC(objectMeta, *pvcSpec)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create PVC")
	}
	return pvc, nil
}
