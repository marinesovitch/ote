// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package container

// OBSOLETE - to remove as we intend to use (local) registry instead of loading images

import (
	"errors"
	"strings"
)

type Engine interface {
	DoesNetworkExist(network string) (bool, error)
	IsNetworkConnectedTo(network string, container string) (bool, error)
	ConnectNetwork(context string, container string) error
}

// type Tool int

// const (
// 	Auto Tool = iota
// 	Docker
// 	Podman
// )

func GetEngine(name string) (Engine, error) {
	switch strings.ToLower(name) {
	case "docker":
		return GetDocker(), nil
	case "podman":
		return GetPodman(), nil
	default:
		return nil, errors.New("unknown container engine " + name)
	}
}
