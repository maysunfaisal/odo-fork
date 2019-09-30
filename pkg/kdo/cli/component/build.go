package component

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/redhat-developer/odo-fork/pkg/build"
	"github.com/redhat-developer/odo-fork/pkg/kdo/genericclioptions"
	ktemplates "k8s.io/kubectl/pkg/util/templates"

	"github.com/spf13/cobra"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildRecommendedCommandName is the recommended catalog command name
const BuildRecommendedCommandName = "build"

var buildCmdExample = ktemplates.Examples(`  # Command for a full-build
%[1]s full <project name>

# Command for an incremental-build.
%[1]s inc <project name>
  `)

// BuildIDPOptions encapsulates the options for the udo catalog list idp command
type BuildIDPOptions struct {
	// list of build options
	buildTaskType       string
	projectName         string
	reuseBuildContainer bool
	useRuntimeContainer bool
	// generic context options common to all commands
	*genericclioptions.Context
}

// NewBuildIDPOptions creates a new BuildIDPOptions instance
func NewBuildIDPOptions() *BuildIDPOptions {
	return &BuildIDPOptions{}
}

// Complete completes BuildIDPOptions after they've been created
func (o *BuildIDPOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	fmt.Println("Build arguments: " + strings.Join(args, " "))
	o.Context = genericclioptions.NewContext(cmd)
	o.buildTaskType = args[0]
	o.projectName = args[1]
	o.reuseBuildContainer = true // Force re-use the build container for now, disable Kube Job
	fmt.Println("useRuntimeContainer flag: ", o.useRuntimeContainer)
	return
}

// Validate validates the BuildIDPOptions based on completed values
func (o *BuildIDPOptions) Validate() (err error) {
	if o.buildTaskType != string(build.Full) && o.buildTaskType != string(build.Incremental) {
		return fmt.Errorf("The first option should be either full or inc")
	}
	return
}

