// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package config_test

import (
	"testing"

	"github.com/marinesovitch/ote/test-suite/util/common"
	"github.com/marinesovitch/ote/test-suite/util/k8s"
	"github.com/marinesovitch/ote/test-suite/util/suite"
)

var unit_cr *suite.Unit

// Create and delete a cluster immediately, before it becomes ONLINE.
func CreateAndDelete(t *testing.T) {
	err := unit_cr.Client.CreateUserSecrets(unit_cr.Namespace, "mypwds", common.RootUser, common.DefaultHost, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}

	// create cluster with mostly default configs
	err = unit_cr.Apply("cluster-races.yaml")
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"PENDING"},
		ExpectedNumOnline: -1,
	}
	err = unit_cr.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	// deleting a cluster right after it's created and before it's ONLINE
	// caused a loop and didn't finish deleting before
	err = unit_cr.Client.DeleteInnoDBCluster(unit_cr.Namespace, "mycluster")
	if err != nil {
		t.Fatal(err)
	}

	err = unit_cr.WaitOnInnoDBClusterGone("mycluster")
	if err != nil {
		t.Fatal(err)
	}

	err = unit_cr.WaitOnPodGone("mycluster-0")
	if err != nil {
		t.Fatal(err)
	}
}

func TestClusterRaces(t *testing.T) {
	const Namespace = "cluster-races"
	var err error
	unit_cr, err = suit.NewUnitSetup(Namespace)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("CreateAndDelete=0", CreateAndDelete)

	err = unit_cr.Teardown()
	if err != nil {
		t.Error(err)
	}
}
