// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package container

import (
	"strings"

	"github.com/marinesovitch/ote/test-suite/util/auxi"
	"github.com/marinesovitch/ote/test-suite/util/system"
)

type docker_podman struct {
	executable string
}

func (dp docker_podman) run(args ...string) error {
	return system.Execute(dp.executable, args...)
}

func (dp docker_podman) runGetOutput(args ...string) (string, error) {
	return system.ExecuteGetOutput(dp.executable, args...)
}

func (dp docker_podman) getNetworks() ([]string, error) {
	networks, err := dp.runGetOutput("network", "ls", "--format", "{{.Name}}")
	if err != nil {
		return nil, err
	}

	return strings.Split(networks, "\n"), nil
}

func (dp docker_podman) DoesNetworkExist(network string) (bool, error) {
	networks, err := dp.getNetworks()
	if err != nil {
		return false, err
	}

	return auxi.Contains(networks, network), nil
}

func (dp docker_podman) IsNetworkConnectedTo(network string, container string) (bool, error) {
	connectedContainers, err := dp.runGetOutput("network", "inspect", "--format", "{{range .Containers}} {{.Name}} {{end}}", network)
	if err != nil {
		return false, err
	}

	containers := strings.Fields(connectedContainers)
	return auxi.Contains(containers, container), nil
}

func (dp docker_podman) ConnectNetwork(context string, container string) error {
	return dp.run("network", "connect", context, container)
}

func GetDocker() Engine {
	return docker_podman{executable: "docker"}
}

func GetPodman() Engine {
	return docker_podman{executable: "podman"}
}
