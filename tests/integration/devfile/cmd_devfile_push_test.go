package devfile

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/openshift/odo/tests/helper"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("odo devfile push command tests", func() {
	var namespace string
	var context string
	var currentWorkingDirectory string

	var sourcePath = "/projects"

	// TODO: all oc commands in all devfile related test should get replaced by kubectl
	// TODO: to goal is not to use "oc"
	oc := helper.NewOcRunner("oc")

	// This is run after every Spec (It)
	var _ = BeforeEach(func() {
		SetDefaultEventuallyTimeout(10 * time.Minute)
		namespace = helper.CreateRandProject()
		context = helper.CreateNewDevfileContext()
		currentWorkingDirectory = helper.Getwd()
		helper.Chdir(context)
		os.Setenv("GLOBALODOCONFIG", filepath.Join(context, "config.yaml"))
	})

	// Clean up after the test
	// This is run after every Spec (It)
	var _ = AfterEach(func() {
		helper.DeleteProject(namespace)
		helper.Chdir(currentWorkingDirectory)
		helper.DeleteDir(context)
		os.Unsetenv("GLOBALODOCONFIG")
	})

	Context("Verify devfile push works", func() {

		It("Check that odo push works with a devfile", func() {
			// Devfile push requires experimental mode to be set
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")

			helper.CopyExample(filepath.Join("source", "devfiles", "nodejs"), context)

			output := helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml", "--namespace", namespace)
			Expect(output).To(ContainSubstring("Changes successfully pushed to component"))

			// update devfile and push again
			helper.ReplaceString("devfile.yaml", "name: FOO", "name: BAR")
			helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml", "--namespace", namespace)
		})

	})

	Context("When devfile push command is executed", func() {

		It("should not build when no changes are detected in the directory and build when a file change is detected", func() {
			// Devfile push requires experimental mode to be set
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")

			helper.CopyExample(filepath.Join("source", "devfiles", "nodejs"), context)

			helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml", "--namespace", namespace)
			output := helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml", "--namespace", namespace)
			Expect(output).To(ContainSubstring("No file changes detected, skipping build"))

			helper.ReplaceString(filepath.Join(context, "app", "app.js"), "Hello World!", "UPDATED!")
			output = helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml", "--namespace", namespace)
			// fmt.Printf("MJF %v", output)
		})

		It("should be able to create a file, push, delete, then push again propagating the deletions", func() {
			// Devfile push requires experimental mode to be set
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")

			helper.CopyExample(filepath.Join("source", "devfiles", "nodejs"), context)

			// Create a new file that we plan on deleting later...
			newFilePath := filepath.Join(context, "foobar.txt")
			if err := helper.CreateFileWithContent(newFilePath, "hello world"); err != nil {
				fmt.Printf("the foobar.txt file was not created, reason %v", err.Error())
			}

			// Create a new directory
			newDirPath := filepath.Join(context, "testdir")
			helper.MakeDir(newDirPath)

			// Push
			helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml", "--namespace", namespace)

			// component name is currently equal to directory name until odo create for devfiles is implemented
			cmpName := filepath.Base(context)

			// Check to see if it's been pushed (foobar.txt abd directory testdir)
			podName := oc.GetRunningPodNameByComponent(cmpName, namespace)

			stdOut := oc.ExecListDir(podName, namespace, sourcePath)
			Expect(stdOut).To(ContainSubstring(("foobar.txt")))
			Expect(stdOut).To(ContainSubstring(("testdir")))

			// Now we delete the file and dir and push
			helper.DeleteDir(newFilePath)
			helper.DeleteDir(newDirPath)
			helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml", "--namespace", namespace, "-v4")

			// Then check to see if it's truly been deleted
			stdOut = oc.ExecListDir(podName, namespace, sourcePath)
			Expect(stdOut).To(Not(ContainSubstring(("foobar.txt"))))
			Expect(stdOut).To(Not(ContainSubstring(("testdir"))))
		})

		It("should delete the files from the container if its removed locally", func() {
			// Devfile push requires experimental mode to be set
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")

			helper.CopyExample(filepath.Join("source", "devfiles", "nodejs"), context)
			helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml", "--namespace", namespace)

			// component name is currently equal to directory name until odo create for devfiles is implemented
			cmpName := filepath.Base(context)

			// Check to see if it's been pushed (foobar.txt abd directory testdir)
			podName := oc.GetRunningPodNameByComponent(cmpName, namespace)

			var statErr error
			oc.CheckCmdOpInRemoteDevfilePod(
				podName,
				"",
				namespace,
				[]string{"stat", "/projects/app/app.js"},
				func(cmdOp string, err error) bool {
					statErr = err
					return true
				},
			)
			Expect(statErr).ToNot(HaveOccurred())
			Expect(os.Remove(filepath.Join(context, "app", "app.js"))).NotTo(HaveOccurred())
			helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml", "--namespace", namespace)

			oc.CheckCmdOpInRemoteDevfilePod(
				podName,
				"",
				namespace,
				[]string{"stat", "/projects/app/app.js"},
				func(cmdOp string, err error) bool {
					statErr = err
					return true
				},
			)
			Expect(statErr).To(HaveOccurred())
			Expect(statErr.Error()).To(ContainSubstring("cannot stat '/projects/app/app.js': No such file or directory"))
		})

		It("should build when no changes are detected in the directory and force flag is enabled", func() {
			// Devfile push requires experimental mode to be set
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")

			helper.CopyExample(filepath.Join("source", "devfiles", "nodejs"), context)
			helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml", "--namespace", namespace)

			// use the force build flag and push
			output := helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml", "--namespace", namespace, "-f")
			Expect(output).To(Not(ContainSubstring("No file changes detected, skipping build")))
		})

		It("should have no errors when no endpoints within the devfile, should create a service when devfile has endpoints", func() {
			// Devfile push requires experimental mode to be set
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")

			helper.CopyExample(filepath.Join("source", "devfiles", "nodejs"), context)
			helper.RenameFile("devfile.yaml", "devfile-old.yaml")
			helper.RenameFile("devfile-no-endpoints.yaml", "devfile.yaml")
			componentName := path.Base(context)
			helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml", "--namespace", namespace)
			output := oc.GetServices(namespace)
			Expect(output).NotTo(ContainSubstring(componentName))

			helper.RenameFile("devfile-old.yaml", "devfile.yaml")
			output = helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml", "--namespace", namespace)

			Expect(output).To(ContainSubstring("Changes successfully pushed to component"))
			output = oc.GetServices(namespace)
			Expect(output).To(ContainSubstring(componentName))
		})

		It("should execute the default devbuild and devrun commands if present", func() {
			// Devfile push requires experimental mode to be set
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")

			helper.CopyExample(filepath.Join("source", "devfiles", "nodejs"), context)

			output := helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml", "--namespace", namespace)
			Expect(output).To(ContainSubstring("Executing devbuild command npm install"))
			Expect(output).To(ContainSubstring("Executing devrun command nodemon app.js"))
		})

		It("should be able to handle a missing devbuild command", func() {
			// Devfile push requires experimental mode to be set
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")

			helper.CopyExample(filepath.Join("source", "devfiles", "nodejs"), context)
			helper.RenameFile("devfile.yaml", "devfile-old.yaml")
			helper.RenameFile("devfile-without-devbuild.yaml", "devfile.yaml")

			output := helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml", "--namespace", namespace)
			Expect(output).NotTo(ContainSubstring("Executing devbuild command"))
			Expect(output).To(ContainSubstring("Executing devrun command npm install && nodemon app.js"))
		})

		It("should error out on a missing devrun command", func() {
			// Devfile push requires experimental mode to be set
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")

			helper.CopyExample(filepath.Join("source", "devfiles", "nodejs"), context)

			// Rename the devrun command
			helper.ReplaceString(filepath.Join(context, "devfile.yaml"), "devrun", "randomcommand")

			output := helper.CmdShouldFail("odo", "push", "--devfile", "devfile.yaml", "--namespace", namespace)
			Expect(output).NotTo(ContainSubstring("Executing devrun command"))
			Expect(output).To(ContainSubstring("Failed to start component with name"))
			Expect(output).To(ContainSubstring("The command \"devrun\" was not found in the devfile"))
		})

		It("should be able to push using the custom commands", func() {
			// Devfile push requires experimental mode to be set
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")

			helper.CopyExample(filepath.Join("source", "devfiles", "nodejs"), context)

			output := helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml", "--namespace", namespace, "--build-command", "build", "--run-command", "run")
			Expect(output).To(ContainSubstring("Executing build command npm install"))
			Expect(output).To(ContainSubstring("Executing run command nodemon app.js"))
		})

		It("should error out on a wrong custom commands", func() {
			garbageCommand := "buildgarbage"
			// Devfile push requires experimental mode to be set
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")

			helper.CopyExample(filepath.Join("source", "devfiles", "nodejs"), context)

			output := helper.CmdShouldFail("odo", "push", "--devfile", "devfile.yaml", "--namespace", namespace, "--build-command", garbageCommand)
			Expect(output).NotTo(ContainSubstring("Executing buildgarbage command"))
			Expect(output).To(ContainSubstring("Failed to start component with name"))
			Expect(output).To(ContainSubstring("The command \"%v\" was not found in the devfile", garbageCommand))
		})

		It("should create pvc and reuse if it shares the same devfile volume name", func() {
			// Devfile push requires experimental mode to be set
			helper.CmdShouldPass("odo", "preference", "set", "Experimental", "true")

			helper.CopyExample(filepath.Join("source", "devfiles", "nodejs"), context)
			helper.RenameFile("devfile.yaml", "devfile-old.yaml")
			helper.RenameFile("devfile-with-volumes.yaml", "devfile.yaml")

			output := helper.CmdShouldPass("odo", "push", "--devfile", "devfile.yaml", "--namespace", namespace)
			Expect(output).To(ContainSubstring("Executing devbuild command"))
			Expect(output).To(ContainSubstring("Executing devrun command"))

			// component name is currently equal to directory name until odo create for devfiles is implemented
			cmpName := filepath.Base(context)

			// Check to see if it's been pushed (foobar.txt abd directory testdir)
			podName := oc.GetRunningPodNameByComponent(cmpName, namespace)

			var statErr error
			var cmdOutput string
			oc.CheckCmdOpInRemoteDevfilePod(
				podName,
				"runtime2",
				namespace,
				[]string{"cat", "/data/myfile.log"},
				func(cmdOp string, err error) bool {
					cmdOutput = cmdOp
					statErr = err
					return true
				},
			)
			Expect(statErr).ToNot(HaveOccurred())
			Expect(cmdOutput).To(ContainSubstring("hello"))

			oc.CheckCmdOpInRemoteDevfilePod(
				podName,
				"runtime2",
				namespace,
				[]string{"stat", "/data2"},
				func(cmdOp string, err error) bool {
					statErr = err
					return true
				},
			)
			Expect(statErr).ToNot(HaveOccurred())

			volumesMatched := false

			// check the volume name and mount paths for the containers
			volNamesAndPaths := oc.GetVolumeMountNamesandPathsFromContainer(cmpName, "runtime", namespace)
			volNamesAndPathsArr := strings.Fields(volNamesAndPaths)
			for _, volNamesAndPath := range volNamesAndPathsArr {
				volNamesAndPathArr := strings.Split(volNamesAndPath, ":")

				if strings.Contains(volNamesAndPathArr[0], "myvol") && volNamesAndPathArr[1] == "/data" {
					volumesMatched = true
				}
			}
			Expect(volumesMatched).To(Equal(true))
		})

	})

})
