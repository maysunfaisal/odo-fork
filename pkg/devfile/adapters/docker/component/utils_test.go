package component

import (
	"reflect"
	"strings"
	"testing"

	"github.com/docker/go-connections/nat"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	adaptersCommon "github.com/openshift/odo/pkg/devfile/adapters/common"
	devfileParser "github.com/openshift/odo/pkg/devfile/parser"
	"github.com/openshift/odo/pkg/devfile/parser/data/common"
	versionsCommon "github.com/openshift/odo/pkg/devfile/parser/data/common"
	envinfo "github.com/openshift/odo/pkg/envinfo"
	"github.com/openshift/odo/pkg/lclient"
	"github.com/openshift/odo/pkg/testingutil"
)

func TestCreateComponent(t *testing.T) {

	testComponentName := "test"
	fakeClient := lclient.FakeNew()
	fakeErrorClient := lclient.FakeErrorNew()

	tests := []struct {
		name          string
		componentType versionsCommon.DevfileComponentType
		client        *lclient.Client
		wantErr       bool
	}{
		{
			name:          "Case 1: Invalid devfile",
			componentType: "",
			client:        fakeClient,
			wantErr:       true,
		},
		{
			name:          "Case 2: Valid devfile",
			componentType: versionsCommon.DevfileComponentTypeDockerimage,
			client:        fakeClient,
			wantErr:       false,
		},
		{
			name:          "Case 3: Valid devfile, docker client error",
			componentType: versionsCommon.DevfileComponentTypeDockerimage,
			client:        fakeErrorClient,
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devObj := devfileParser.DevfileObj{
				Data: testingutil.TestDevfileData{
					ComponentType: tt.componentType,
				},
			}

			adapterCtx := adaptersCommon.AdapterContext{
				ComponentName: testComponentName,
				Devfile:       devObj,
			}

			componentAdapter := New(adapterCtx, *tt.client)
			err := componentAdapter.createComponent()

			// Checks for unexpected error cases
			if !tt.wantErr == (err != nil) {
				t.Errorf("component adapter create unexpected error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

}

func TestUpdateComponent(t *testing.T) {

	fakeClient := lclient.FakeNew()
	fakeErrorClient := lclient.FakeErrorNew()

	tests := []struct {
		name          string
		componentType versionsCommon.DevfileComponentType
		componentName string
		client        *lclient.Client
		wantErr       bool
	}{
		{
			name:          "Case 1: Invalid devfile",
			componentType: "",
			componentName: "",
			client:        fakeClient,
			wantErr:       true,
		},
		{
			name:          "Case 2: Valid devfile",
			componentType: versionsCommon.DevfileComponentTypeDockerimage,
			componentName: "test",
			client:        fakeClient,
			wantErr:       false,
		},
		{
			name:          "Case 3: Valid devfile, docker client error",
			componentType: versionsCommon.DevfileComponentTypeDockerimage,
			componentName: "",
			client:        fakeErrorClient,
			wantErr:       true,
		},
		{
			name:          "Case 3: Odo component does not exist", // should create proj vols and pull image if its the case
			componentType: versionsCommon.DevfileComponentTypeDockerimage,
			componentName: "fakecomponent",
			client:        fakeClient,
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devObj := devfileParser.DevfileObj{
				Data: testingutil.TestDevfileData{
					ComponentType: tt.componentType,
				},
			}

			adapterCtx := adaptersCommon.AdapterContext{
				ComponentName: tt.componentName,
				Devfile:       devObj,
			}

			componentAdapter := New(adapterCtx, *tt.client)
			_, err := componentAdapter.updateComponent()

			// Checks for unexpected error cases
			if !tt.wantErr == (err != nil) {
				t.Errorf("component adapter update unexpected error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

}

func TestPullAndStartContainer(t *testing.T) {

	testComponentName := "test"
	testVolumeName := "projects"

	fakeClient := lclient.FakeNew()
	fakeErrorClient := lclient.FakeErrorNew()

	tests := []struct {
		name          string
		componentType versionsCommon.DevfileComponentType
		client        *lclient.Client
		mounts        []mount.Mount
		wantErr       bool
	}{
		{
			name:          "Case 1: Successfully start container, no mount",
			componentType: versionsCommon.DevfileComponentTypeDockerimage,
			client:        fakeClient,
			mounts:        []mount.Mount{},
			wantErr:       false,
		},
		{
			name:          "Case 2: Docker client error",
			componentType: versionsCommon.DevfileComponentTypeDockerimage,
			client:        fakeErrorClient,
			mounts:        []mount.Mount{},
			wantErr:       true,
		},
		{
			name:          "Case 3: Successfully start container, one mount",
			componentType: versionsCommon.DevfileComponentTypeDockerimage,
			client:        fakeClient,
			mounts: []mount.Mount{
				{
					Source: "test-vol",
					Target: "/path",
				},
			},
			wantErr: false,
		},
		{
			name:          "Case 4: Successfully start container, multiple mounts",
			componentType: versionsCommon.DevfileComponentTypeDockerimage,
			client:        fakeClient,
			mounts: []mount.Mount{
				{
					Source: "test-vol",
					Target: "/path",
				},
				{
					Source: "test-vol-two",
					Target: "/path-two",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devObj := devfileParser.DevfileObj{
				Data: testingutil.TestDevfileData{
					ComponentType: tt.componentType,
				},
			}

			adapterCtx := adaptersCommon.AdapterContext{
				ComponentName: testComponentName,
				Devfile:       devObj,
			}

			componentAdapter := New(adapterCtx, *tt.client)
			componentAdapter.projectVolumeName = testVolumeName
			err := componentAdapter.pullAndStartContainer(tt.mounts, adapterCtx.Devfile.Data.GetAliasedComponents()[0])

			// Checks for unexpected error cases
			if !tt.wantErr == (err != nil) {
				t.Errorf("component adapter create unexpected error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

}

func TestStartContainer(t *testing.T) {

	testComponentName := "test"
	testVolumeName := "projects"

	fakeClient := lclient.FakeNew()
	fakeErrorClient := lclient.FakeErrorNew()

	tests := []struct {
		name          string
		componentType versionsCommon.DevfileComponentType
		client        *lclient.Client
		mounts        []mount.Mount
		wantErr       bool
	}{
		{
			name:          "Case 1: Successfully start container, no mount",
			componentType: versionsCommon.DevfileComponentTypeDockerimage,
			client:        fakeClient,
			mounts:        []mount.Mount{},
			wantErr:       false,
		},
		{
			name:          "Case 2: Docker client error",
			componentType: versionsCommon.DevfileComponentTypeDockerimage,
			client:        fakeErrorClient,
			mounts:        []mount.Mount{},
			wantErr:       true,
		},
		{
			name:          "Case 3: Successfully start container, one mount",
			componentType: versionsCommon.DevfileComponentTypeDockerimage,
			client:        fakeClient,
			mounts: []mount.Mount{
				{
					Source: "test-vol",
					Target: "/path",
				},
			},
			wantErr: false,
		},
		{
			name:          "Case 4: Successfully start container, multiple mount",
			componentType: versionsCommon.DevfileComponentTypeDockerimage,
			client:        fakeClient,
			mounts: []mount.Mount{
				{
					Source: "test-vol",
					Target: "/path",
				},
				{
					Source: "test-vol-two",
					Target: "/path-two",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devObj := devfileParser.DevfileObj{
				Data: testingutil.TestDevfileData{
					ComponentType: tt.componentType,
				},
			}

			adapterCtx := adaptersCommon.AdapterContext{
				ComponentName: testComponentName,
				Devfile:       devObj,
			}

			componentAdapter := New(adapterCtx, *tt.client)
			componentAdapter.projectVolumeName = testVolumeName
			err := componentAdapter.startComponent(tt.mounts, adapterCtx.Devfile.Data.GetAliasedComponents()[0])

			// Checks for unexpected error cases
			if !tt.wantErr == (err != nil) {
				t.Errorf("component adapter create unexpected error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

}

func TestGenerateAndGetHostConfig(t *testing.T) {
	fakeClient := lclient.FakeNew()
	testComponentName := "test"
	componentType := versionsCommon.DevfileComponentTypeDockerimage

	endpointName := []string{"8080/tcp", "9090/tcp", "9080/tcp"}
	var endpointPort = []int32{8080, 9090, 9080}
	var expectPortNameMapping = map[nat.Port]string{
		nat.Port("8080/tcp"): "url1",
		nat.Port("9090/tcp"): "url2",
		nat.Port("9080/tcp"): "url3",
	}

	tests := []struct {
		name         string
		urlValue     []envinfo.EnvInfoURL
		expectResult nat.PortMap
		client       *lclient.Client
		endpoints    []versionsCommon.DockerimageEndpoint
	}{
		{
			name:         "Case 1: no port mappings",
			urlValue:     []envinfo.EnvInfoURL{},
			expectResult: nil,
			client:       fakeClient,
			endpoints:    []versionsCommon.DockerimageEndpoint{},
		},
		{
			name: "Case 2: only one port mapping",
			urlValue: []envinfo.EnvInfoURL{
				{Name: "url1", Port: 8080, ExposedPort: 65432},
			},
			expectResult: nat.PortMap{
				"8080/tcp": []nat.PortBinding{
					{
						HostIP:   LocalhostIP,
						HostPort: "65432",
					},
				},
			},
			client: fakeClient,
			endpoints: []versionsCommon.DockerimageEndpoint{
				{
					Name: &endpointName[0],
					Port: &endpointPort[0],
				},
			},
		},
		{
			name: "Case 3: multiple port mappings",
			urlValue: []envinfo.EnvInfoURL{
				{Name: "url1", Port: 8080, ExposedPort: 65432},
				{Name: "url2", Port: 9090, ExposedPort: 54321},
				{Name: "url3", Port: 9080, ExposedPort: 45678},
			},
			expectResult: nat.PortMap{
				"8080/tcp": []nat.PortBinding{
					{
						HostIP:   LocalhostIP,
						HostPort: "65432",
					},
				},
				"9090/tcp": []nat.PortBinding{
					{
						HostIP:   LocalhostIP,
						HostPort: "54321",
					},
				},
				"9080/tcp": []nat.PortBinding{
					{
						HostIP:   LocalhostIP,
						HostPort: "45678",
					},
				},
			},
			client: fakeClient,
			endpoints: []versionsCommon.DockerimageEndpoint{
				{
					Name: &endpointName[0],
					Port: &endpointPort[0],
				},
				{
					Name: &endpointName[1],
					Port: &endpointPort[1],
				},
				{
					Name: &endpointName[2],
					Port: &endpointPort[2],
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			devObj := devfileParser.DevfileObj{
				Data: testingutil.TestDevfileData{
					ComponentType: componentType,
				},
			}

			adapterCtx := adaptersCommon.AdapterContext{
				ComponentName: testComponentName,
				Devfile:       devObj,
			}

			esi, err := envinfo.NewEnvSpecificInfo("")
			if err != nil {
				t.Error(err)
			}
			for _, url := range tt.urlValue {
				err = esi.SetConfiguration("URL", url)
				if err != nil {
					t.Error(err)
				}
			}
			componentAdapter := New(adapterCtx, *tt.client)
			hostConfig, portURLNameMapping, err := componentAdapter.generateAndGetHostConfig(tt.endpoints)
			if err != nil {
				t.Error(err)
			}

			if len(hostConfig.PortBindings) != len(tt.expectResult) {
				t.Errorf("host config PortBindings length mismatch: actual value %v, expected value %v", len(hostConfig.PortBindings), len(tt.expectResult))
			}
			if len(hostConfig.PortBindings) != 0 {
				for key, value := range hostConfig.PortBindings {
					if tt.expectResult[key][0].HostIP != value[0].HostIP || tt.expectResult[key][0].HostPort != value[0].HostPort {
						t.Errorf("host config PortBindings mismatch: actual value %v, expected value %v", hostConfig.PortBindings, tt.expectResult)
					}
				}
			}
			if len(portURLNameMapping) != 0 {
				for key, value := range portURLNameMapping {
					if expectPortNameMapping[key] != value {
						t.Errorf("port and urlName mapping mismatch for port %v: actual value %v, expected value %v", key, value, expectPortNameMapping[key])
					}
				}
			}
			err = esi.DeleteEnvInfoFile()
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestExecDevfile(t *testing.T) {

	testComponentName := "test"
	componentType := versionsCommon.DevfileComponentTypeDockerimage
	command := "ls -la"
	workDir := "/tmp"
	component := "alias1"
	var actionType versionsCommon.DevfileCommandType = versionsCommon.DevfileCommandTypeExec

	containers := []types.Container{
		{
			ID: "someid",
			Labels: map[string]string{
				"alias": "somealias",
			},
		},
		{
			ID: "someid2",
			Labels: map[string]string{
				"alias": "somealias2",
			},
		},
	}

	fakeClient := lclient.FakeNew()
	fakeErrorClient := lclient.FakeErrorNew()

	tests := []struct {
		name                string
		client              *lclient.Client
		pushDevfileCommands []versionsCommon.DevfileCommand
		componentExists     bool
		wantErr             bool
	}{
		{
			name:   "Case 1: Successful devfile command exec of devbuild and devrun",
			client: fakeClient,
			pushDevfileCommands: []versionsCommon.DevfileCommand{
				{
					Name: "devrun",
					Actions: []versionsCommon.DevfileCommandAction{
						{
							Command:   &command,
							Workdir:   &workDir,
							Type:      &actionType,
							Component: &component,
						},
					},
				},
				{
					Name: "devbuild",
					Actions: []versionsCommon.DevfileCommandAction{
						{
							Command:   &command,
							Workdir:   &workDir,
							Type:      &actionType,
							Component: &component,
						},
					},
				},
			},
			componentExists: false,
			wantErr:         false,
		},
		{
			name:   "Case 2: Successful devfile command exec of devrun",
			client: fakeClient,
			pushDevfileCommands: []versionsCommon.DevfileCommand{
				{
					Name: "devrun",
					Actions: []versionsCommon.DevfileCommandAction{
						{
							Command:   &command,
							Workdir:   &workDir,
							Type:      &actionType,
							Component: &component,
						},
					},
				},
			},
			componentExists: true,
			wantErr:         false,
		},
		{
			name:                "Case 3: No devfile push commands should result in an err",
			client:              fakeClient,
			pushDevfileCommands: []versionsCommon.DevfileCommand{},
			componentExists:     false,
			wantErr:             true,
		},
		{
			name:   "Case 4: Unsuccessful devfile command exec of devrun",
			client: fakeErrorClient,
			pushDevfileCommands: []versionsCommon.DevfileCommand{
				{
					Name: "devrun",
					Actions: []versionsCommon.DevfileCommandAction{
						{
							Command:   &command,
							Workdir:   &workDir,
							Type:      &actionType,
							Component: &component,
						},
					},
				},
			},
			componentExists: true,
			wantErr:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			devObj := devfileParser.DevfileObj{
				Data: testingutil.TestDevfileData{
					ComponentType: componentType,
				},
			}

			adapterCtx := adaptersCommon.AdapterContext{
				ComponentName: testComponentName,
				Devfile:       devObj,
			}

			componentAdapter := New(adapterCtx, *tt.client)
			err := componentAdapter.execDevfile(tt.pushDevfileCommands, tt.componentExists, false, containers)
			if !tt.wantErr && err != nil {
				t.Errorf("TestExecDevfile error: unexpected error during executing devfile commands: %v", err)
			}
		})
	}
}

func TestInitRunContainerSupervisord(t *testing.T) {

	testComponentName := "test"
	componentType := versionsCommon.DevfileComponentTypeDockerimage

	containers := []types.Container{
		{
			ID: "someid",
			Labels: map[string]string{
				"alias": "somealias",
			},
		},
		{
			ID: "someid2",
			Labels: map[string]string{
				"alias": "somealias2",
			},
		},
	}

	fakeClient := lclient.FakeNew()
	fakeErrorClient := lclient.FakeErrorNew()

	tests := []struct {
		name      string
		client    *lclient.Client
		component string
		wantErr   bool
	}{
		{
			name:      "Case 1: Successful initialization of supervisord",
			client:    fakeClient,
			component: "somealias",
			wantErr:   false,
		},
		{
			name:      "Case 2: Unsuccessful initialization of supervisord",
			client:    fakeErrorClient,
			component: "somealias",
			wantErr:   true,
		},
		{
			name:      "Case 3: Unsuccessful initialization of supervisord with wrong component",
			client:    fakeErrorClient,
			component: "somealias123",
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			devObj := devfileParser.DevfileObj{
				Data: testingutil.TestDevfileData{
					ComponentType: componentType,
				},
			}

			adapterCtx := adaptersCommon.AdapterContext{
				ComponentName: testComponentName,
				Devfile:       devObj,
			}

			componentAdapter := New(adapterCtx, *tt.client)
			err := componentAdapter.initRunContainerSupervisord(tt.component, containers)
			if !tt.wantErr && err != nil {
				t.Errorf("TestInitRunContainerSupervisord error: unexpected error during init supervisord: %v", err)
			}
		})
	}
}

func TestCreateProjectVolumeIfReqd(t *testing.T) {
	fakeClient := lclient.FakeNew()
	fakeErrorClient := lclient.FakeErrorNew()

	tests := []struct {
		name           string
		componentName  string
		client         *lclient.Client
		wantVolumeName string
		wantErr        bool
	}{
		{
			name:           "Case 1: Volume does not exist",
			componentName:  "somecomponent",
			client:         fakeClient,
			wantVolumeName: projectSourceVolumeName + "-somecomponent",
			wantErr:        false,
		},
		{
			name:           "Case 2: Volume exist",
			componentName:  "test",
			client:         fakeClient,
			wantVolumeName: projectSourceVolumeName + "-test",
			wantErr:        false,
		},
		{
			name:           "Case 3: More than one project volume exist",
			componentName:  "duplicate",
			client:         fakeClient,
			wantVolumeName: "",
			wantErr:        true,
		},
		{
			name:           "Case 4: Client error",
			componentName:  "random",
			client:         fakeErrorClient,
			wantVolumeName: "",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devObj := devfileParser.DevfileObj{
				Data: testingutil.TestDevfileData{
					ComponentType: versionsCommon.DevfileComponentTypeDockerimage,
				},
			}

			adapterCtx := adaptersCommon.AdapterContext{
				ComponentName: tt.componentName,
				Devfile:       devObj,
			}

			componentAdapter := New(adapterCtx, *tt.client)
			volumeName, err := componentAdapter.createProjectVolumeIfReqd()
			if !tt.wantErr && err != nil {
				t.Errorf("TestCreateAndGetProjectVolume error: Unexpected error: %v", err)
			} else if !tt.wantErr && !strings.Contains(volumeName, tt.wantVolumeName) {
				t.Errorf("TestCreateAndGetProjectVolume error: project volume name did not match, expected: %v got: %v", tt.wantVolumeName, volumeName)
			}
		})
	}
}

func TestStartBootstrapSupervisordInitContainer(t *testing.T) {

	supervisordVolumeName := "supervisord"
	componentName := "myComponent"

	fakeClient := lclient.FakeNew()
	fakeErrorClient := lclient.FakeErrorNew()

	tests := []struct {
		name    string
		client  *lclient.Client
		wantErr bool
	}{
		{
			name:    "Case 1: Successfully create a bootstrap container",
			client:  fakeClient,
			wantErr: false,
		},
		{
			name:    "Case 2: Failed to create a bootstrap container ",
			client:  fakeErrorClient,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devObj := devfileParser.DevfileObj{
				Data: testingutil.TestDevfileData{
					ComponentType: versionsCommon.DevfileComponentTypeDockerimage,
				},
			}

			adapterCtx := adaptersCommon.AdapterContext{
				ComponentName: componentName,
				Devfile:       devObj,
			}

			componentAdapter := New(adapterCtx, *tt.client)
			err := componentAdapter.startBootstrapSupervisordInitContainer(supervisordVolumeName)
			if !tt.wantErr && err != nil {
				t.Errorf("TestStartBootstrapSupervisordInitContainer: unexpected error got: %v wanted: %v", err, tt.wantErr)
			}
		})
	}

}

func TestCreateAndInitSupervisordVolumeIfReqd(t *testing.T) {

	fakeClient := lclient.FakeNew()
	fakeErrorClient := lclient.FakeErrorNew()

	componentName := "myComponent"

	tests := []struct {
		name    string
		client  *lclient.Client
		wantErr bool
	}{
		{
			name:    "Case 1: Successfully create a bootstrap vol and container",
			client:  fakeClient,
			wantErr: false,
		},
		{
			name:    "Case 2: Failed to create a bootstrap vol and container ",
			client:  fakeErrorClient,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devObj := devfileParser.DevfileObj{
				Data: testingutil.TestDevfileData{
					ComponentType: versionsCommon.DevfileComponentTypeDockerimage,
				},
			}

			adapterCtx := adaptersCommon.AdapterContext{
				ComponentName: componentName,
				Devfile:       devObj,
			}

			componentAdapter := New(adapterCtx, *tt.client)
			volName, err := componentAdapter.createAndInitSupervisordVolumeIfReqd(false)
			if !tt.wantErr && err != nil {
				t.Errorf("TestCreateAndInitSupervisordVolume: unexpected error %v, wanted %v", err, tt.wantErr)
			} else if !tt.wantErr && !strings.Contains(volName, adaptersCommon.SupervisordVolumeName+"-"+componentName) {
				t.Errorf("TestCreateAndInitSupervisordVolume: unexpected supervisord vol name, expected: %v got: %v", adaptersCommon.SupervisordVolumeName, volName)
			}
		})
	}

}

func TestUpdateComponentWithSupervisord(t *testing.T) {

	command := "ls -la"
	component := "alias1"
	workDir := "/"
	emptyString := ""
	garbageString := "garbageString"
	validCommandType := common.DevfileCommandTypeExec
	supervisordVolumeName := "supervisordVolumeName"
	defaultWorkDirEnv := adaptersCommon.EnvOdoCommandRunWorkingDir
	defaultCommandEnv := adaptersCommon.EnvOdoCommandRun

	tests := []struct {
		name                  string
		commandActions        []common.DevfileCommandAction
		commandName           string
		comp                  common.DevfileComponent
		supervisordVolumeName string
		hostConfig            container.HostConfig
		wantHostConfig        container.HostConfig
		wantCommand           []string
		wantArgs              []string
		wantEnv               []common.DockerimageEnv
	}{
		{
			name: "Case 1: No component commands, args, env",
			commandActions: []common.DevfileCommandAction{
				{
					Command:   &command,
					Component: &component,
					Workdir:   &workDir,
					Type:      &validCommandType,
				},
			},
			commandName: emptyString,
			comp: common.DevfileComponent{
				Alias: &component,
				DevfileComponentDockerimage: common.DevfileComponentDockerimage{
					Command: []string{},
					Args:    []string{},
					Env:     []common.DockerimageEnv{},
				},
			},
			supervisordVolumeName: supervisordVolumeName,
			hostConfig:            container.HostConfig{},
			wantHostConfig: container.HostConfig{
				Mounts: []mount.Mount{
					{
						Type:   mount.TypeVolume,
						Source: supervisordVolumeName,
						Target: adaptersCommon.SupervisordMountPath,
					},
				},
			},
			wantCommand: []string{adaptersCommon.SupervisordBinaryPath},
			wantArgs:    []string{"-c", adaptersCommon.SupervisordConfFile},
			wantEnv: []common.DockerimageEnv{
				{
					Name:  &defaultWorkDirEnv,
					Value: &workDir,
				},
				{
					Name:  &defaultCommandEnv,
					Value: &command,
				},
			},
		},
		{
			name: "Case 2: Existing component command and no args, env",
			commandActions: []common.DevfileCommandAction{
				{
					Command:   &command,
					Component: &component,
					Workdir:   &workDir,
					Type:      &validCommandType,
				},
			},
			commandName: emptyString,
			comp: common.DevfileComponent{
				Alias: &component,
				DevfileComponentDockerimage: common.DevfileComponentDockerimage{
					Command: []string{"some", "command"},
					Args:    []string{},
					Env:     []common.DockerimageEnv{},
				},
			},
			supervisordVolumeName: supervisordVolumeName,
			hostConfig:            container.HostConfig{},
			wantHostConfig: container.HostConfig{
				Mounts: []mount.Mount{
					{
						Type:   mount.TypeVolume,
						Source: supervisordVolumeName,
						Target: adaptersCommon.SupervisordMountPath,
					},
				},
			},
			wantCommand: []string{"some", "command"},
			wantArgs:    []string{},
			wantEnv: []common.DockerimageEnv{
				{
					Name:  &defaultWorkDirEnv,
					Value: &workDir,
				},
				{
					Name:  &defaultCommandEnv,
					Value: &command,
				},
			},
		},
		{
			name: "Case 3: Existing component command and args and no env",
			commandActions: []common.DevfileCommandAction{
				{
					Command:   &command,
					Component: &component,
					Workdir:   &workDir,
					Type:      &validCommandType,
				},
			},
			commandName: emptyString,
			comp: common.DevfileComponent{
				Alias: &component,
				DevfileComponentDockerimage: common.DevfileComponentDockerimage{
					Command: []string{"some", "command"},
					Args:    []string{"some", "args"},
					Env:     []common.DockerimageEnv{},
				},
			},
			supervisordVolumeName: supervisordVolumeName,
			hostConfig:            container.HostConfig{},
			wantHostConfig: container.HostConfig{
				Mounts: []mount.Mount{
					{
						Type:   mount.TypeVolume,
						Source: supervisordVolumeName,
						Target: adaptersCommon.SupervisordMountPath,
					},
				},
			},
			wantCommand: []string{"some", "command"},
			wantArgs:    []string{"some", "args"},
			wantEnv: []common.DockerimageEnv{
				{
					Name:  &defaultWorkDirEnv,
					Value: &workDir,
				},
				{
					Name:  &defaultCommandEnv,
					Value: &command,
				},
			},
		},
		{
			name: "Case 4: Existing component command, args and env",
			commandActions: []common.DevfileCommandAction{
				{
					Command:   &command,
					Component: &component,
					Workdir:   &workDir,
					Type:      &validCommandType,
				},
			},
			commandName: emptyString,
			comp: common.DevfileComponent{
				Alias: &component,
				DevfileComponentDockerimage: common.DevfileComponentDockerimage{
					Command: []string{"some", "command"},
					Args:    []string{"some", "args"},
					Env: []common.DockerimageEnv{
						{
							Name:  &defaultWorkDirEnv,
							Value: &garbageString,
						},
						{
							Name:  &defaultCommandEnv,
							Value: &garbageString,
						},
					},
				},
			},
			supervisordVolumeName: supervisordVolumeName,
			hostConfig:            container.HostConfig{},
			wantHostConfig: container.HostConfig{
				Mounts: []mount.Mount{
					{
						Type:   mount.TypeVolume,
						Source: supervisordVolumeName,
						Target: adaptersCommon.SupervisordMountPath,
					},
				},
			},
			wantCommand: []string{"some", "command"},
			wantArgs:    []string{"some", "args"},
			wantEnv: []common.DockerimageEnv{
				{
					Name:  &defaultWorkDirEnv,
					Value: &garbageString,
				},
				{
					Name:  &defaultCommandEnv,
					Value: &garbageString,
				},
			},
		},
		{
			name: "Case 5: Existing host config, should append to it",
			commandActions: []common.DevfileCommandAction{
				{
					Command:   &command,
					Component: &component,
					Workdir:   &workDir,
					Type:      &validCommandType,
				},
			},
			commandName: emptyString,
			comp: common.DevfileComponent{
				Alias: &component,
				DevfileComponentDockerimage: common.DevfileComponentDockerimage{
					Command: []string{"some", "command"},
					Args:    []string{"some", "args"},
					Env: []common.DockerimageEnv{
						{
							Name:  &defaultWorkDirEnv,
							Value: &garbageString,
						},
						{
							Name:  &defaultCommandEnv,
							Value: &garbageString,
						},
					},
				},
			},
			supervisordVolumeName: supervisordVolumeName,
			hostConfig: container.HostConfig{
				Mounts: []mount.Mount{
					{
						Type:   mount.TypeVolume,
						Source: garbageString,
						Target: garbageString,
					},
				},
			},
			wantHostConfig: container.HostConfig{
				Mounts: []mount.Mount{
					{
						Type:   mount.TypeVolume,
						Source: supervisordVolumeName,
						Target: adaptersCommon.SupervisordMountPath,
					},
					{
						Type:   mount.TypeVolume,
						Source: garbageString,
						Target: garbageString,
					},
				},
			},
			wantCommand: []string{"some", "command"},
			wantArgs:    []string{"some", "args"},
			wantEnv: []common.DockerimageEnv{
				{
					Name:  &defaultWorkDirEnv,
					Value: &garbageString,
				},
				{
					Name:  &defaultCommandEnv,
					Value: &garbageString,
				},
			},
		},
		{
			name: "Case 6: Not a run command component",
			commandActions: []common.DevfileCommandAction{
				{
					Command:   &command,
					Component: &component,
					Workdir:   &workDir,
					Type:      &validCommandType,
				},
			},
			commandName: emptyString,
			comp: common.DevfileComponent{
				Alias: &garbageString,
				DevfileComponentDockerimage: common.DevfileComponentDockerimage{
					Command: []string{},
					Args:    []string{},
					Env:     []common.DockerimageEnv{},
				},
			},
			supervisordVolumeName: supervisordVolumeName,
			hostConfig:            container.HostConfig{},
			wantHostConfig: container.HostConfig{
				Mounts: []mount.Mount{},
			},
			wantCommand: []string{},
			wantArgs:    []string{},
			wantEnv:     []common.DockerimageEnv{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devObj := devfileParser.DevfileObj{
				Data: testingutil.TestDevfileData{
					CommandActions: tt.commandActions,
					ComponentType:  common.DevfileComponentTypeDockerimage,
				},
			}

			runCommand, err := adaptersCommon.GetRunCommand(devObj.Data, tt.commandName)
			if err != nil {
				t.Errorf("TestUpdateComponentWithSupervisord: error getting the run command")
			}

			updateComponentWithSupervisord(&tt.comp, runCommand, tt.supervisordVolumeName, &tt.hostConfig)

			// Check the container host config
			for _, containerHostConfigMount := range tt.hostConfig.Mounts {
				matched := false
				for _, wantHostConfigMount := range tt.wantHostConfig.Mounts {
					if reflect.DeepEqual(wantHostConfigMount, containerHostConfigMount) {
						matched = true
					}
				}

				if !matched {
					t.Errorf("TestUpdateComponentWithSupervisord: host configs source: %v target:%v do not match wanted host config", containerHostConfigMount.Source, containerHostConfigMount.Target)
				}
			}

			// Check the component command
			if !reflect.DeepEqual(tt.comp.Command, tt.wantCommand) {
				t.Errorf("TestUpdateComponentWithSupervisord: component commands dont match actual: %v wanted: %v", tt.comp.Command, tt.wantCommand)
			}

			// Check the component args
			if !reflect.DeepEqual(tt.comp.Args, tt.wantArgs) {
				t.Errorf("TestUpdateComponentWithSupervisord: component args dont match actual: %v wanted: %v", tt.comp.Args, tt.wantArgs)
			}

			// Check the component env
			for _, compEnv := range tt.comp.Env {
				matched := false
				for _, wantEnv := range tt.wantEnv {
					if reflect.DeepEqual(wantEnv, compEnv) {
						matched = true
					}
				}

				if !matched {
					t.Errorf("TestUpdateComponentWithSupervisord: component env dont match env: %v:%v not present in wanted list", *compEnv.Name, *compEnv.Value)
				}
			}

		})
	}

}
