// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package suite

import (
	"fmt"

	"github.com/marinesovitch/ote/test-suite/util/k8s"
)

func CheckRouterPods(client *k8s.Client, namespace string, name string, expectedPodsNum int) error {
	pattern := fmt.Sprintf("%s-router-.*", name)
	pods, err := client.ListPodsWithFilter(namespace, pattern)
	if err != nil {
		return err
	}

	routerPodsNum := len(pods.Items)
	if routerPodsNum != expectedPodsNum {
		return fmt.Errorf("there are %d router pods but expected %d", routerPodsNum, expectedPodsNum)
	}

	return nil
}
