// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package suite

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/marinesovitch/ote/test-suite/util/common"
	"github.com/marinesovitch/ote/test-suite/util/k8s"
	"github.com/marinesovitch/ote/test-suite/util/log"
	"github.com/marinesovitch/ote/test-suite/util/mysql"
	"github.com/marinesovitch/ote/test-suite/util/setup"

	corev1 "k8s.io/api/core/v1"
)

const yamlSubdir = "yaml"

type Unit struct {
	Cfg          setup.Configuration
	Client       *k8s.Client
	Namespace    string
	AuxNamespace string
	SuiteDir     string
}

func (u *Unit) Setup() error {
	err := u.WipeNamespace(u.Namespace)
	if err != nil {
		return err
	}

	if err := u.WipeNamespace(u.AuxNamespace); err != nil {
		return err
	}

	return u.Client.CreateNamespace(u.Namespace)
}

func (u *Unit) Teardown() error {
	if err := u.WipeNamespace(u.AuxNamespace); err != nil {
		return err
	}
	return u.WipeNamespace(u.Namespace)
}

func (u *Unit) GetServerImage(versionTag string) string {
	return fmt.Sprintf("%s/%s:%s", u.Cfg.GetImageRegistryRepository(), u.Cfg.Images.MysqlServerImage, versionTag)
}

func (u *Unit) GetDefaultServerImage() string {
	return u.GetServerImage(u.Cfg.Images.DefaultVersionTag)
}

func (u *Unit) GetRouterImage(versionTag string) string {
	return fmt.Sprintf("%s/%s:%s", u.Cfg.GetImageRegistryRepository(), u.Cfg.Images.MysqlRouterImage, versionTag)
}

func (u *Unit) GetDefaultRouterImage() string {
	return u.GetRouterImage(u.Cfg.Images.DefaultVersionTag)
}

func (u *Unit) GetOperatorImage(versionTag string) string {
	return fmt.Sprintf("%s/%s:%s", u.Cfg.GetImageRegistryRepository(), u.Cfg.Operator.Image, versionTag)
}

func (u *Unit) GetDefaultOperatorImage() string {
	return u.GetOperatorImage(u.Cfg.Operator.VersionTag)
}

func (u *Unit) ApplyInNamespace(namespace string, yamlFilename string) error {
	yamlPath := filepath.Join(u.SuiteDir, yamlSubdir, yamlFilename)
	return u.Client.Apply(namespace, yamlPath)
}

func (u *Unit) ApplyInNamespaceGetOutput(namespace string, yamlFilename string) (string, error) {
	yamlPath := filepath.Join(u.SuiteDir, yamlSubdir, yamlFilename)
	return u.Client.ApplyGetOutput(namespace, yamlPath)
}

func (u *Unit) Apply(yamlFilename string) error {
	return u.ApplyInNamespace(u.Namespace, yamlFilename)
}

func (u *Unit) ApplyGetOutput(yamlFilename string) (string, error) {
	return u.ApplyInNamespaceGetOutput(u.Namespace, yamlFilename)
}

func (u *Unit) Generate(yamlTemplateFilename string, data interface{}) (string, error) {
	yamlTemplatePath := filepath.Join(u.SuiteDir, yamlSubdir, yamlTemplateFilename)
	yamlPath, err := setup.GenerateFromGenericFile(&u.Cfg, yamlTemplatePath, data)
	if err != nil {
		return "", fmt.Errorf("cannot generate %s: %s", yamlTemplatePath, err)
	}
	return yamlPath, nil
}

func (u *Unit) GenerateAndApplyInNamespace(namespace string, yamlTemplateFilename string, data interface{}) error {
	yamlPath, err := u.Generate(yamlTemplateFilename, data)
	if err != nil {
		return err
	}
	return u.Client.Apply(namespace, yamlPath)
}

func (u *Unit) GenerateAndApply(yamlTemplateFilename string, data interface{}) error {
	return u.GenerateAndApplyInNamespace(u.Namespace, yamlTemplateFilename, data)
}

