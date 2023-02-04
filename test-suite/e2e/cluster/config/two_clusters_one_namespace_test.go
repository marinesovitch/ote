// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package config_test

import (
	"testing"

	"github.com/marinesovitch/ote/test-suite/util/common"
	"github.com/marinesovitch/ote/test-suite/util/k8s"
	"github.com/marinesovitch/ote/test-suite/util/suite"
	corev1 "k8s.io/api/core/v1"
)

var unit_tc *suite.Unit

const ClusterOneName = "mycluster"
const ClusterTwoName = "mycluster2"
const ClusterTwoPassword = "sakilax"

func CreateClusterOne(t *testing.T) {
	err := unit_tc.Client.CreateUserSecrets(unit_tc.Namespace, "mypwds", common.RootUser, common.DefaultHost, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}

	// create cluster with mostly default configs
	err = unit_tc.Apply("cluster-one.yaml")
	if err != nil {
		t.Fatal(err)
	}

	// ensure router pods don't get created until the cluster is ONLINE
	if err := suite.CheckRouterPods(unit_tc.Client, unit_tc.Namespace, ClusterOneName, 0); err != nil {
		t.Fatal(err)
	}

	err = unit_tc.WaitOnPod(ClusterOneName+"-0", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              ClusterOneName,
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 1,
	}
	err = unit_tc.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_tc.WaitOnRouters(ClusterOneName, 1); err != nil {
		t.Fatal(err)
	}

	params := unit_tc.GetDefaultCheckParams()
	params.Name = ClusterOneName
	params.Instances = 1
	params.Routers = 1
	params.Primary = 0
	if _, err := suite.CheckAll(unit_tc, params); err != nil {
		t.Fatal(err)
	}
}

func CreateClusterTwo(t *testing.T) {
	err := unit_tc.Client.CreateUserSecrets(
		unit_tc.Namespace, "mypwds2", common.RootUser, common.DefaultHost, ClusterTwoPassword)
	if err != nil {
		t.Fatal(err)
	}

	// create cluster with mostly default configs
	err = unit_tc.Apply("cluster-two.yaml")
	if err != nil {
		t.Fatal(err)
	}

	// ensure router pods don't get created until the cluster is ONLINE
	if err := suite.CheckRouterPods(unit_tc.Client, unit_tc.Namespace, ClusterTwoName, 0); err != nil {
		t.Fatal(err)
	}

	err = unit_tc.WaitOnPod(ClusterTwoName+"-0", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              ClusterTwoName,
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 1,
	}
	err = unit_tc.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_tc.WaitOnRouters(ClusterTwoName, 2); err != nil {
		t.Fatal(err)
	}

	params := unit_tc.GetDefaultCheckParams()
	params.Name = ClusterTwoName
	params.Instances = 1
	params.Routers = 2
	params.Primary = 0
	params.Password = ClusterTwoPassword
	params.SharedNamespace = true
	if _, err := suite.CheckAll(unit_tc, params); err != nil {
		t.Fatal(err)
	}
}

func DestroyClusterOne(t *testing.T) {
	err := unit_tc.Client.DeleteInnoDBCluster(unit_tc.Namespace, ClusterOneName)
	if err != nil {
		t.Error(err)
	}

	err = unit_tc.WaitOnPodGone(ClusterOneName + "-0")
	if err != nil {
		t.Error(err)
	}

	err = unit_tc.WaitOnInnoDBClusterGone(ClusterOneName)
	if err != nil {
		t.Error(err)
	}

	// mycluster2 should still be fine
	params := unit_tc.GetDefaultCheckParams()
	params.Name = ClusterTwoName
	params.Instances = 1
	params.Routers = 2
	params.Primary = 0
	params.Password = ClusterTwoPassword
	if _, err := suite.CheckAll(unit_tc, params); err != nil {
		t.Fatal(err)
	}
}

func DestroyClusterTwo(t *testing.T) {
	err := unit_tc.Client.DeleteInnoDBCluster(unit_tc.Namespace, ClusterTwoName)
	if err != nil {
		t.Error(err)
	}

	err = unit_tc.WaitOnPodGone(ClusterTwoName + "-0")
	if err != nil {
		t.Error(err)
	}

	err = unit_tc.WaitOnInnoDBClusterGone(ClusterTwoName)
	if err != nil {
		t.Error(err)
	}
}

func TestTwoClustersOneNamespace(t *testing.T) {
	const Namespace = "two-clusters-one-ns"
	var err error
	unit_tc, err = suit.NewUnitSetup(Namespace)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("CreateClusterOne=0", CreateClusterOne)
	t.Run("CreateClusterTwo=0", CreateClusterTwo)
	t.Run("DestroyClusterOne=1", DestroyClusterOne)
	t.Run("DestroyClusterTwo=1", DestroyClusterTwo)

	err = unit_tc.Teardown()
	if err != nil {
		t.Error(err)
	}
}
