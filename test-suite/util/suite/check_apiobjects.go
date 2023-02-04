// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package suite

import (
	"fmt"
	"strings"
	"time"

	"github.com/marinesovitch/ote/test-suite/util/auxi"
	"github.com/marinesovitch/ote/test-suite/util/k8s"
	"github.com/marinesovitch/ote/test-suite/util/setup"

	"errors"

	corev1 "k8s.io/api/core/v1"
)

func checkPodLabels(pod *corev1.Pod, cluster string, role string) error {
	labels := pod.GetLabels()
	if labels["component"] != "mysqld" {
		return fmt.Errorf("pod component label is '%s' but expected 'mysqld'", labels["component"])
	}
	if labels["tier"] != "mysql" {
		return fmt.Errorf("pod tier label is '%s' but expected 'mysql'", labels["tier"])
	}
	if labels["mysql.oracle.com/cluster"] != cluster {
		return fmt.Errorf("pod cluster label is '%s' but expected '%s'", labels["mysql.oracle.com/cluster"], cluster)
	}
	if labels["mysql.oracle.com/cluster-role"] != role {
		return fmt.Errorf("pod cluster-role label is '%s' but expected '%s'", labels["mysql.oracle.com/cluster-role"], role)
	}
	return nil
}

// Check internal sanity of the object
func checkClusterObject(client *k8s.Client, icobj *k8s.InnoDBCluster, name string) error {
	icName := icobj.GetName()
	if icName != name {
		return fmt.Errorf("expected ic %s but got %s", name, icName)
	}

	// check for expected finalizers
	finalizers := icobj.GetFinalizers()
	const ClusterFinalizer = "mysql.oracle.com/cluster"
	if !auxi.Contains(finalizers, ClusterFinalizer) {
		return fmt.Errorf("ic finalizer '%s', not found in '%v'", ClusterFinalizer, finalizers)
	}

	// server statefulset
	sts, err := client.GetStatefulSet(icobj.GetNamespace(), icName)
	if err != nil {
		return err
	}
	if sts == nil {
		return fmt.Errorf("ic %s is incorrect (nil)", icName)
	}

	// cluster router deployment
	clusterRouterName := icName + "-router"
	hasRouter, err := client.HasDeployment(icobj.GetNamespace(), clusterRouterName)
	if err != nil {
		return err
	}
	if icobj.HasField("spec", "router", "instances") && icobj.GetInt("spec", "router", "instances") > 0 {
		if !hasRouter {
			return fmt.Errorf("deployment %s is not found but one is expected to exist", clusterRouterName)
		}
	} else {
		if hasRouter {
			return fmt.Errorf("deployment %s is found while it is expected not to exist", clusterRouterName)
		}
	}

	// main router service
	svc, err := client.GetService(icobj.GetNamespace(), icName)
	if err != nil {
		return err
	}
	if svc == nil {
		return errors.New("main router service is incorrect (nil)")
	}

	// direct server service
	svc, err = client.GetService(icobj.GetNamespace(), icName+"-instances")
	if err != nil {
		return err
	}
	if svc == nil {
		return errors.New("direct server service is incorrect (nil)")
	}

	return nil
}

func checkPodObject(pod *corev1.Pod, name string) error {
	if pod.GetName() != name {
		return fmt.Errorf("expected pod %s but got %s", name, pod.GetName())
	}

	finalizers := pod.GetFinalizers()
	const MembershipFinalizer = "mysql.oracle.com/membership"
	if !auxi.Contains(finalizers, MembershipFinalizer) {
		return fmt.Errorf("pod finalizer '%s', not found in '%v'", MembershipFinalizer, finalizers)
	}

	return nil
}

