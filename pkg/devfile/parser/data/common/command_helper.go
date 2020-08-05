package common

// GetID returns the ID of the command
func (dc DevfileCommand) GetID() string {
	if dc.Composite != nil {
		return dc.Composite.Id
	} else if dc.Exec != nil {
		return dc.Exec.Id
	}

	return ""
}

// GetGroup returns the group the command belongs to
func (dc DevfileCommand) GetGroup() *Group {
	if dc.Composite != nil {
		return dc.Composite.Group
	} else if dc.Exec != nil {
		return dc.Exec.Group
	}

	return nil
}

// GetExecComponent returns the component of the exec command
func (dc DevfileCommand) GetExecComponent() string {
	if dc.Exec != nil {
		return dc.Exec.Component
	}

	return ""
}

// GetExecCommandLine returns the command line of the exec command
func (dc DevfileCommand) GetExecCommandLine() string {
	if dc.Exec != nil {
		return dc.Exec.CommandLine
	}

	return ""
}

// GetExecWorkingDir returns the working dir of the exec command
func (dc DevfileCommand) GetExecWorkingDir() string {
	if dc.Exec != nil {
		return dc.Exec.WorkingDir
	}

	return ""
}

// IsComposite checks if the command is a composite command
func (dc DevfileCommand) IsComposite() bool {
	isComposite := false
	if dc.Composite != nil {
		isComposite = true
	}

	return isComposite
}