func (u *Unit) LoadScript(podName string, containerId k8s.ContainerId, script string) error {
	return mysql.LoadScript(u.Namespace, podName, containerId, script)
}

func (u *Unit) AssertGotClusterEvent(
	t *testing.T, cluster string, sinceResourceVersion string, evType string, reason string, msg string) error {

	rx, err := regexp.Compile(msg)
	if err != nil {
		return err
	}

	var events *corev1.EventList
	checker := func(args ...interface{}) (bool, error) {
		events, err = u.Client.ListInnoDBClusterEvents(u.Namespace, cluster, sinceResourceVersion)
		if err != nil {
			return false, err
		}
		for _, event := range events.Items {
			if event.Type == evType && event.Reason == reason && rx.MatchString(event.Message) {
				return true, nil
			}
		}
		return false, nil
	}
	ok, err := u.Wait(checker, 60, 3)
	if err != nil {
		return err
	}

	// if events != nil {
	// 	t.Logf("events for cluster %s", cluster)
	// 	for i, event := range events.Items {
	// 		t.Logf("event %d: %s, %s, %s", i, event.Type, event.Reason, event.Message)
	// 	}
	// }

	if !ok {
		return fmt.Errorf("event (%s, %s, %s) not found for '%s'", cluster, evType, reason, msg)
	}
	return nil
}

type ConditionChecker func(args ...interface{}) (bool, error)

func (u *Unit) Wait(checker ConditionChecker, timeout time.Duration, interval time.Duration, args ...interface{}) (bool, error) {
	timeout *= time.Second
	interval *= time.Second
	for elapsed := 0 * time.Second; elapsed < timeout; elapsed += interval {
		if result, err := checker(args...); result || err != nil {
			return result, err
		}
		time.Sleep(interval)
	}
	return false, errors.New("timeout waiting for condition")
}

func (u *Unit) GetInnoDBClusterResourceVersion(name string) (string, error) {
	ic, err := u.Client.GetInnoDBCluster(u.Namespace, name)
	if err != nil {
		return common.AnyResourceVersion, err
	}

	return ic.GetResourceVersion(), nil
}

func (u *Unit) WaitOnInnoDBCluster(params k8s.WaitOnInnoDBClusterParams) error {
	// Wait for given ic object to reach one of the states in the list.
	// Aborts on timeout or when an unexpected error is detected in the operator.
	if params.Namespace == "" {
		params.Namespace = u.Namespace
	}

	const DefaultICTimeout = 300
	if params.Timeout == 0 {
		params.Timeout = DefaultICTimeout
	}

	return u.Client.WaitOnInnoDBCluster(params)
}

func (u *Unit) WaitOnPodInNamespaceSince(namespace string, name string, sinceResourceVersion string, status corev1.PodPhase) error {
	// Wait for given pod object to reach one of the states in the list.
	// Aborts on timeout or when an unexpected error is detected in the operator.
	const Timeout = 120
	return u.Client.WaitOnPod(namespace, name, sinceResourceVersion, status, Timeout)
}

func (u *Unit) WaitOnPodInNamespace(namespace string, name string, status corev1.PodPhase) error {
	return u.WaitOnPodInNamespaceSince(namespace, name, common.AnyResourceVersion, status)
}

func (u *Unit) WaitOnPodSince(name string, sinceResourceVersion string, status corev1.PodPhase) error {
	return u.WaitOnPodInNamespaceSince(u.Namespace, name, sinceResourceVersion, status)
}

func (u *Unit) WaitOnPod(name string, status corev1.PodPhase) error {
	return u.WaitOnPodInNamespace(u.Namespace, name, status)
}

func (u *Unit) WaitOnRoutersInNamespace(namespace string, clusterName string, expectedNumOnline int) error {
	log.Info.Printf("Waiting for %d routers of the cluster %s/%s to become running", expectedNumOnline, namespace, clusterName)

	routerChecker := func(args ...interface{}) (bool, error) {
		routers, err := u.Client.ListPodsWithFilter(namespace, clusterName+"-router-.*")
		if err != nil {
			return false, err
		}

		if len(routers.Items) != expectedNumOnline {
			return false, err
		}

		for _, router := range routers.Items {
			if router.Status.Phase != corev1.PodRunning {
				return false, nil
			}
		}

		return true, nil
	}

	_, err := u.Wait(routerChecker, 120, 3)
	return err
}