func checkClusterSpecCompliant(cfg *setup.Configuration, client *k8s.Client, icobj *k8s.InnoDBCluster) error {
	icName := icobj.GetName()

	// server statefulset
	sts, err := client.GetStatefulSet(icobj.GetNamespace(), icName)
	if err != nil {
		return err
	}

	stsSpecReplicas := int(*sts.Spec.Replicas)
	icSpecInstances := icobj.GetInt("spec", "instances")
	if stsSpecReplicas != icSpecInstances {
		return fmt.Errorf("stsSpecReplicas (%d) != icSpecInstances (%d)", stsSpecReplicas, icSpecInstances)
	}

	// router replicaset
	clusterRouterName := icName + "-router"
	hasRouter, err := client.HasDeployment(icobj.GetNamespace(), clusterRouterName)
	if err != nil {
		return err
	}
	if icobj.HasField("spec", "router", "instances") && icobj.GetInt("spec", "router", "instances") > 0 {
		if !hasRouter {
			return errors.New("incorrect router deployment (nil)")
		}

		clusterRouter, err := client.GetDeployment(icobj.GetNamespace(), clusterRouterName)
		if err != nil {
			return err
		}

		routerSpecReplicas := int(*clusterRouter.Spec.Replicas)
		icSpecRouterInstances := icobj.GetInt("spec", "router", "instances")
		if routerSpecReplicas != icSpecRouterInstances {
			return fmt.Errorf("router %s spec.replicas (%d) != ic %s spec.router.instances (%d)", clusterRouterName, routerSpecReplicas, icName, icSpecRouterInstances)
		}
	} else {
		if hasRouter {
			return fmt.Errorf("unexpected router deployment found (%s), it should not exist", clusterRouterName)
		}
	}

	// check actual pod count
	icOnlineInstances := icobj.GetInt("status", "cluster", "onlineInstances")
	if icOnlineInstances != icSpecInstances {
		return fmt.Errorf("icOnlineInstances (%d) != icSpecInstances (%d)", icOnlineInstances, icSpecInstances)
	}

	pods, err := client.ListPods(icobj.GetNamespace())
	if err != nil {
		return err
	}

	var serverPods []*corev1.Pod
	for _, pod := range pods.Items {
		podName := pod.GetName()
		if strings.HasPrefix(podName, icName+"-") && !strings.Contains(podName, "router") {
			serverPods = append(serverPods, &pod)
		}
	}
	if len(serverPods) != icSpecInstances {
		return fmt.Errorf("server pods number (%d) != icSpecInstances (%d)", len(serverPods), icSpecInstances)
	}

	var routerPods []*corev1.Pod
	for _, pod := range pods.Items {
		podName := pod.GetName()
		if strings.HasPrefix(podName, icName+"-router-") {
			routerPods = append(routerPods, &pod)
		}
	}
	icSpecRouterInstances := icobj.GetInt("spec", "router", "instances")
	if len(routerPods) != icSpecRouterInstances {
		return fmt.Errorf("router pods number (%d) != icSpecRouterInstances (%d)", len(routerPods), icSpecRouterInstances)
	}

	// icStatusVersion := icobj.GetString("status", "version")
	// if icobj.HasField("spec", "version") {
	// 	isSpecImageVersion := icobj.GetString("spec", "version")
	// 	if isSpecImageVersion != icStatusVersion {
	// 		return fmt.Errorf("image version in spec (%s) different than in status (%s)", isSpecImageVersion, icStatusVersion)
	// 	}
	// } else {
	// 	defaultServerVersionTag := cfg.Images.DefaultServerVersionTag
	// 	if icStatusVersion != defaultServerVersionTag {
	// 		return fmt.Errorf("image version in spec is not declared, so it should be %s but in status it is %s", defaultServerVersionTag, icStatusVersion)
	// 	}
	// }
	return nil
}

