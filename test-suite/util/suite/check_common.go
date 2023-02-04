// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package suite

import (
	"errors"
	"fmt"
	"strings"

	"github.com/marinesovitch/ote/test-suite/util/k8s"

	corev1 "k8s.io/api/core/v1"
)

func checkSidecarHealth(client *k8s.Client, namespace string, pod *corev1.Pod) error {
	logs, err := client.Logs(namespace, pod.GetName(), k8s.Sidecar)
	if err != nil {
		return err
	}
	// check that the sidecar is running and waiting for events
	if !strings.Contains(logs, "Starting Operator request handler...") {
		return errors.New("sidecar is not healthy")
	}
	return nil
}

type CheckParams struct {
	Client           *k8s.Client
	Namespace        string
	Name             string
	Instances        int
	Routers          int
	Primary          int
	CountSessions    bool
	RestartsExpected bool
	User             string
	Password         string
	SharedNamespace  bool
	Version          string
}

func CheckAll(unit *Unit, params CheckParams) ([]*corev1.Pod, error) {
	client := unit.Client
	namespace := params.Namespace
	name := params.Name
	instances := params.Instances
	routers := params.Routers
	primary := params.Primary
	countSessions := params.CountSessions
	restartsExpected := params.RestartsExpected
	user := params.User
	password := params.Password
	sharedNamespace := params.SharedNamespace
	version := params.Version

	icobj, allPods, err := getClusterObject(params.Client, namespace, name)
	if err != nil {
		return nil, err
	}

	if err := checkClusterSpec(icobj, instances, routers); err != nil {
		return nil, err
	}

	if err := checkOnlineCluster(&unit.Cfg, client, icobj, restartsExpected, sharedNamespace); err != nil {
		return nil, err
	}

	info, err := CheckGroup(icobj, allPods, user, password)
	if err != nil {
		return nil, err
	}
	if primary == NoPrimary {
		// detect primary from cluster
		primary = info["primary"]
	}

	for i, pod := range allPods {
		podName := pod.GetName()
		expectedPodName := fmt.Sprintf("%s-%d", name, i)
		if podName != expectedPodName {
			return nil, fmt.Errorf("expected a pod '%s' but got '%s'", expectedPodName, podName)
		}
		var role string
		if i == primary {
			role = "PRIMARY"
		} else {
			role = "SECONDARY"
		}

		if err := checkOnlinePod(client, icobj, pod, restartsExpected, role); err != nil {
			return nil, err
		}

		numSessions := NoNumSessions
		if countSessions {
			numSessions = 0
			if i == primary {
				// PRIMARY has the GR observer session
				numSessions += 1
			} else {
				numSessions = 0
			}
		}

		if len(version) > 0 {
			if !strings.HasSuffix(pod.Status.ContainerStatuses[0].Image, version) {
				return nil, fmt.Errorf("pod '%s' image is '%s' but should end with '%s'",
					pod.GetName(), pod.Status.ContainerStatuses[0].Image, version)
			}
		}

		if err := checkInstance(icobj, allPods, pod, i == primary, numSessions, version, user, password); err != nil {
			return nil, err
		}

		if err := checkSidecarHealth(client, namespace, pod); err != nil {
			return nil, err
		}
	}

	routerPods, err := client.ListPodsWithFilter(namespace, fmt.Sprintf("%s-router-.*", name))
	if err != nil {
		return nil, err
	}

	if routers != NoRouters {
		if len(routerPods.Items) != routers {
			return nil, fmt.Errorf("expected %d routers but got %d", routers, len(routerPods.Items))
		}
		for _, router := range routerPods.Items {
			routerStatusPhase := router.Status.Phase
			expectedRouterStatusPhase := corev1.PodRunning
			if routerStatusPhase != expectedRouterStatusPhase {
				return nil, fmt.Errorf("expected router in the phase %s but got %s", expectedRouterStatusPhase, routerStatusPhase)
			}

			routerPod, err := client.GetPod(namespace, router.GetName())
			if err != nil {
				return nil, err
			}
			if err := checkRouterPod(routerPod, NoRestarts); err != nil {
				return nil, err
			}
		}
	}

	return allPods, nil
}
