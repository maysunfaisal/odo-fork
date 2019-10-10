package component

import (
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/redhat-developer/odo-fork/pkg/config"
	"github.com/redhat-developer/odo-fork/pkg/idp"
	"github.com/redhat-developer/odo-fork/pkg/kclient"
	"github.com/redhat-developer/odo-fork/pkg/log"
	"github.com/redhat-developer/odo-fork/pkg/storage"
	"github.com/redhat-developer/odo-fork/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildTaskExec is the Build Task execution implementation of the IDP build task
func BuildTaskExec(Client *kclient.Client, componentConfig config.LocalConfigInfo, fullBuild bool, devPack *idp.IDP) error {
	// clientset := Client.KubeClient
	namespace := Client.Namespace
	cmpName := componentConfig.GetName()
	appName := componentConfig.GetApplication()
	// Namespace the component
	namespacedKubernetesObject, err := util.NamespaceKubernetesObject(cmpName, appName)

	glog.V(0).Infof("Namespace: %s\n", namespace)

	idpClaimName := ""
	var cmpPVC *corev1.PersistentVolumeClaim

	PVCs, err := Client.GetPVCsFromSelector("app.kubernetes.io/component-name=" + cmpName + ",app.kubernetes.io/storage-name=" + cmpName)
	if err != nil {
		glog.V(0).Infof("Error occured while getting the PVC")
		err = errors.New("Unable to get the PVC: " + err.Error())
		return err
	}
	if len(PVCs) == 1 {
		cmpPVC = &PVCs[0]
	}

	if len(PVCs) == 0 {
		cmpPVC, err = storage.Create(Client, cmpName, devPack.Spec.Shared.Volumes[0].Size, cmpName, appName)
		if err != nil {
			glog.V(0).Infof("Error creating the PVC")
			err = errors.New("Error creating the PVC: " + err.Error())
			return err
		}
	}

	idpClaimName = cmpPVC.GetName()

	glog.V(0).Infof("Persistent Volume Claim: %s\n", idpClaimName)

	serviceAccountName := "default"
	glog.V(0).Infof("Service Account: %s\n", serviceAccountName)

	// cwd is the project root dir, where udo command will run
	cwd, err := os.Getwd()
	if err != nil {
		err = errors.New("Unable to get the cwd" + err.Error())
		return err
	}
	glog.V(0).Infof("CWD: %s\n", cwd)

	// Create the Reusable Build Container deployment object
	ReusableBuildContainerInstance := BuildTask{
		UseRuntime:         false,
		Kind:               ReusableBuildContainerType,
		Name:               namespacedKubernetesObject[:40] + "-build-container",
		Image:              devPack.Spec.Shared.Containers[0].Image,
		ContainerName:      devPack.Spec.Shared.Containers[0].Name,
		Namespace:          namespace,
		PVCName:            idpClaimName,
		ServiceAccountName: serviceAccountName,
		// OwnerReferenceName: ownerReferenceName,
		// OwnerReferenceUID:  ownerReferenceUID,
		Privileged: true,
		MountPath:  devPack.Spec.Shared.Containers[0].VolumeMappings[0].ContainerPath,
		SubPath:    "projects/" + cmpName,
	}
	ReusableBuildContainerInstance.Labels = map[string]string{
		"app": ReusableBuildContainerInstance.Name,
	}

	// Check if the Reusable Build Container has already been deployed
	// Check if the pod is running and grab the pod name
	glog.V(0).Infof("Checking if Reusable Build Container has already been deployed...\n")
	foundReusableBuildContainer := false
	timeout := int64(10)
	watchOptions := metav1.ListOptions{
		LabelSelector:  "app=" + ReusableBuildContainerInstance.Name,
		TimeoutSeconds: &timeout,
	}
	po, _ := Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Checking to see if a Reusable Container is up")
	if po != nil {
		glog.V(0).Infof("Running pod found: %s...\n\n", po.Name)
		ReusableBuildContainerInstance.PodName = po.Name
		foundReusableBuildContainer = true
	}

	if !foundReusableBuildContainer {
		glog.V(0).Info("===============================")
		glog.V(0).Info("Creating a pod...")
		volumes, volumeMounts := SetVolumes(ReusableBuildContainerInstance)
		envVars := SetEnvVars(ReusableBuildContainerInstance)

		pod, err := Client.CreatePod(ReusableBuildContainerInstance.Name, ReusableBuildContainerInstance.ContainerName, ReusableBuildContainerInstance.Image, ReusableBuildContainerInstance.ServiceAccountName, ReusableBuildContainerInstance.Labels, volumes, volumeMounts, envVars, ReusableBuildContainerInstance.Privileged)
		if err != nil {
			glog.V(0).Info("Failed to create a pod: " + err.Error())
			err = errors.New("Failed to create a pod " + ReusableBuildContainerInstance.Name)
			return err
		}
		glog.V(0).Info("Created pod: " + pod.GetName())
		glog.V(0).Info("===============================")
		// Wait for pods to start and grab the pod name
		glog.V(0).Infof("Waiting for pod to run\n")
		watchOptions := metav1.ListOptions{
			LabelSelector: "app=" + ReusableBuildContainerInstance.Name,
		}
		po, err := Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Waiting for the Reusable Build Container to run")
		if err != nil {
			err = errors.New("The Reusable Build Container failed to run")
			return err
		}

		ReusableBuildContainerInstance.PodName = po.Name
	}

	glog.V(0).Infof("The Reusable Build Container Pod Name: %s\n", ReusableBuildContainerInstance.PodName)

	watchOptions = metav1.ListOptions{
		LabelSelector: "app=" + ReusableBuildContainerInstance.Name,
	}
	err = syncProjectToRunningContainer(Client, watchOptions, cwd, ReusableBuildContainerInstance.MountPath+"/src")
	if err != nil {
		glog.V(0).Infof("Error occured while syncing to the pod %s: %s\n", ReusableBuildContainerInstance.PodName, err)
		err = errors.New("Unable to sync to the pod: " + err.Error())
		return err
	}

	if fullBuild {
		for _, scenario := range devPack.Spec.Scenarios {
			if scenario.Name == "full-build" {
				for _, scenariotask := range scenario.Tasks {
					for _, task := range devPack.Spec.Tasks {
						if scenariotask == task.Name {
							err = executetask(Client, strings.Join(task.Command, " "), ReusableBuildContainerInstance.PodName)
							if err != nil {
								glog.V(0).Infof("Error occured while executing command %s in the pod %s: %s\n", strings.Join(task.Command, " "), ReusableBuildContainerInstance.PodName, err)
								err = errors.New("Unable to exec command " + strings.Join(task.Command, " ") + " in the runtime container: " + err.Error())
								return err
							}
						}
					}
				}
			}
		}
	} else {
		for _, scenario := range devPack.Spec.Scenarios {
			if scenario.Name == "incremental-build" {
				for _, scenariotask := range scenario.Tasks {
					for _, task := range devPack.Spec.Tasks {
						if scenariotask == task.Name {
							err = executetask(Client, strings.Join(task.Command, " "), ReusableBuildContainerInstance.PodName)
							if err != nil {
								glog.V(0).Infof("Error occured while executing command %s in the pod %s: %s\n", strings.Join(task.Command, " "), ReusableBuildContainerInstance.PodName, err)
								err = errors.New("Unable to exec command " + strings.Join(task.Command, " ") + " in the runtime container: " + err.Error())
								return err
							}
						}
					}
				}
			}
		}
	}

	// Create the Runtime Task Instance
	RuntimeTaskInstance := BuildTask{
		UseRuntime:         false,
		Kind:               ComponentType,
		Name:               namespacedKubernetesObject[:40] + "-runtime",
		Image:              devPack.Spec.Runtime.Image,
		ContainerName:      namespacedKubernetesObject[:40] + "-container",
		Namespace:          namespace,
		PVCName:            idpClaimName,
		ServiceAccountName: serviceAccountName,
		// OwnerReferenceName: ownerReferenceName,
		// OwnerReferenceUID:  ownerReferenceUID,
		Privileged: true,
		MountPath:  devPack.Spec.Runtime.VolumeMappings[0].ContainerPath,
		SubPath:    "projects/" + cmpName + "/buildartifacts/",
	}

	foundRuntimeContainer := false
	timeout = int64(10)
	watchOptions = metav1.ListOptions{
		LabelSelector:  "app=" + namespacedKubernetesObject + ",deployment=" + namespacedKubernetesObject,
		TimeoutSeconds: &timeout,
	}
	po, _ = Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Checking to see if a Runtime Container has already been deployed")
	if po != nil {
		glog.V(0).Infof("Running pod found: %s...\n\n", po.Name)
		RuntimeTaskInstance.PodName = po.Name
		foundRuntimeContainer = true
	}

	if !foundRuntimeContainer {
		// Deploy the application if it is a full build type and a running pod is not found
		glog.V(0).Info("Deploying the application")

		RuntimeTaskInstance.Labels = map[string]string{
			"app":     RuntimeTaskInstance.Name + "-selector",
			"chart":   RuntimeTaskInstance.Name + "-1.0.0",
			"release": RuntimeTaskInstance.Name,
		}

		s := log.Spinner("Creating component")
		defer s.End(false)
		if err = RuntimeTaskInstance.CreateComponent(Client, componentConfig, devPack, cmpPVC); err != nil {
			err = errors.New("Unable to create component deployment: " + err.Error())
			return err
		}
		s.End(true)
	}

	return nil
}
