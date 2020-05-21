package version100

import (
	"github.com/openshift/odo/pkg/devfile/parser/data/common"
	"strings"
)

func (d *Devfile100) GetMetadata() common.DevfileMetadata {
	// No GenerateName field in V2
	return common.DevfileMetadata{
		Name: d.Metadata.Name,
		//Version: No field in V1
	}
}

/// GetComponents returns the slice of DevfileComponent objects parsed from the Devfile
func (d *Devfile100) GetComponents() []common.DevfileComponent {
	var comps []common.DevfileComponent
	for _, v := range d.Components {
		comps = append(comps, convertV1ComponentToCommon(v))
	}
	return comps
}

// GetAliasedComponents returns the slice of DevfileComponent objects that each have an alias
func (d *Devfile100) GetAliasedComponents() []common.DevfileComponent {
	// TODO(adi): we might not need this for V2 as name is a required field now.
	var comps []common.DevfileComponent
	for _, v := range d.Components {
		comps = append(comps, convertV1ComponentToCommon(v))
	}

	var aliasedComponents = []common.DevfileComponent{}
	for _, comp := range comps {
		if comp.Container != nil {
			if comp.Container.Name != "" {
				aliasedComponents = append(aliasedComponents, comp)
			}
		}
	}
	return aliasedComponents
}

// GetProjects returns the slice of DevfileProject objects parsed from the Devfile
func (d *Devfile100) GetProjects() []common.DevfileProject {

	var projects []common.DevfileProject
	for _, v := range d.Projects {
		// We are only supporting ProjectType git in V1
		if v.Source.Type == ProjectTypeGit {
			projects = append(projects, convertV1ProjectToCommon(v))
		}
	}

	return projects
}

// GetCommands returns the slice of DevfileCommand objects parsed from the Devfile
func (d *Devfile100) GetCommands() []common.DevfileCommand {

	var commands []common.DevfileCommand
	for _, v := range d.Commands {
		cmd := convertV1CommandToCommon(v)

		commands = append(commands, cmd)
	}

	return commands
}

func (d *Devfile100) GetParent() common.DevfileParent {
	return common.DevfileParent{}

}

func (d *Devfile100) GetEvents() common.DevfileEvents {
	return common.DevfileEvents{}

}

func convertV1CommandToCommon(c Command) (d common.DevfileCommand) {
	var exec common.Exec

	for _, action := range c.Actions {

		if action.Type == DevfileCommandTypeExec {
			exec = common.Exec{
				Attributes:  c.Attributes,
				CommandLine: action.Command,
				Component:   action.Component,
				Group:       getGroup(c.Name),
				Id:          c.Name,
				WorkingDir:  action.Workdir,
				// Env:
				// Label:
			}
		}

	}

	// TODO: Previewurl
	return common.DevfileCommand{
		//TODO(adi): Type
		Exec: &exec,
	}
}

func convertV1ComponentToCommon(c Component) (component common.DevfileComponent) {

	var endpoints []common.Endpoint
	for _, v := range c.ComponentDockerimage.Endpoints {
		endpoints = append(endpoints, convertV1EndpointsToCommon(v))
	}

	var envs []common.Env
	for _, v := range c.ComponentDockerimage.Env {
		envs = append(envs, convertV1EnvToCommon(v))
	}

	var volumes []common.VolumeMount
	for _, v := range c.ComponentDockerimage.Volumes {
		volumes = append(volumes, convertV1VolumeToCommon(v))
	}

	container := common.Container{
		Name:         c.Alias,
		Endpoints:    endpoints,
		Env:          envs,
		Image:        c.ComponentDockerimage.Image,
		MemoryLimit:  c.ComponentDockerimage.MemoryLimit,
		MountSources: c.MountSources,
		VolumeMounts: volumes,
		// SourceMapping: Not present in V1
	}

	component = common.DevfileComponent{Container: &container}

	return component
}

func convertV1EndpointsToCommon(e DockerimageEndpoint) common.Endpoint {
	return common.Endpoint{
		// Attributes:
		// Configuration:
		Name:       e.Name,
		TargetPort: e.Port,
	}
}

func convertV1EnvToCommon(e DockerimageEnv) common.Env {
	return common.Env{
		Name:  e.Name,
		Value: e.Value,
	}
}

func convertV1VolumeToCommon(v DockerimageVolume) common.VolumeMount {
	return common.VolumeMount{
		Name: v.Name,
		Path: v.ContainerPath,
	}
}

func convertV1ProjectToCommon(p Project) common.DevfileProject {

	git := common.Git{
		Branch:            p.Source.Branch,
		Location:          p.Source.Location,
		SparseCheckoutDir: p.Source.SparseCheckoutDir,
		StartPoint:        p.Source.StartPoint,
	}

	return common.DevfileProject{
		ClonePath:  p.ClonePath,
		Git:        &git,
		Name:       p.Name,
		SourceType: common.GitProjectSourceType,
	}

}

func getGroup(name string) *common.Group {
	group := common.Group{}

	switch strings.ToLower(name) {
	case "devrun":
		group.Kind = common.RunCommandGroupType
		group.IsDefault = true
	case "devbuild":
		group.Kind = common.BuildCommandGroupType
		group.IsDefault = true
	case "devinit":
		group.Kind = common.InitCommandGroupType
		group.IsDefault = true
	}

	return &group
}
