// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package k8s

import (
	"fmt"
	"strings"

	"github.com/marinesovitch/ote/test-suite/util/auxi"
	corev1 "k8s.io/api/core/v1"
)

type PodState int

const (
	PodInitImagePullIssueError PodState = iota
	PodInitCreateContainerConfigError
	PodUnknownState
)

var initImagePullIssueError = []string{"ImagePullBackOff", "ErrImageNeverPull", "ErrImagePull"}

const initCreateContainerConfigErrorReason = "CreateContainerConfigError"

func isContainerInitImagePullIssueError(pod *corev1.Pod, contId ContainerId) (bool, error) {
	contState, err := GetContainerState(pod, contId)
	if err != nil {
		if _, ok := err.(ContainerNotFoundError); ok {
			return false, nil
		}
		return false, err
	}

	contStateWaiting := contState.Waiting
	if contStateWaiting == nil {
		return false, nil
	}

	return auxi.Contains(initImagePullIssueError, contStateWaiting.Reason), nil
}

func IsPodInitImagePullIssueError(pod *corev1.Pod) (bool, error) {
	if pod.Status.Phase != corev1.PodPending {
		return false, nil
	}

	if found, err := isContainerInitImagePullIssueError(pod, FixDataDir); found || err != nil {
		return found, err
	}

	return isContainerInitImagePullIssueError(pod, InitMysql)
}

func IsPodInitCreateContainerConfigError(pod *corev1.Pod) (bool, error) {
	if pod.Status.Phase != corev1.PodPending {
		return false, nil
	}

	initMysqlState, err := GetContainerState(pod, InitMysql)
	if err != nil {
		if _, ok := err.(ContainerNotFoundError); ok {
			return false, nil
		}
		return false, err
	}

	initMysqlStateWaiting := initMysqlState.Waiting
	return initMysqlStateWaiting != nil &&
		initMysqlStateWaiting.Reason == initCreateContainerConfigErrorReason, nil
}

func GetPodStateDescription(podState PodState) string {
	switch podState {
	case PodInitImagePullIssueError:
		return strings.Join(initImagePullIssueError, ",")
	case PodInitCreateContainerConfigError:
		return initCreateContainerConfigErrorReason
	default:
		panic(fmt.Errorf("unknown pod state: %d", podState))
	}
}