func (u *Unit) WaitOnRouters(clusterName string, expectedNumOnline int) error {
	return u.WaitOnRoutersInNamespace(u.Namespace, clusterName, expectedNumOnline)
}

func (u *Unit) WaitOnInnoDBClusterGoneInNamespaceSince(namespace string, name string, sinceResourceVersion string) error {
	const Timeout = 120
	return u.Client.WaitOnInnoDBClusterGone(namespace, name, sinceResourceVersion, Timeout)
}

func (u *Unit) WaitOnInnoDBClusterGoneInNamespace(namespace string, name string) error {
	return u.WaitOnInnoDBClusterGoneInNamespaceSince(namespace, name, common.AnyResourceVersion)
}

func (u *Unit) WaitOnInnoDBClusterGone(name string) error {
	return u.WaitOnInnoDBClusterGoneInNamespace(u.Namespace, name)
}

func (u *Unit) WaitOnPodGoneInNamespaceSince(namespace string, name string, sinceResourceVersion string) error {
	const Timeout = 120
	return u.Client.WaitOnPodGone(namespace, name, sinceResourceVersion, Timeout)
}

func (u *Unit) WaitOnPodGoneInNamespace(namespace string, name string) error {
	return u.WaitOnPodGoneInNamespaceSince(namespace, name, common.AnyResourceVersion)
}

func (u *Unit) WaitOnPodGone(name string) error {
	return u.WaitOnPodGoneInNamespace(u.Namespace, name)
}

func (u *Unit) WaitOnRoutersGone(clusterName string) error {
	log.Info.Printf("Waiting for routers of the cluster %s/%s to gone", u.Namespace, clusterName)

	routerChecker := func(args ...interface{}) (bool, error) {
		routers, err := u.Client.ListPodsWithFilter(u.Namespace, clusterName+"-router-.*")
		if err != nil {
			return false, err
		}

		if len(routers.Items) > 0 {
			return false, err
		}

		return true, nil
	}

	_, err := u.Wait(routerChecker, 120, 3)
	return err
}

func (u *Unit) DeleteAllPersistentVolumeClaims() error {
	pvcs, err := u.Client.ListPersistentVolumeClaims(u.Namespace)
	if err != nil {
		return err
	}
	for _, pvc := range pvcs.Items {
		err = u.Client.DeletePersistentVolumeClaim(u.Namespace, pvc.GetName())
		if err != nil {
			return err
		}
	}
	return nil
}

func (u *Unit) GetDefaultCheckParams() CheckParams {
	return CheckParams{
		Client:    u.Client,
		Namespace: u.Namespace,
		Routers:   NoRouters,
		Primary:   NoPrimary,
		User:      common.RootUser,
		Password:  common.RootPassword,
	}
}

func (u *Unit) verifyNamespaceIsEmpty(namespace string) (string, error) {
	vnie := collectPendingItems{
		client:    u.Client,
		namespace: namespace,
	}

	return vnie.run()
}

func (u *Unit) waitNamespaceIsEmpty(namespace string) error {
	var pendingItems string
	checker := func(args ...interface{}) (bool, error) {
		var err error
		pendingItems, err = u.verifyNamespaceIsEmpty(namespace)
		return len(pendingItems) == 0, err
	}
	_, err := u.Wait(checker, 300, 10)
	if err != nil {
		if len(pendingItems) > 0 {
			return fmt.Errorf("%s: namespace %s is not empty: %s", err, namespace, pendingItems)
		}
		return err
	}
	return nil
}

func (u *Unit) WipeNamespace(namespace string) error {
	if hasNamespace, _, err := u.Client.HasNamespace(namespace); !hasNamespace || err != nil {
		return err
	}

	wn := wipeNamespace{u.Client, namespace}
	if err := wn.run(); err != nil {
		return err
	}

	return u.waitNamespaceIsEmpty(namespace)
}
