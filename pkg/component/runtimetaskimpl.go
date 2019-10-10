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
	"github.com/redhat-developer/odo-fork/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RunTaskExec is the Run Task execution implementation of the IDP run task
func RunTaskExec(Client *kclient.Client, componentConfig config.LocalConfigInfo, fullBuild bool, devPack *idp.IDP) error {
	// clientset := Client.KubeClient
	namespace := Client.Namespace
	cmpName := componentConfig.GetName()
	appName := componentConfig.GetApplication()
	// Namespace the component
	namespacedKubernetesObject, err := util.NamespaceKubernetesObject(cmpName, appName)
	if err != nil {
		return errors.Wrapf(err, "unable to create namespaced name")
	}

	glog.V(0).Infof("Namespace: %s\n", namespace)

	idpClaimName := ""
	PVCs, err := Client.GetPVCsFromSelector("app=idp")
	if err != nil {
		glog.V(0).Infof("Error occured while getting the PVC")
		err = errors.New("Unable to get the PVC: " + err.Error())
		return err
	}
	if len(PVCs) == 1 {
		idpClaimName = PVCs[0].GetName()
	}
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

	// Create the Runtime Task Instance
	RuntimeTaskInstance := BuildTask{
		UseRuntime:         true,
		Kind:               ComponentType,
		Name:               namespacedKubernetesObject[:40] + "-runtime",
		Image:              devPack.Spec.Runtime.Image,
		ContainerName:      namespacedKubernetesObject[:40] + "-container",
		Namespace:          namespace,
		PVCName:            "",
		ServiceAccountName: serviceAccountName,
		// OwnerReferenceName: ownerReferenceName,
		// OwnerReferenceUID:  ownerReferenceUID,
		Privileged: true,
		MountPath:  devPack.Spec.Runtime.VolumeMappings[0].ContainerPath,
		SubPath:    "",
	}

	glog.V(0).Info("Checking if Runtime Container has already been deployed...\n")
	foundRuntimeContainer := false
	timeout := int64(10)
	watchOptions := metav1.ListOptions{
		LabelSelector:  "app=" + namespacedKubernetesObject + ",deployment=" + namespacedKubernetesObject,
		TimeoutSeconds: &timeout,
	}
	po, _ := Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Checking to see if a Runtime Container has already been deployed")
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
		if err = RuntimeTaskInstance.CreateComponent(Client, componentConfig, devPack, nil); err != nil {
			err = errors.New("Unable to create component deployment: " + err.Error())
			return err
		}
		s.End(true)

		// Wait for the pod to run
		glog.V(0).Info("Waiting for pod to run\n")
		watchOptions := metav1.ListOptions{
			LabelSelector: "app=" + namespacedKubernetesObject + ",deployment=" + namespacedKubernetesObject,
		}
		po, err := Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Waiting for the Component Container to run")
		if err != nil {
			err = errors.New("The Component Container failed to run")
			return err
		}
		glog.V(0).Info("The Component Pod is up and running: " + po.Name)
		RuntimeTaskInstance.PodName = po.Name
	}

	watchOptions = metav1.ListOptions{
		LabelSelector: "app=" + namespacedKubernetesObject + ",deployment=" + namespacedKubernetesObject,
	}
	err = syncProjectToRunningContainer(Client, watchOptions, cwd, RuntimeTaskInstance.MountPath+"/src")
	if err != nil {
		glog.V(0).Infof("Error occured while syncing to the pod %s: %s\n", RuntimeTaskInstance.PodName, err)
		err = errors.New("Unable to sync to the pod: " + err.Error())
		return err
	}

	if fullBuild {
		for _, scenario := range devPack.Spec.Scenarios {
			if scenario.Name == "full-build" {
				for _, scenariotask := range scenario.Tasks {
					for _, task := range devPack.Spec.Tasks {
						if scenariotask == task.Name {
							err = executetask(Client, strings.Join(task.Command, " "), RuntimeTaskInstance.PodName)
							if err != nil {
								glog.V(0).Infof("Error occured while executing command %s in the pod %s: %s\n", strings.Join(task.Command, " "), RuntimeTaskInstance.PodName, err)
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
							err = executetask(Client, strings.Join(task.Command, " "), RuntimeTaskInstance.PodName)
							if err != nil {
								glog.V(0).Infof("Error occured while executing command %s in the pod %s: %s\n", strings.Join(task.Command, " "), RuntimeTaskInstance.PodName, err)
								err = errors.New("Unable to exec command " + strings.Join(task.Command, " ") + " in the runtime container: " + err.Error())
								return err
							}
						}
					}
				}
			}
		}
	}

	return nil
}