func checkPodSpecCompliant(icobj *k8s.InnoDBCluster, pod *corev1.Pod) error {
	// check that the spec of the pod complies with the cluster spec or
	// hardcoded/expected values
	icName := icobj.GetName()
	spec := pod.Spec

	const ExpectedTerminationGracePeriodSeconds = 30
	if *spec.TerminationGracePeriodSeconds != ExpectedTerminationGracePeriodSeconds {
		return fmt.Errorf("TerminationGracePeriodSeconds is %d but expected %d",
			*spec.TerminationGracePeriodSeconds, ExpectedTerminationGracePeriodSeconds)
	}

	if spec.RestartPolicy != corev1.RestartPolicyAlways {
		return fmt.Errorf("RestartPolicy is %s but expected %s", spec.RestartPolicy, corev1.RestartPolicyAlways)
	}

	specSubdomain := spec.Subdomain
	icSubdomain := icName + "-instances"
	if specSubdomain != icSubdomain {
		return fmt.Errorf("subdomain is %s but expected %s", specSubdomain, icSubdomain)
	}

	mysqlCont, err := k8s.GetContainer(pod, k8s.Mysql)
	if err != nil {
		return err
	}

	// check imagePull stuff
	if icobj.HasField("spec", "imagePullPolicy") {
		specPullPolicy := icobj.GetString("spec", "imagePullPolicy")
		mysqlContPullPolicy := string(mysqlCont.ImagePullPolicy)
		if specPullPolicy != mysqlContPullPolicy {
			return fmt.Errorf("ic %s spec imagePullPolicy (%s) is different than mysql container imagePullPolicy (%s)",
				icName, specPullPolicy, mysqlContPullPolicy)
		}
	}

	if icobj.HasField("spec", "imagePullSecrets") {
		specPullSecrets := icobj.GetSliceOfStrings("spec", "imagePullSecrets")

		podSpecImagePullSecrets := pod.Spec.ImagePullSecrets
		podSpecPullSecretNames := make([]string, len(podSpecImagePullSecrets))
		for i, podSpecPullSecret := range podSpecImagePullSecrets {
			podSpecPullSecretNames[i] = podSpecPullSecret.Name
		}

		if !auxi.AreStringSlicesEqual(specPullSecrets, podSpecPullSecretNames) {
			return fmt.Errorf("ic %s spec imagePullSecrets %q is different than mysql container imagePullSecrets %q",
				icName, specPullSecrets, podSpecPullSecretNames)
		}
	}

	return nil
}

func checkPodCondition(podCondition corev1.PodCondition) error {
	if podCondition.Status == corev1.ConditionTrue {
		return nil
	}

	// allow a pod not to be ready yet due to specified reason
	if podCondition.Type == "Ready" && podCondition.Reason == "ReadinessGatesNotReady" {
		return nil
	}

	return fmt.Errorf("condition %s status should be %s but got %s, reason %s",
		podCondition.Type, corev1.ConditionTrue, podCondition.Status, podCondition.Reason)
}

// Check pod status (containers etc)
func checkOnlinePodStatus(pod *corev1.Pod, restartsExpected bool) error {
	status := pod.Status

	if status.Phase != corev1.PodRunning {
		return fmt.Errorf("pod status phase should be %s but it is %s", corev1.PodRunning, status.Phase)
	}

	// all conditions true
	const expectedStatusConditionsNum = 6
	statusConditionsNum := len(status.Conditions)
	if statusConditionsNum != expectedStatusConditionsNum {
		return fmt.Errorf("expected conditions is %d but pod status has %d", expectedStatusConditionsNum, statusConditionsNum)
	}

	for _, cond := range status.Conditions {
		if err := checkPodCondition(cond); err != nil {
			return err
		}
	}

	initContainerStatusesNum := len(status.InitContainerStatuses)
	const expectedInitContainerStatusesNum = 3
	if initContainerStatusesNum != expectedInitContainerStatusesNum {
		return fmt.Errorf("expected %d init container status(es) but got %d", expectedInitContainerStatusesNum, initContainerStatusesNum)
	}

	initCont := status.InitContainerStatuses[1]
	expectedInitContainer := k8s.GetContainerName(k8s.InitConf)
	if initCont.Name != expectedInitContainer {
		return fmt.Errorf("expected init container name is %s but got %s", expectedInitContainer, initCont.Name)
	}

	// should be ready and no restarts expected
	containerStatusesNum := len(status.InitContainerStatuses)
	const expectedContainerStatusesNum = 2
	if containerStatusesNum != expectedInitContainerStatusesNum {
		return fmt.Errorf("expected %d container status(es) but got %d", expectedContainerStatusesNum, containerStatusesNum)
	}

	cont := status.ContainerStatuses[0]
	expectedContainer := k8s.GetContainerName(k8s.Mysql)
	if cont.Name != expectedContainer {
		return fmt.Errorf("expected container name is %s but got %s", expectedContainer, cont.Name)
	}
	if !cont.Ready {
		return fmt.Errorf("container %s should be ready", cont.Name)
	}
	if cont.RestartCount != 0 && !restartsExpected {
		return fmt.Errorf("container %s shouldn't restart but it restarted %d time(s)", cont.Name, cont.RestartCount)
	}
	if cont.State.Running == nil {
		return fmt.Errorf("container %s should has running state", cont.Name)
	}

	return nil
}

