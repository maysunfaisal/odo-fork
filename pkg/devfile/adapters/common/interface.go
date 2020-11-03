package common

import (
	"io"

	corev1 "k8s.io/api/core/v1"
)

// ComponentAdapter defines the functions that platform-specific adapters must implement
type ComponentAdapter interface {
	commandExecutor
	Push(parameters PushParameters) error
	DoesComponentExist(cmpName string) (bool, error)
	Delete(labels map[string]string, show bool) error
	Test(testCmd string, show bool) error
	StartContainerStatusWatch()
	StartSupervisordCtlStatusWatch()
	Log(follow, debug bool) (io.ReadCloser, error)
	Exec(command []string) error
}

// StorageAdapter defines the storage functions that platform-specific adapters must implement
type StorageAdapter interface {
	Create(map[corev1.Volume]*corev1.PersistentVolumeClaimSpec) error
}
