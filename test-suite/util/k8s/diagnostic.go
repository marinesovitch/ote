package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8s_err "k8s.io/apimachinery/pkg/api/errors"
)

func IsNotFoundError(err error) bool {
	if se, ok := err.(*k8s_err.StatusError); ok {
		const NotFoundErrorCode = 404
		return se.Status().Code == NotFoundErrorCode
	}
	return false
}

func GetPodNames(pods *corev1.PodList) []string {
	names := make([]string, len(pods.Items))
	for i, pod := range pods.Items {
		names[i] = pod.Name
	}
	return names
}

func GetStsNames(stss *appsv1.StatefulSetList) []string {
	names := make([]string, len(stss.Items))
	for i, sts := range stss.Items {
		names[i] = sts.Name
	}
	return names
}
