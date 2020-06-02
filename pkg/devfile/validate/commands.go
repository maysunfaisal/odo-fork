package validate

import (
	"fmt"

	"github.com/openshift/odo/pkg/devfile/parser/data/common"
)

// Errors
var (
	ErrorNoCommands = "no commands present"
	// ErrorNoContainerComponent = fmt.Sprintf("odo requires atleast one component of type '%s' in devfile", common.ContainerComponentType)
)

// ValidateCommands validates all the devfile commands
func ValidateCommands(commands []common.DevfileCommand) error {

	// components cannot be empty
	if len(commands) < 1 {
		return fmt.Errorf(ErrorNoCommands)
	}

	// for _, command := range commands {
	// 	if command.Exec != nil && command.Exec.Group != nil
	// }

	// Successful
	return nil
}
