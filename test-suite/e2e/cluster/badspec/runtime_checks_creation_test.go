// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package badspec_test

import (
	"testing"

	"github.com/marinesovitch/ote/test-suite/util/common"
	"github.com/marinesovitch/ote/test-suite/util/k8s"
	"github.com/marinesovitch/ote/test-suite/util/suite"

	corev1 "k8s.io/api/core/v1"
)

// spec errors checked by the operator, once the ic object was accepted
// by the admission controllers.
// In all cases:
// - the status of the ic should become ERROR
// - an event describing the error should be posted

// Also:
// - fixing the error should recover from error
// - deleting a cluster with an error should be possible

var unit_rcc *suite.Unit

func SetupClusterSpecRuntimeChecksCreation(t *testing.T) {
	// this also checks that the root user can be completely customized
	err := unit_rcc.Client.CreateUserSecrets(
		unit_rcc.Namespace, "mypwds", common.AdminUser, common.DefaultHost, common.AdminPassword)
	if err != nil {
		t.Fatal(err)
	}
}

func BadSecretDelete(t *testing.T) {
	// Checks:
	// - secret that doesn't exist
	// - cluster can be deleted after the failure
	const PodName = "mycluster-0"

	err := unit_rcc.Apply("bad-secret-delete.yaml")
	if err != nil {
		t.Fatal(err)
	}

	err = unit_rcc.WaitOnPod(PodName, corev1.PodPending)
	if err != nil {
		t.Fatal(err)
	}

	// the initmysql container will fail during creation with
	// CreateContainerConfigError because the container is setup to read from
	// it to set MYSQL_ROOT_PASSWORD, so the operator or sidecars will never
	// run
	checker := func(args ...interface{}) (bool, error) {
		pod, err := unit_rcc.Client.GetPod(unit_rcc.Namespace, PodName)
		if err != nil {
			return false, err
		}
		return k8s.IsPodInitCreateContainerConfigError(pod)
	}
	updateFailed, err := unit_rcc.Wait(checker, 100, 4)
	if err != nil {
		t.Fatal(err)
	}
	if !updateFailed {
		t.Fatalf("after update the expected status of pod '%s' is '%s'",
			PodName, k8s.GetPodStateDescription(k8s.PodInitCreateContainerConfigError))
	}

	err = unit_rcc.Client.DeleteInnoDBCluster(unit_rcc.Namespace, "mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcc.WaitOnPodGone(PodName)
	if err != nil {
		t.Error(err)
	}

	err = unit_rcc.WaitOnInnoDBClusterGone("mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcc.DeleteAllPersistentVolumeClaims()
	if err != nil {
		t.Error(err)
	}
}

func BadSecretRecover(t *testing.T) {
}

func UnsupportedVersionDelete(t *testing.T) {
	// Checks that setting an unsupported version is detected before any pods
	// are created and that the cluster can be deleted in that state.
	// create cluster with mostly default configs but a specific server version
	err := unit_rcc.Apply("unsupported-version-delete.yaml")
	if err != nil {
		t.Fatal(err)
	}

	checker := func(args ...interface{}) (bool, error) {
		icEvents, err := unit_rcc.Client.ListInnoDBClusterEvents(unit_rcc.Namespace, "mycluster", common.AnyResourceVersion)
		if err != nil {
			return false, err
		}
		return len(icEvents.Items) > 0, nil
	}
	_, err = unit_rcc.Wait(checker, 60, 2)
	if err != nil {
		t.Fatal(err)
	}

	// version is invalid/not supported, runtime check should prevent the
	// sts from being created
	pods, err := unit_rcc.Client.ListPods(unit_rcc.Namespace)
	if err != nil {
		t.Fatal(err)
	}
	if len(pods.Items) != 0 {
		t.Errorf("unexpected pods %v", k8s.GetPodNames(pods))
	}

	sts, err := unit_rcc.Client.ListStatefulSets(unit_rcc.Namespace)
	if err != nil {
		t.Fatal(err)
	}
	if len(sts.Items) != 0 {
		t.Errorf("unexpected sts %v", k8s.GetStsNames(sts))
	}

	// there should be an event for the cluster resource indicating the problem
	err = unit_rcc.AssertGotClusterEvent(t,
		"mycluster", common.AnyResourceVersion, "Error", "InvalidArgument", "version 5.7.30 must be between .*")
	if err != nil {
		t.Error(err)
	}

	// deleting the ic should work despite the error
	err = unit_rcc.Client.DeleteInnoDBCluster(unit_rcc.Namespace, "mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcc.WaitOnPodGone("mycluster-0")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcc.WaitOnInnoDBClusterGone("mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcc.DeleteAllPersistentVolumeClaims()
	if err != nil {
		t.Error(err)
	}
}

func UnsupportedVersionRecover(t *testing.T) {
	// Checks that setting an unsupported version is detected before any pods
	// are created and that the cluster can be recovered by fixing the version.

	// create cluster with mostly default configs but a specific server version
	err := unit_rcc.Apply("unsupported-version-recover.yaml")
	if err != nil {
		t.Fatal(err)
	}

	checker := func(args ...interface{}) (bool, error) {
		icEvents, err := unit_rcc.Client.ListInnoDBClusterEvents(unit_rcc.Namespace, "mycluster", common.AnyResourceVersion)
		if err != nil {
			return false, err
		}
		return len(icEvents.Items) > 0, nil
	}
	_, err = unit_rcc.Wait(checker, 60, 2)
	if err != nil {
		t.Fatal(err)
	}

	// fixing the version should let the cluster resume creation
	patch := k8s.JsonPatch{
		Operation: k8s.PatchReplace,
		Path:      "/spec/version",
		Value:     unit_rcc.Cfg.Images.DefaultVersionTag,
	}
	err = unit_rcc.Client.JSONPatchInnoDBCluster(unit_rcc.Namespace, "mycluster", patch)
	if err != nil {
		t.Error(err)
	}

	// check cluster ok now
	err = unit_rcc.WaitOnPod("mycluster-0", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 1,
	}
	err = unit_rcc.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	// cleanup
	err = unit_rcc.Client.DeleteInnoDBCluster(unit_rcc.Namespace, "mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcc.WaitOnPodGone("mycluster-0")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcc.WaitOnInnoDBClusterGone("mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcc.WaitOnRoutersGone("mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcc.DeleteAllPersistentVolumeClaims()
	if err != nil {
		t.Error(err)
	}
}

func BadPodDelete(t *testing.T) {
	// Checks that using a bad spec that fails at the pod can be deleted.
	// create cluster with mostly default configs but a specific option
	// that will be accepted by the runtime checks but will fail at pod
	// creation
	err := unit_rcc.Apply("bad-pod-delete.yaml")
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"PENDING"},
		ExpectedNumOnline: 0,
	}
	err = unit_rcc.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_rcc.WaitOnPod("mycluster-0", corev1.PodPending)
	if err != nil {
		t.Fatal(err)
	}

	pods, err := unit_rcc.Client.ListPods(unit_rcc.Namespace)
	if err != nil {
		t.Fatal(err)
	}
	if len(pods.Items) != 1 {
		t.Errorf("expected 1 pod but found %d %v", len(pods.Items), k8s.GetPodNames(pods))
	}

	sts, err := unit_rcc.Client.ListStatefulSets(unit_rcc.Namespace)
	if err != nil {
		t.Fatal(err)
	}
	if len(sts.Items) != 1 {
		t.Errorf("expected 1 sts but found %d %v", len(sts.Items), k8s.GetStsNames(sts))
	}

	const PodName = "mycluster-0"
	checkPodError := func(args ...interface{}) (bool, error) {
		pod, err := unit_rcc.Client.GetPod(unit_rcc.Namespace, PodName)
		if err != nil {
			return false, err
		}
		return k8s.IsPodInitImagePullIssueError(pod)
	}
	_, err = unit_rcc.Wait(checkPodError, 60, 2)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_rcc.Client.DeleteInnoDBCluster(unit_rcc.Namespace, "mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcc.WaitOnPodGone("mycluster-0")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcc.WaitOnInnoDBClusterGone("mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcc.DeleteAllPersistentVolumeClaims()
	if err != nil {
		t.Error(err)
	}
}

func BadPodRecover(t *testing.T) {
	t.Skip("TODO - may need a fix in operator")
	// Checks that using a bad spec that fails at the pod can be recovered.
	// create cluster with mostly default configs but a specific option
	// that will be accepted by the runtime checks but will fail at pod
	// creation
	err := unit_rcc.Apply("bad-pod-recover.yaml")
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"PENDING"},
		ExpectedNumOnline: 0,
	}

	err = unit_rcc.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_rcc.WaitOnPod("mycluster-0", corev1.PodPending)
	if err != nil {
		t.Fatal(err)
	}

	pods, err := unit_rcc.Client.ListPods(unit_rcc.Namespace)
	if err != nil {
		t.Fatal(err)
	}
	if len(pods.Items) != 1 {
		t.Errorf("expected 1 pod but found %d %v", len(pods.Items), k8s.GetPodNames(pods))
	}

	sts, err := unit_rcc.Client.ListStatefulSets(unit_rcc.Namespace)
	if err != nil {
		t.Fatal(err)
	}
	if len(sts.Items) != 1 {
		t.Errorf("expected 1 sts but found %d %v", len(sts.Items), k8s.GetStsNames(sts))
	}

	const PodName = "mycluster-0"
	checkPodError := func(args ...interface{}) (bool, error) {
		pod, err := unit_rcc.Client.GetPod(unit_rcc.Namespace, PodName)
		if err != nil {
			return false, err
		}
		return k8s.IsPodInitImagePullIssueError(pod)
	}
	updateFailed, err := unit_rcc.Wait(checkPodError, 60, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !updateFailed {
		t.Fatalf("after update the expected status of pod '%s' is '%s'",
			PodName, k8s.GetPodStateDescription(k8s.PodInitImagePullIssueError))
	}

	// fixing the imageRepository should let the cluster resume creation
	const PatchImageRepositoryTemplate = "patch-image-repository.yaml"
	generateData := struct {
		ImageRepository string
	}{
		ImageRepository: unit_rcc.Cfg.GetImageRegistryRepository(),
	}
	yamlPatchPath, err := unit_rcc.Generate(PatchImageRepositoryTemplate, generateData)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_rcc.Client.MergePatchInnoDBClusterFromFile(unit_rcc.Namespace, "mycluster", yamlPatchPath)
	if err != nil {
		t.Error(err)
	}

	// NOTE: seems we have to delete the pod to force it to be recreated
	// correctly
	err = unit_rcc.Client.DeletePod(unit_rcc.Namespace, "mycluster-0")
	if err != nil {
		t.Error(err)
	}

	// check cluster ok now
	err = unit_rcc.WaitOnPod("mycluster-0", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: -1,
	}

	err = unit_rcc.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	// cleanup
	err = unit_rcc.Client.DeleteInnoDBCluster(unit_rcc.Namespace, "mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcc.WaitOnPodGone("mycluster-0")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcc.WaitOnInnoDBClusterGone("mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcc.DeleteAllPersistentVolumeClaims()
	if err != nil {
		t.Error(err)
	}
}

func TeardownClusterSpecRuntimeChecksCreation(t *testing.T) {
	err := unit_rcc.Client.DeleteSecret(unit_rcc.Namespace, "mypwds")
	if err != nil {
		t.Error(err)
	}
}

func TestClusterSpecRuntimeChecksCreation(t *testing.T) {
	const Namespace = "badspec-creation"
	var err error
	unit_rcc, err = suit.NewUnitSetup(Namespace)
	if err != nil {
		t.Fatal(err)
	}

	SetupClusterSpecRuntimeChecksCreation(t)
	t.Run("BadSecretDelete", BadSecretDelete)
	t.Run("BadSecretRecover", BadSecretRecover)
	t.Run("UnsupportedVersionDelete", UnsupportedVersionDelete)
	t.Run("UnsupportedVersionRecover", UnsupportedVersionRecover)
	t.Run("BadPodDelete", BadPodDelete)
	t.Run("BadPodRecover", BadPodRecover)
	TeardownClusterSpecRuntimeChecksCreation(t)

	err = unit_rcc.Teardown()
	if err != nil {
		t.Error(err)
	}
}
