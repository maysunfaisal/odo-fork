package component

import (
	"os"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/redhat-developer/odo-fork/pkg/config"
	"github.com/redhat-developer/odo-fork/pkg/devfile"
	"github.com/redhat-developer/odo-fork/pkg/idp"
	"github.com/redhat-developer/odo-fork/pkg/kclient"
	"github.com/redhat-developer/odo-fork/pkg/log"
	"github.com/redhat-developer/odo-fork/pkg/storage"
	"github.com/redhat-developer/odo-fork/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TaskExecIDP is the Build Task or the Runtime Task execution implementation of the IDP
func TaskExecIDP(Client *kclient.Client, componentConfig config.LocalConfigInfo, fullBuild bool, devPack *idp.IDP) error {
	// clientset := Client.KubeClient
	namespace := Client.Namespace
	cmpName := componentConfig.GetName()
	appName := componentConfig.GetApplication()
	// Namespace the component
	namespacedKubernetesObject, err := util.NamespaceKubernetesObject(cmpName, appName)

	glog.V(0).Infof("Namespace: %s\n", namespace)

	// Get the IDP Scenario
	var idpScenario idp.SpecScenario
	if fullBuild {
		idpScenario, err = devPack.GetScenario("full-build")
	} else {
		idpScenario, err = devPack.GetScenario("incremental-build")
	}
	if err != nil {
		glog.V(0).Infof("Error occured while getting the scenarios from the IDP")
		err = errors.New("Error occured while getting the scenarios from the IDP: " + err.Error())
		return err
	}

	// Get the IDP Tasks
	var idpTasks []idp.SpecTask
	idpTasks = devPack.GetTasks(idpScenario)

	// Get the Runtime Ports
	// runtimePorts := devPack.GetPorts()

	// Get the Shared Volumes
	// This may need to be updated to handle mount and unmount of PVCs,
	// if user updates idp.yaml, check storage.go's Push() func for ref
	idpPVC := make(map[string]*corev1.PersistentVolumeClaim)
	sharedVolumes := devPack.GetSharedVolumes()

	for _, vol := range sharedVolumes {
		PVCs, err := Client.GetPVCsFromSelector("app.kubernetes.io/component-name=" + cmpName + ",app.kubernetes.io/storage-name=" + vol.Name)
		if err != nil {
			glog.V(0).Infof("Error occured while getting the PVC")
			err = errors.New("Unable to get the PVC: " + err.Error())
			return err
		}
		if len(PVCs) == 1 {
			existingPVC := &PVCs[0]
			idpPVC[vol.Name] = existingPVC
		}
		if len(PVCs) == 0 {
			createdPVC, err := storage.Create(Client, vol.Name, vol.Size, cmpName, appName)
			idpPVC[vol.Name] = createdPVC
			if err != nil {
				glog.V(0).Infof("Error creating the PVC: " + err.Error())
				err = errors.New("Error creating the PVC: " + err.Error())
				return err
			}
		}

		glog.V(0).Infof("Using PVC: %s\n", idpPVC[vol.Name].GetName())
	}

	serviceAccountName := "default"
	glog.V(0).Infof("Service Account: %s\n", serviceAccountName)

	// cwd is the project root dir, where udo command will run
	cwd, err := os.Getwd()
	if err != nil {
		err = errors.New("Unable to get the cwd" + err.Error())
		return err
	}
	glog.V(0).Infof("CWD: %s\n", cwd)

	timeout := int64(10)

	// Check if the component exists, otherwise create one
	glog.V(0).Infof("Checking if the Component has already been deployed...\n")

	// var taskContainerInfo idp.TaskContainerInfo
	// var containerName, containerImage, trimmedNamespacedKubernetesObject, srcDestination string
	// var pvcClaimName, mountPath, subPath []string
	// var cmpPVC []*corev1.PersistentVolumeClaim
	// var BuildTaskInstance BuildTask

	// if len(namespacedKubernetesObject) > 40 {
	// 	trimmedNamespacedKubernetesObject = namespacedKubernetesObject[:40]
	// } else {
	// 	trimmedNamespacedKubernetesObject = namespacedKubernetesObject
	// }
	// containerName = trimmedNamespacedKubernetesObject

	foundComponent := false
	watchOptions := metav1.ListOptions{
		LabelSelector:  "app=" + namespacedKubernetesObject + ",deployment=" + namespacedKubernetesObject,
		TimeoutSeconds: &timeout,
	}
	po, _ := Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Checking to see if a Component has already been deployed")
	if po != nil {
		glog.V(0).Infof("Running pod found: %s...\n\n", po.Name)
		// BuildTaskInstance.PodName = po.Name
		foundComponent = true
	}

	// Create component if it doesn't exist
	if !foundComponent {
		// taskContainerInfo = devPack.GetRuntimeInfo()

		// containerImage = taskContainerInfo.Image

		// for _, vm := range taskContainerInfo.VolumeMappings {
		// 	cmpPVC = append(cmpPVC, idpPVC[vm.VolumeName])
		// 	pvcClaimName = append(pvcClaimName, idpPVC[vm.VolumeName].Name)
		// 	mountPath = append(mountPath, vm.ContainerPath)
		// 	subPath = append(subPath, vm.SubPath)
		// }

		// BuildTaskInstance = BuildTask{
		// 	// UseRuntime:    true,
		// 	Name:          containerName,
		// 	Image:         containerImage,
		// 	ContainerName: containerName,
		// 	Namespace:     namespace,
		// 	// PVCName:            pvcClaimName,
		// 	ServiceAccountName: serviceAccountName,
		// 	// OwnerReferenceName: ownerReferenceName,
		// 	// OwnerReferenceUID:  ownerReferenceUID,
		// 	Privileged:     true,
		// 	MountPath:      mountPath,
		// 	SubPath:        subPath,
		// 	SrcDestination: srcDestination,
		// }
		// BuildTaskInstance.Labels = map[string]string{
		// 	"app": BuildTaskInstance.Name,
		// }
		// BuildTaskInstance.Ports = runtimePorts

		glog.V(0).Info("===============================")
		glog.V(0).Info("Creating the Component")

		s := log.Spinner("Creating component")
		defer s.End(false)
		// if err = BuildTaskInstance.CreateComponent(Client, componentConfig, cmpPVC); err != nil {
		// 	err = errors.New("Unable to create component deployment: " + err.Error())
		// 	return err
		// }
		labels := map[string]string{
			"app":        namespacedKubernetesObject,
			"deployment": namespacedKubernetesObject,
		}
		if po, err = createComponentFromIDP(devPack, namespace, serviceAccountName, labels); err != nil {
			err = errors.New("Unable to create component deployment: " + err.Error())
			return err
		}

		if _, err = Client.CreateDeployment(po); err != nil {
			err = errors.New("Unable to create component deployment: " + err.Error())
			return err
		}
		s.End(true)

		glog.V(0).Info("Successfully created the component")
		glog.V(0).Info("===============================")
	}

	// Execute task on component
	runTasks(Client, idpTasks, po)

	return nil
}

// Create the component based on all containers referenced in the IDP, we will have a single fat pod with all containers
func createComponentFromIDP(devPack *idp.IDP, namespace, serviceAccount string, labels map[string]string) (*corev1.Pod, error) {

	// Get all the possible tasks that can be run
	// var tasks []idp.SpecTask = devPack.Spec.Tasks
	// containerNames := make(map[string]struct{})
	// var exists = struct{}{}

	// // Find the container for each task and add it to the set
	// for _, task := range tasks {
	// 	if _, ok := containerNames[task.Container]; !ok {
	// 		containerNames[task.Container] = exists
	// 	}
	// }

	// Get a container reference for each container in the set
	containers := []corev1.Container{}

	for _, devContainer := range devPack.Spec.Shared.Containers {
		glog.V(0).Info("Shared container: ", devContainer.Name)
		k8container := kclient.GenerateContainerSpec(devContainer.Name, devContainer.Image, true)
		containers = append(containers, k8container)
	}

	if len(containers) == 0 {
		return nil, errors.New("No containers defined")
	}

	containerVolumesMap := make(map[string][]devfile.DockerimageVolume)
	volumePVCMap := make(map[string]*corev1.PersistentVolumeClaim)
	// Create a pod that includes all of the containers
	po, err := kclient.GeneratePodSpec("fatpod", namespace, serviceAccount, labels, containers, containerVolumesMap, volumePVCMap)

	return po, err

}