func getClusterObject(client *k8s.Client, namespace string, name string) (*k8s.InnoDBCluster, []*corev1.Pod, error) {
	icobj, err := client.GetInnoDBCluster(namespace, name)
	if err != nil {
		return nil, nil, err
	}
	if icobj.GetName() != name {
		return nil, nil, fmt.Errorf("expected innodbcluster %s but got %s", name, icobj.GetName())
	}
	if icobj.GetNamespace() != namespace {
		return nil, nil, fmt.Errorf("expected namespace %s but got %s", namespace, icobj.GetNamespace())
	}

	var mysqlPods []*corev1.Pod
	instanceCount := icobj.GetInt("spec", "instances")
	for i := 0; i < instanceCount; i++ {
		instanceName := fmt.Sprintf("%s-%d", name, i)
		instancePod, err := client.GetPod(namespace, instanceName)
		if err != nil {
			return nil, nil, err
		}
		if instancePod == nil {
			return nil, nil, fmt.Errorf("pod %s is incorrect (nil)", instanceName)
		}
		if instancePod.GetName() != instanceName {
			return nil, nil, fmt.Errorf("expected pod %s but got %s", instanceName, instancePod.GetName())
		}
		if instancePod.GetNamespace() != namespace {
			return nil, nil, fmt.Errorf("expected namespace %s but got %s", namespace, instancePod.GetNamespace())
		}

		mysqlPods = append(mysqlPods, instancePod)
	}

	return icobj, mysqlPods, nil
}

func checkOnlineCluster(cfg *setup.Configuration, client *k8s.Client, icobj *k8s.InnoDBCluster, restartsExpected bool, allowOthers bool) error {
	if err := checkClusterSpecCompliant(cfg, client, icobj); err != nil {
		return err
	}

	return checkClusterObject(client, icobj, icobj.GetName())
}

func checkOnlinePod(client *k8s.Client, icobj *k8s.InnoDBCluster, pod *corev1.Pod, restartsExpected bool, role string) error {
	if err := checkPodSpecCompliant(icobj, pod); err != nil {
		return err
	}

	if err := checkPodObject(pod, pod.GetName()); err != nil {
		return err
	}

	if err := checkOnlinePodStatus(pod, restartsExpected); err != nil {
		return err
	}

	// try a couple of times because the labels can take a while to get updated
	var err error
	for i := 0; i < 5; i++ {
		err = checkPodLabels(pod, icobj.GetName(), role)
		if err == nil {
			break
		}

		pod, err = client.GetPod(pod.GetNamespace(), pod.GetName())
		if err != nil {
			break
		}

		time.Sleep(2 * time.Second)
	}
	return err
}

func CheckPodContainer(client *k8s.Client, pod *corev1.Pod, containerId k8s.ContainerId, restarts int32, running bool) (*k8s.ContainerInfo, error) {
	containerName := k8s.GetContainerName(containerId)
	cont, err := k8s.GetContainerInfo(pod, containerName)
	if err != nil {
		return nil, err
	}

	if cont.Container.Name != containerName {
		return nil, fmt.Errorf("container name should be %s but got %s", containerName, cont.Container.Name)
	}
	if restarts != NoRestarts {
		if restarts != cont.Status.RestartCount {
			return nil, fmt.Errorf("container %s restarts count should be %d but got %d",
				containerName, restarts, cont.Status.RestartCount)
		}
	}
	if running && cont.Status.State.Running == nil {
		return nil, fmt.Errorf("container %s should be running but it doesn't", containerName)
	}
	return cont, nil
}

func checkRouterPod(pod *corev1.Pod, restarts int32) error {
	_, err := CheckPodContainer(nil, pod, k8s.Router, restarts, true)
	return err
}

// Check that the spec matches what we expect
func checkClusterSpec(icobj *k8s.InnoDBCluster, instances int, routers int) error {
	if instances > 0 {
		specInstances := icobj.GetInt("spec", "instances")
		if specInstances != instances {
			return fmt.Errorf("according to spec there are expected %d instance(s) but got %d", specInstances, instances)
		}
	}

	if routers != NoRouters && routers > 0 {
		if icobj.HasField("spec", "router", "instances") {
			specRouterInstances := icobj.GetInt("spec", "router", "instances")
			if specRouterInstances != routers {
				return fmt.Errorf("according to spec there are expected %d instance(s) but got %d", specRouterInstances, routers)
			}
		}
	}

	return nil
}
