// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package k8s

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

type ContainerId int

const (
	FixDataDir ContainerId = iota
	InitConf
	InitMysql
	Sidecar
	Mysql
	Router
	UnknownContainer
)

var contIdToName = map[ContainerId]string{
	FixDataDir: "fixdatadir",
	InitConf:   "initconf",
	InitMysql:  "initmysql",
	Sidecar:    "sidecar",
	Mysql:      "mysql",
	Router:     "router",
}

func GetContainerName(contId ContainerId) string {
	if name, ok := contIdToName[contId]; ok {
		return name
	}
	panic(fmt.Errorf("unexpected containerId %d", contId))
}

var nameToContId = map[string]ContainerId{
	"fixdatadir": FixDataDir,
	"initconf":   InitConf,
	"initmysql":  InitMysql,
	"sidecar":    Sidecar,
	"mysql":      Mysql,
	"router":     Router,
}

func GetContainerId(name string) (ContainerId, error) {
	if contId, ok := nameToContId[name]; ok {
		return contId, nil
	}
	return UnknownContainer, fmt.Errorf("unexpected container name %s", name)
}

// ------------

type ContainerNotFoundError struct {
	error
}

// ------------

// func getContainerItem[Item *corev1.Container | *corev1.ContainerStatus](items []Item, name string) (Item, error) {
// 	for _, item := range items {
// 		if item.Name == name {
// 			return item, nil
// 		}
// 	}
// 	return nil, fmt.Errorf("container %s not found", name)
// }

func getContainer(containers []corev1.Container, contId ContainerId) (*corev1.Container, error) {
	name := GetContainerName(contId)
	for _, container := range containers {
		if container.Name == name {
			return &container, nil
		}
	}
	return nil, ContainerNotFoundError{error: fmt.Errorf("container of name %s not found", name)}
}

func GetContainer(pod *corev1.Pod, contId ContainerId) (*corev1.Container, error) {
	switch contId {
	case FixDataDir, InitConf, InitMysql:
		return getContainer(pod.Spec.InitContainers, contId)
	case Sidecar, Mysql, Router:
		return getContainer(pod.Spec.Containers, contId)
	default:
		return nil, fmt.Errorf("incorrect container id %d", contId)
	}
}

func GetContainerByName(pod *corev1.Pod, name string) (*corev1.Container, error) {
	contId, err := GetContainerId(name)
	if err != nil {
		return nil, err
	}
	return GetContainer(pod, contId)
}

// ------------

func getContainerStatus(containerStatuses []corev1.ContainerStatus, contId ContainerId) (*corev1.ContainerStatus, error) {
	name := GetContainerName(contId)
	for _, containerStatus := range containerStatuses {
		if containerStatus.Name == name {
			return &containerStatus, nil
		}
	}
	return nil, ContainerNotFoundError{error: fmt.Errorf("containerStatus of name %s not found", name)}
}

func GetContainerStatus(pod *corev1.Pod, contId ContainerId) (*corev1.ContainerStatus, error) {
	switch contId {
	case FixDataDir, InitConf, InitMysql:
		return getContainerStatus(pod.Status.InitContainerStatuses, contId)
	case Sidecar, Mysql, Router:
		return getContainerStatus(pod.Status.ContainerStatuses, contId)
	default:
		return nil, fmt.Errorf("incorrect container id %d", contId)
	}
}

func GetContainerStatusByName(pod *corev1.Pod, name string) (*corev1.ContainerStatus, error) {
	contId, err := GetContainerId(name)
	if err != nil {
		return nil, err
	}
	return GetContainerStatus(pod, contId)
}

// ------------

func GetContainerState(pod *corev1.Pod, contId ContainerId) (*corev1.ContainerState, error) {
	cs, err := GetContainerStatus(pod, contId)
	if err != nil {
		return nil, err
	}
	return &cs.State, nil
}

// ------------

type ContainerInfo struct {
	Container *corev1.Container
	Status    *corev1.ContainerStatus
}

func GetContainerInfo(pod *corev1.Pod, name string) (*ContainerInfo, error) {
	contId, err := GetContainerId(name)
	if err != nil {
		return nil, err
	}

	cont, err := GetContainer(pod, contId)
	if err != nil {
		return nil, err
	}

	status, err := GetContainerStatus(pod, contId)
	if err != nil {
		return nil, err
	}

	return &ContainerInfo{Container: cont, Status: status}, nil
}
