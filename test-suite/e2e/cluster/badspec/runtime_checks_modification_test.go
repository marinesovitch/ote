// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package badspec_test

// Same as ClusterSpecRuntimeChecksCreation but for clusters that already
// exist and have invalid spec changes made.

import (
	"fmt"
	"testing"

	"github.com/marinesovitch/ote/test-suite/util/common"
	"github.com/marinesovitch/ote/test-suite/util/k8s"
	"github.com/marinesovitch/ote/test-suite/util/suite"
	corev1 "k8s.io/api/core/v1"
)

var unit_rcm *suite.Unit

func SetupBadUpgrade(t *testing.T) {
	err := unit_rcm.Client.CreateUserSecrets(
		unit_rcm.Namespace, "mypwds", common.RootUser, common.DefaultHost, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_rcm.Apply("cluster-to-modify.yaml")
	if err != nil {
		t.Fatal(err)
	}

	err = unit_rcm.WaitOnPod("mycluster-2", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 3,
	}
	err = unit_rcm.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}
}

func BadUpgrade(t *testing.T) {
	// Check invalid spec change that would cause a rolling restart by setting
	// an invalid version.
	sinceResourceVersion, err := unit_rcm.GetInnoDBClusterResourceVersion("mycluster")
	if err != nil {
		t.Fatal(err)
	}

	patch := k8s.JsonPatch{
		Operation: k8s.PatchReplace,
		Path:      "/spec/version",
		Value:     "8.8.8",
	}

	err = unit_rcm.Client.JSONPatchInnoDBCluster(unit_rcm.Namespace, "mycluster", patch)
	if err != nil {
		t.Fatal(err)
	}

	// there should be events for the cluster resource indicating the update problem
	err = unit_rcm.AssertGotClusterEvent(t,
		"mycluster", sinceResourceVersion, "Normal", "Logging",
		fmt.Sprintf("Propagating spec.version=8.8.8 for %s/mycluster \\(was None\\)", unit_rcm.Namespace))
	if err != nil {
		t.Fatal(err)
	}

	err = unit_rcm.AssertGotClusterEvent(t,
		"mycluster", sinceResourceVersion, "Error", "Logging",
		"Handler 'on_innodbcluster_field_version/spec.version' failed permanently: version 8.8.8 must be between .*")
	if err != nil {
		t.Fatal(err)
	}

	err = unit_rcm.AssertGotClusterEvent(t,
		"mycluster", sinceResourceVersion, "Normal", "Logging",
		"Updating is processed: 0 succeeded; 1 failed.")
	if err != nil {
		t.Fatal(err)
	}

	// ensure cluster is still healthy
	if err = unit_rcm.WaitOnPod("mycluster-0", corev1.PodRunning); err != nil {
		t.Fatal(err)
	}
	if err = unit_rcm.WaitOnPod("mycluster-1", corev1.PodRunning); err != nil {
		t.Fatal(err)
	}
	if err = unit_rcm.WaitOnPod("mycluster-2", corev1.PodRunning); err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 3,
	}
	if err = unit_rcm.WaitOnInnoDBCluster(waitParams); err != nil {
		t.Fatal(err)
	}
}

func TeardownBadUpgrade(t *testing.T) {
	err := unit_rcm.Client.DeleteInnoDBCluster(unit_rcm.Namespace, "mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcm.WaitOnPodGone("mycluster-2")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcm.WaitOnPodGone("mycluster-1")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcm.WaitOnPodGone("mycluster-0")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcm.WaitOnInnoDBClusterGone("mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_rcm.Client.DeleteSecret(unit_rcm.Namespace, "mypwds")
	if err != nil {
		t.Error(err)
	}
}

func TestClusterSpecRuntimeChecksModification(t *testing.T) {
	const Namespace = "badspec-modification"
	var err error
	unit_rcm, err = suit.NewUnitSetup(Namespace)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("SetupBadUpgrade", SetupBadUpgrade)
	t.Run("BadUpgrade", BadUpgrade)
	t.Run("TeardownBadUpgrade", TeardownBadUpgrade)

	err = unit_rcm.Teardown()
	if err != nil {
		t.Error(err)
	}
}