// Run contains the logic for the command associated with BuildIDPOptions
func (o *BuildIDPOptions) Run() (err error) {
	clientset := o.Context.Client.KubeClient
	namespace := o.Context.Client.Namespace

	fmt.Printf("Namespace: %s\n", namespace)

	idpClaimName := build.GetIDPPVC(o.Context.Client, namespace, "app=idp")
	fmt.Printf("Persistent Volume Claim: %s\n", idpClaimName)

	serviceAccountName := "default"
	fmt.Printf("Service Account: %s\n", serviceAccountName)

	// cwd is determined by Turbine, which will run the udo command in the project root directory
	cwd, err := os.Getwd()
	if err != nil {
		err = errors.New("Unable to get the cwd" + err.Error())
		return err
	}
	fmt.Printf("CWD: %s\n", cwd)

	if o.reuseBuildContainer && !o.useRuntimeContainer {
		// Create a Build Container for re-use if not present

		// Create the Reusable Build Container deployment object
		ReusableBuildContainerInstance := build.BuildTask{
			UseRuntime:         o.useRuntimeContainer,
			Kind:               string(build.ReusableBuildContainer),
			Name:               strings.ToLower(o.projectName) + "-reusable-build-container",
			Image:              "docker.io/maven:3.6",
			ContainerName:      "maven-build",
			Namespace:          namespace,
			PVCName:            idpClaimName,
			ServiceAccountName: serviceAccountName,
			// OwnerReferenceName: ownerReferenceName,
			// OwnerReferenceUID:  ownerReferenceUID,
			Privileged: true,
			MountPath:  "/data/idp/",
			SubPath:    "projects/" + o.projectName,
		}
		ReusableBuildContainerInstance.Labels = map[string]string{
			"app": ReusableBuildContainerInstance.Name,
		}

		// Check if the Reusable Build Container has already been deployed
		// Check if the pod is running and grab the pod name
		fmt.Printf("Checking if Reusable Build Container has already been deployed...\n")
		foundReusableBuildContainer := false
		timeout := int64(10)
		watchOptions := metav1.ListOptions{
			LabelSelector:  "app=" + ReusableBuildContainerInstance.Name,
			TimeoutSeconds: &timeout,
		}
		po, _ := o.Context.Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Checking to see if a Reusable Container is up")
		if po != nil {
			fmt.Printf("Running pod found: %s...\n\n", po.Name)
			ReusableBuildContainerInstance.PodName = po.Name
			foundReusableBuildContainer = true
		}

		if !foundReusableBuildContainer {
			fmt.Println("===============================")
			fmt.Println("Creating a pod...")
			volumes, volumeMounts := ReusableBuildContainerInstance.SetPFEVolumes()
			envVars := ReusableBuildContainerInstance.SetPFEEnvVars()

			pod, err := o.Context.Client.CreatePod(ReusableBuildContainerInstance.Name, ReusableBuildContainerInstance.ContainerName, ReusableBuildContainerInstance.Image, ReusableBuildContainerInstance.ServiceAccountName, ReusableBuildContainerInstance.Labels, volumes, volumeMounts, envVars, ReusableBuildContainerInstance.Privileged)
			if err != nil {
				err = errors.New("Failed to create a pod " + ReusableBuildContainerInstance.Name)
				return err
			}
			fmt.Println("Created pod: " + pod.GetName())
			fmt.Println("===============================")
			// Wait for pods to start and grab the pod name
			fmt.Printf("Waiting for pod to run\n")
			watchOptions := metav1.ListOptions{
				LabelSelector: "app=" + ReusableBuildContainerInstance.Name,
			}
			po, err := o.Context.Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Waiting for the Reusable Build Container to run")
			if err != nil {
				err = errors.New("The Reusable Build Container failed to run")
				return err
			}

			ReusableBuildContainerInstance.PodName = po.Name
		}

		fmt.Printf("The Reusable Build Container Pod Name: %s\n", ReusableBuildContainerInstance.PodName)

		watchOptions = metav1.ListOptions{
			LabelSelector: "app=" + ReusableBuildContainerInstance.Name,
		}
		err := o.syncProjectToRunningContainer(watchOptions, cwd, ReusableBuildContainerInstance.MountPath+"/src", ReusableBuildContainerInstance.ContainerName)
		if err != nil {
			fmt.Printf("Error occured while syncing to the pod %s: %s\n", ReusableBuildContainerInstance.PodName, err)
			err = errors.New("Unable to sync to the pod: " + err.Error())
			return err
		}

		// Execute the Build Tasks in the Build Container
		command := []string{"/bin/sh", "-c", ReusableBuildContainerInstance.MountPath + "/src" + string(build.FullBuildTask)}
		if o.buildTaskType == string(build.Incremental) {
			command = []string{"/bin/sh", "-c", ReusableBuildContainerInstance.MountPath + "/src" + string(build.IncrementalBuildTask)}
		}
		err = o.Context.Client.ExecCMDInContainer(ReusableBuildContainerInstance.PodName, "", command, os.Stdout, os.Stdout, nil, false)
		if err != nil {
			fmt.Printf("Error occured while executing command %s in the pod %s: %s\n", strings.Join(command, " "), ReusableBuildContainerInstance.PodName, err)
			err = errors.New("Unable to exec command " + strings.Join(command, " ") + " in the reusable build container: " + err.Error())
			return err
		}

		fmt.Println("Finished executing the IDP Build Task in the Reusable Build Container...")
	} else if !o.reuseBuildContainer && !o.useRuntimeContainer {
		// Create a Kube Job for building
		fmt.Println("Creating a Kube Job for building...")

		buildTaskJobName := "codewind-liberty-build-job"

		job, err := build.CreateBuildTaskKubeJob(buildTaskJobName, o.buildTaskType, namespace, idpClaimName, "projects/"+o.projectName, o.projectName)
		if err != nil {
			err = errors.New("There was a problem with the job configuration: " + err.Error())
			return err
		}

		kubeJob, err := clientset.BatchV1().Jobs(namespace).Create(job)
		if err != nil {
			err = errors.New("Failed to create a job: " + err.Error())
			return err
		}

		fmt.Printf("The job %s has been created\n", kubeJob.Name)
		watchOptions := metav1.ListOptions{
			LabelSelector: "job-name=codewind-liberty-build-job",
		}
		// Wait for pods to start running so that we can tail the logs
		po, err := o.Context.Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Waiting for the build job to run")
		if err != nil {
			err = errors.New("The build job failed to run")
			return err
		}

		err = o.Context.Client.GetPodLogs(po, os.Stdout)
		if err != nil {
			err = errors.New("Failed to get the logs for the build job")
			return err
		}

		// TODO-KDO: Set owner references
		var jobSucceeded bool
		// Loop and see if the job either succeeded or failed
		loop := true
		for loop == true {
			jobs, err := clientset.BatchV1().Jobs(namespace).List(metav1.ListOptions{})
			if err != nil {
				err = errors.New("Failed to list the job in the namepace: " + err.Error())
				return err
			}
			for _, job := range jobs.Items {
				if strings.Contains(job.Name, buildTaskJobName) {
					succeeded := job.Status.Succeeded
					failed := job.Status.Failed
					if succeeded == 1 {
						fmt.Printf("The job %s succeeded\n", job.Name)
						jobSucceeded = true
						loop = false
					} else if failed > 0 {
						fmt.Printf("The job %s failed\n", job.Name)
						jobSucceeded = false
						loop = false
					}
				}
			}
		}

		if loop == false {
			// delete the job
			gracePeriodSeconds := int64(0)
			deletionPolicy := metav1.DeletePropagationForeground
			err := clientset.BatchV1().Jobs(namespace).Delete(buildTaskJobName, &metav1.DeleteOptions{
				PropagationPolicy:  &deletionPolicy,
				GracePeriodSeconds: &gracePeriodSeconds,
			})
			if err != nil {
				return err
			}

			fmt.Printf("The job %s has been deleted\n", buildTaskJobName)
		}

		if !jobSucceeded {
			err = errors.New("The Kubernetes job failed")
			return err
		}
	}

	// Create the Codewind deployment object
	BuildTaskInstance := build.BuildTask{
		UseRuntime:         o.useRuntimeContainer,
		Kind:               string(build.Component),
		Name:               "cw-maysunliberty2-6c1b1ce0-cb4c-11e9-be96",
		Image:              string(build.RuntimeImage),
		ContainerName:      "libertyproject",
		Namespace:          namespace,
		PVCName:            idpClaimName,
		ServiceAccountName: serviceAccountName,
		// OwnerReferenceName: ownerReferenceName,
		// OwnerReferenceUID:  ownerReferenceUID,
		Privileged: true,
		MountPath:  "/config",
		SubPath:    "projects/" + o.projectName + "/buildartifacts/",
	}

	if o.useRuntimeContainer {
		BuildTaskInstance.Image = string(build.RuntimeWithMavenJavaImage)
		BuildTaskInstance.MountPath = "/home/default/idp"
		BuildTaskInstance.SubPath = ""
	}

	if o.useRuntimeContainer || o.buildTaskType == string(build.Full) {
		// Check if the Runtime Pod has been deployed
		// Check if the pod is running and grab the pod name
		fmt.Printf("Checking if Runtime Container has already been deployed...\n")
		foundRuntimeContainer := false
		timeout := int64(10)
		watchOptions := metav1.ListOptions{
			LabelSelector:  "app=javamicroprofiletemplate-selector,chart=javamicroprofiletemplate-1.0.0,release=" + BuildTaskInstance.Name,
			TimeoutSeconds: &timeout,
		}
		po, _ := o.Context.Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Checking to see if a Runtime Container has already been deployed")
		if po != nil {
			fmt.Printf("Running pod found: %s...\n\n", po.Name)
			BuildTaskInstance.PodName = po.Name
			foundRuntimeContainer = true
		}

		if !foundRuntimeContainer {
			// Deploy the application if it is a full build type and a running pod is not found
			fmt.Println("Deploying the application")

			BuildTaskInstance.Labels = map[string]string{
				"app":     "javamicroprofiletemplate-selector",
				"chart":   "javamicroprofiletemplate-1.0.0",
				"release": BuildTaskInstance.Name,
			}

			// Deploy Application
			deploy := BuildTaskInstance.CreateDeploy()
			service := BuildTaskInstance.CreateService()

			fmt.Println("===============================")
			fmt.Println("Deploying application...")
			_, err = clientset.CoreV1().Services(namespace).Create(&service)
			if err != nil {
				err = errors.New("Unable to create component service: " + err.Error())
				return err
			}
			fmt.Println("The service has been created.")

			_, err = clientset.AppsV1().Deployments(namespace).Create(&deploy)
			if err != nil {
				err = errors.New("Unable to create component deployment: " + err.Error())
				return err
			}
			fmt.Println("The deployment has been created.")
			fmt.Println("===============================")

			// Wait for the pod to run
			fmt.Printf("Waiting for pod to run\n")
			watchOptions := metav1.ListOptions{
				LabelSelector: "app=javamicroprofiletemplate-selector,chart=javamicroprofiletemplate-1.0.0,release=" + BuildTaskInstance.Name,
			}
			po, err := o.Context.Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Waiting for the Component Container to run")
			if err != nil {
				err = errors.New("The Component Container failed to run")
				return err
			}
			fmt.Println("The Component Pod is up and running: " + po.Name)
			BuildTaskInstance.PodName = po.Name
		}
	}

	if o.useRuntimeContainer {
		watchOptions := metav1.ListOptions{
			LabelSelector: "app=javamicroprofiletemplate-selector,chart=javamicroprofiletemplate-1.0.0,release=" + BuildTaskInstance.Name,
		}
		err := o.syncProjectToRunningContainer(watchOptions, cwd, BuildTaskInstance.MountPath+"/src", BuildTaskInstance.ContainerName)
		if err != nil {
			fmt.Printf("Error occured while syncing to the pod %s: %s\n", BuildTaskInstance.PodName, err)
			err = errors.New("Unable to sync to the pod: " + err.Error())
			return err
		}

		// Execute the Runtime task in the Runtime Container
		command := []string{"/bin/sh", "-c", BuildTaskInstance.MountPath + "/src" + string(build.FullRunTask)}
		if o.buildTaskType == string(build.Incremental) {
			command = []string{"/bin/sh", "-c", BuildTaskInstance.MountPath + "/src" + string(build.IncrementalRunTask)}
		}
		err = o.Context.Client.ExecCMDInContainer(BuildTaskInstance.PodName, "", command, os.Stdout, os.Stdout, nil, false)
		if err != nil {
			fmt.Printf("Error occured while executing command %s in the pod %s: %s\n", strings.Join(command, " "), BuildTaskInstance.PodName, err)
			err = errors.New("Unable to exec command " + strings.Join(command, " ") + " in the runtime container: " + err.Error())
			return err
		}
	}

	return
}

