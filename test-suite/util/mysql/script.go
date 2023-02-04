// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package mysql

import (
	"github.com/marinesovitch/ote/test-suite/util/common"
	"github.com/marinesovitch/ote/test-suite/util/k8s"
)

func LoadScript(namespace string, podName string, containerId k8s.ContainerId, script string) error {
	kubectl := k8s.Kubectl{}
	return kubectl.ExecuteWithInput(script, namespace, podName, containerId,
		"-i", "--", "mysql", "-u"+common.RootUser, "-p"+common.RootPassword)
}