// SyncProjectToRunningContainer Wait for the Pod to run, create the targetPath in the Pod and sync the project to the targetPath
func (o *BuildIDPOptions) syncProjectToRunningContainer(watchOptions metav1.ListOptions, sourcePath, targetPath, containerName string) error {
	// Wait for the pod to run
	fmt.Printf("Waiting for pod to run\n")
	po, err := o.Context.Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Checking if the container is up before syncing")
	if err != nil {
		err = errors.New("The Container failed to run")
		return err
	}
	podName := po.Name
	fmt.Println("The Pod is up and running: " + podName)

	// Before Syncing, create the destination directory in the Build Container
	command := []string{"/bin/sh", "-c", "rm -rf " + targetPath + " && mkdir -p " + targetPath}
	err = o.Context.Client.ExecCMDInContainer(podName, "", command, os.Stdout, os.Stdout, nil, false)
	if err != nil {
		fmt.Printf("Error occured while executing command %s in the pod %s: %s\n", strings.Join(command, " "), podName, err)
		err = errors.New("Unable to exec command " + strings.Join(command, " ") + " in the reusable build container: " + err.Error())
		return err
	}

	// Sync the project to the Runtime Container on first deploy & update for the S2I model, skip if its a Build Container Model
	err = o.Context.Client.CopyFile(sourcePath, podName, targetPath, []string{}, []string{})
	if err != nil {
		err = errors.New("Unable to copy files to the pod " + podName + ": " + err.Error())
		return err
	}

	return nil
}

// NewCmdBuild implements the udo catalog list idps command
func NewCmdBuild(name, fullName string) *cobra.Command {
	o := NewBuildIDPOptions()

	var buildCmd = &cobra.Command{
		Use:     name,
		Short:   "Start a IDP Build",
		Long:    "Start a IDP Build using the Build Tasks.",
		Example: fmt.Sprintf(buildCmdExample, fullName),
		Args:    cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}

	buildCmd.Flags().BoolVar(&o.useRuntimeContainer, "useRuntimeContainer", false, "Use the runtime container for IDP Builds")

	return buildCmd
}
