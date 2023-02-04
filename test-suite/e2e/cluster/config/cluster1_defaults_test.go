// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package config_test

// test single instance cluster with all default configs

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/marinesovitch/ote/test-suite/util/auxi"
	"github.com/marinesovitch/ote/test-suite/util/common"
	"github.com/marinesovitch/ote/test-suite/util/k8s"
	"github.com/marinesovitch/ote/test-suite/util/mysql"
	"github.com/marinesovitch/ote/test-suite/util/suite"

	corev1 "k8s.io/api/core/v1"
)

var unit_c1d *suite.Unit

func CreateClusterOneInstance(t *testing.T) {
	// Create cluster, check posted events.
	err := unit_c1d.Client.CreateUserSecrets(unit_c1d.Namespace, "mypwds", common.RootUser, common.DefaultHost, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}

	// create cluster with mostly default configs
	err = unit_c1d.Apply("cluster1-defaults.yaml")
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"PENDING", "INITIALIZING", "ONLINE"},
		ExpectedNumOnline: -1,
	}
	err = unit_c1d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_c1d.WaitOnPod("mycluster-0", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 1,
	}
	err = unit_c1d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	params := unit_c1d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 1
	params.Routers = 0
	params.Primary = 0
	if _, err := suite.CheckAll(unit_c1d, params); err != nil {
		t.Fatal(err)
	}

	if err := unit_c1d.AssertGotClusterEvent(t,
		"mycluster", common.AnyResourceVersion, "Normal",
		"ResourcesCreated",
		"Dependency resources created, switching status to PENDING"); err != nil {
		t.Fatal(err)
	}
	if err := unit_c1d.AssertGotClusterEvent(t,
		"mycluster", common.AnyResourceVersion, "Normal",
		"StatusChange", "Cluster status changed to INITIALIZING. 0 member\\(s\\) ONLINE"); err != nil {
		t.Fatal(err)
	}
	if err := unit_c1d.AssertGotClusterEvent(t,
		"mycluster", common.AnyResourceVersion, "Normal",
		"StatusChange", "Cluster status changed to ONLINE. 1 member\\(s\\) ONLINE"); err != nil {
		t.Fatal(err)
	}
}

func CheckAccounts1(t *testing.T) {
	accounts, err := suite.QuerySet(
		unit_c1d.Namespace, "mycluster-0", "root", "sakila",
		"SELECT concat(user,'@',host) FROM mysql.user", 0)
	if err != nil {
		t.Fatal(err)
	}

	expectedAccounts := []string{"root@%",
		"localroot@localhost", "mysqladmin@%", "mysqlbackup@%", "mysqlrouter@%",
		"mysqlhealthchecker@localhost", "mysql_innodb_cluster_1000@%"}
	expectedAccountSet := suite.PrepareAccountSet(expectedAccounts, true)

	if !auxi.AreStringSetsEqual(accounts, expectedAccountSet) {
		t.Fatalf("expected accounts are %v but got %v", expectedAccountSet.ToSortedSlice(), accounts.ToSortedSlice())
	}
}

func BadChanges(t *testing.T) {
	t.Skip("it was marked as TODO - not completed yet")
	// this should trigger an error and no changes
	// changes after this should continue working normally
	patch := k8s.JsonPatch{
		Operation: k8s.PatchReplace,
		Path:      "/spec/instances",
		Value:     22,
	}
	err := unit_c1d.Client.JSONPatchInnoDBCluster(unit_c1d.Namespace, "mycluster", patch)
	if err != nil {
		t.Error(err)
	}

	// check that the error appears in describe ic output

	// check that nothing changed
	params := unit_c1d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 1
	params.Routers = 0
	params.Primary = 0
	if _, err := suite.CheckAll(unit_c1d, params); err != nil {
		t.Fatal(err)
	}
}

func GrowTwoInstances(t *testing.T) {
	patch := k8s.JsonPatch{
		Operation: k8s.PatchReplace,
		Path:      "/spec/instances",
		Value:     2,
	}
	err := unit_c1d.Client.JSONPatchInnoDBCluster(unit_c1d.Namespace, "mycluster", patch)
	if err != nil {
		t.Error(err)
	}

	err = unit_c1d.WaitOnPod("mycluster-1", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 2,
	}
	err = unit_c1d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	params := unit_c1d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 2
	params.Primary = 0
	params.Routers = 0
	if _, err := suite.CheckAll(unit_c1d, params); err != nil {
		t.Fatal(err)
	}
}

func AddRouters(t *testing.T) {
	patch := k8s.JsonPatch{
		Operation: k8s.PatchReplace,
		Path:      "/spec/router/instances",
		Value:     3,
	}
	err := unit_c1d.Client.JSONPatchInnoDBCluster(unit_c1d.Namespace, "mycluster", patch)
	if err != nil {
		t.Error(err)
	}

	routersReady := func(args ...interface{}) (bool, error) {
		pods, err := unit_c1d.Client.ListPods(unit_c1d.Namespace)
		if err != nil {
			t.Fatal(err)
		}

		const expectedRoutersNum = 3
		routersCounter := 0
		for _, pod := range pods.Items {
			podName := pod.GetName()
			if strings.HasPrefix(podName, "mycluster-router-") && pod.Status.Phase == corev1.PodRunning {
				routersCounter++
			}
		}

		if routersCounter != expectedRoutersNum {
			return false, nil
		}

		return true, nil
	}

	if _, err := unit_c1d.Wait(routersReady, 240, 3); err != nil {
		t.Fatal(err)
	}

	params := unit_c1d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 2
	params.Routers = 3
	params.Primary = 0
	if _, err := suite.CheckAll(unit_c1d, params); err != nil {
		t.Fatal(err)
	}
}

func GrowThreeInstances(t *testing.T) {
	patch := k8s.JsonPatch{
		Operation: k8s.PatchReplace,
		Path:      "/spec/instances",
		Value:     3,
	}
	err := unit_c1d.Client.JSONPatchInnoDBCluster(unit_c1d.Namespace, "mycluster", patch)
	if err != nil {
		t.Error(err)
	}

	err = unit_c1d.WaitOnPod("mycluster-2", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 3,
	}
	err = unit_c1d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_c1d.WaitOnRouters("mycluster", 3); err != nil {
		t.Fatal(err)
	}

	params := unit_c1d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 3
	params.Primary = 0
	params.Routers = 3
	if _, err := suite.CheckAll(unit_c1d, params); err != nil {
		t.Fatal(err)
	}
}

func ShrinkToOneInstance(t *testing.T) {
	patch := k8s.JsonPatch{
		Operation: k8s.PatchReplace,
		Path:      "/spec/instances",
		Value:     1,
	}
	err := unit_c1d.Client.JSONPatchInnoDBCluster(unit_c1d.Namespace, "mycluster", patch)
	if err != nil {
		t.Error(err)
	}

	err = unit_c1d.WaitOnPodGone("mycluster-2")
	if err != nil {
		t.Error(err)
	}

	err = unit_c1d.WaitOnPodGone("mycluster-1")
	if err != nil {
		t.Error(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE_PARTIAL"},
		ExpectedNumOnline: 1,
	}
	err = unit_c1d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_c1d.WaitOnRouters("mycluster", 3); err != nil {
		t.Fatal(err)
	}

	params := unit_c1d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 1
	params.Primary = 0
	params.Routers = 3
	if _, err := suite.CheckAll(unit_c1d, params); err != nil {
		t.Fatal(err)
	}
}

func RecoverCrash(t *testing.T) {
	// Force a mysqld process crash.
	// The only thing expected to happen is that mysql restarts and the
	// cluster is resumed.

	pod, err := unit_c1d.Client.GetPod(unit_c1d.Namespace, "mycluster-0")
	if err != nil {
		t.Fatal(err)
	}

	mysqlCont, err := k8s.GetContainerStatus(pod, k8s.Mysql)
	if err != nil {
		t.Fatal(err)
	}

	sidecarCont, err := k8s.GetContainerStatus(pod, k8s.Sidecar)
	if err != nil {
		t.Fatal(err)
	}

	sinceResourceVersion, err := unit_c1d.GetInnoDBClusterResourceVersion("mycluster")
	if err != nil {
		t.Fatal(err)
	}

	// kill mysqld (pid 1)
	if err := unit_c1d.Client.Kill(unit_c1d.Namespace, "mycluster-0", k8s.Mysql, 11, 1); err != nil {
		t.Fatal(err)
	}

	// wait for operator to notice it gone
	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:                 "mycluster",
		ExpectedStatus:       []string{"OFFLINE", "OFFLINE_UNCERTAIN"},
		ExpectedNumOnline:    0,
		SinceResourceVersion: sinceResourceVersion,
	}
	err = unit_c1d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	sinceResourceVersion, err = unit_c1d.GetInnoDBClusterResourceVersion("mycluster")
	if err != nil {
		t.Fatal(err)
	}

	// wait for operator to restore it
	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:                 "mycluster",
		ExpectedStatus:       []string{"ONLINE"},
		ExpectedNumOnline:    1,
		Timeout:              600,
		SinceResourceVersion: sinceResourceVersion,
	}
	err = unit_c1d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_c1d.WaitOnRouters("mycluster", 3); err != nil {
		t.Fatal(err)
	}

	if err := unit_c1d.AssertGotClusterEvent(t,
		"mycluster", sinceResourceVersion, "Normal",
		"Rebooting", "Restoring OFFLINE cluster"); err != nil {
		t.Fatal(err)
	}

	// ensure persisted config didn't change after recovery
	jsonConfig, err := unit_c1d.Client.Cat(unit_c1d.Namespace, "mycluster-0", k8s.Mysql, "/var/lib/mysql/mysqld-auto.cnf")
	if err != nil {
		t.Fatal(jsonConfig)
	}
	var config map[string]interface{}
	err = json.Unmarshal([]byte(jsonConfig), &config)
	if err != nil {
		t.Fatal(err)
	}
	groupReplicationStartOnBoot, err := suite.GetStringFromJSONTree(config, "mysql_static_variables", "group_replication_start_on_boot", "Value")
	if err != nil {
		t.Fatal(err)
	}
	expectedGroupReplicationStartOnBoot := "OFF"
	if groupReplicationStartOnBoot != expectedGroupReplicationStartOnBoot {
		t.Fatalf("expected group-replication-start-on-boot is %s but got %s", expectedGroupReplicationStartOnBoot, groupReplicationStartOnBoot)
	}

	pod, err = unit_c1d.Client.GetPod(unit_c1d.Namespace, "mycluster-0")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := suite.CheckPodContainer(unit_c1d.Client, pod, k8s.Mysql, mysqlCont.RestartCount+1, true); err != nil {
		t.Fatal(err)
	}
	if _, err := suite.CheckPodContainer(unit_c1d.Client, pod, k8s.Sidecar, sidecarCont.RestartCount, true); err != nil {
		t.Fatal(err)
	}

	params := unit_c1d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 1
	params.Primary = 0
	params.Routers = 3
	params.RestartsExpected = true
	if _, err := suite.CheckAll(unit_c1d, params); err != nil {
		t.Fatal(err)
	}
}

func RecoverSidecarCrash(t *testing.T) {
	// Force a sidecar process crash.
	// Nothing is expected to happen other than sidecar restarting and
	// going back to ready state, since the sidecar is idle.

	t.Skip("killing the sidecar isn't working somehow")

	pod, err := unit_c1d.Client.GetPod(unit_c1d.Namespace, "mycluster-0")
	if err != nil {
		t.Fatal(err)
	}
	mysqlCont, err := k8s.GetContainerStatus(pod, k8s.Mysql)
	if err != nil {
		t.Fatal(err)
	}
	sidecarCont, err := k8s.GetContainerStatus(pod, k8s.Sidecar)
	if err != nil {
		t.Fatal(err)
	}

	jsonConfig, err := unit_c1d.Client.Cat(unit_c1d.Namespace, "mycluster-0", k8s.Mysql, "/var/lib/mysql/mysqld-auto.cnf")
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]interface{}
	err = json.Unmarshal([]byte(jsonConfig), &config)
	if err != nil {
		t.Fatal(err)
	}
	groupReplicationStartOnBoot, err := suite.GetStringFromJSONTree(config, "mysql_static_variables", "group_replication_start_on_boot", "Value")
	if err != nil {
		t.Fatal(err)
	}
	expectedGroupReplicationStartOnBoot := "OFF"
	if groupReplicationStartOnBoot != expectedGroupReplicationStartOnBoot {
		t.Fatalf("expected group-replication-start-on-boot is %s but got %s", expectedGroupReplicationStartOnBoot, groupReplicationStartOnBoot)
	}

	// kill sidecar (pid 1)
	err = unit_c1d.Client.Kill(unit_c1d.Namespace, "mycluster-0", k8s.Sidecar, 11, 1)
	if err != nil {
		t.Fatal(err)
	}

	checkIsReady := func(args ...interface{}) (bool, error) {
		t := args[0].(testing.T)
		pod, err := unit_c1d.Client.GetPod(unit_c1d.Namespace, "mycluster-0")
		if err != nil {
			t.Fatal(err)
		}
		podSidecarCont, err := k8s.GetContainerStatus(pod, k8s.Sidecar)
		if err != nil {
			t.Fatal(err)
		}
		currentRestartCount := podSidecarCont.RestartCount
		previousRestartCount := sidecarCont.RestartCount + 1
		if podSidecarCont.RestartCount != previousRestartCount {
			return false, fmt.Errorf("RestartCount current %d, previous %d, expected equal", currentRestartCount, previousRestartCount)
		}
		return true, nil
	}

	if _, err := unit_c1d.Wait(checkIsReady, 60, 2, t); err != nil {
		t.Fatal(err)
	}

	pod, err = unit_c1d.Client.GetPod(unit_c1d.Namespace, "mycluster-0")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := suite.CheckPodContainer(unit_c1d.Client, pod, k8s.Mysql, mysqlCont.RestartCount, true); err != nil {
		t.Fatal(err)
	}
	if _, err := suite.CheckPodContainer(unit_c1d.Client, pod, k8s.Sidecar, sidecarCont.RestartCount+1, true); err != nil {
		t.Fatal(err)
	}

	// ensure persisted config didn't change after recovery (regression test)
	jsonConfig, err = unit_c1d.Client.Cat(unit_c1d.Namespace, "mycluster-0", k8s.Mysql, "/var/lib/mysql/mysqld-auto.cnf")
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal([]byte(jsonConfig), &config)
	if err != nil {
		t.Fatal(err)
	}
	groupReplicationStartOnBoot, err = suite.GetStringFromJSONTree(config, "mysql_static_variables", "group_replication_start_on_boot", "Value")
	if err != nil {
		t.Fatal(err)
	}
	expectedGroupReplicationStartOnBoot = "OFF"
	if groupReplicationStartOnBoot != expectedGroupReplicationStartOnBoot {
		t.Fatalf("expected group-replication-start-on-boot is %s but got %s", expectedGroupReplicationStartOnBoot, groupReplicationStartOnBoot)
	}

	// check that all containers are OK
	params := unit_c1d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 1
	params.Primary = 0
	params.Routers = 3
	params.RestartsExpected = true
	if _, err := suite.CheckAll(unit_c1d, params); err != nil {
		t.Fatal(err)
	}
}

func RecoverRestart(t *testing.T) {
	pod, err := unit_c1d.Client.GetPod(unit_c1d.Namespace, "mycluster-0")
	if err != nil {
		t.Fatal(err)
	}
	mysqlCont, err := k8s.GetContainerStatus(pod, k8s.Mysql)
	if err != nil {
		t.Fatal(err)
	}
	sidecarCont, err := k8s.GetContainerStatus(pod, k8s.Sidecar)
	if err != nil {
		t.Fatal(err)
	}

	sinceResourceVersion, err := unit_c1d.GetInnoDBClusterResourceVersion("mycluster")
	if err != nil {
		t.Fatal(err)
	}

	podSession, err := mysql.NewSession(unit_c1d.Namespace, "mycluster-0", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := podSession.Exec("restart"); err != nil {
		t.Fatal(err)
	}

	// wait for operator to notice it gone
	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"OFFLINE", "OFFLINE_UNCERTAIN"},
		ExpectedNumOnline: 0,
	}
	err = unit_c1d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	// wait/ensure pod restarted
	pod, err = unit_c1d.Client.GetPod(unit_c1d.Namespace, "mycluster-0")
	if err != nil {
		t.Fatal(err)
	}

	currentMysqlCont, err := k8s.GetContainerStatus(pod, k8s.Mysql)
	if err != nil {
		t.Fatal(err)
	}
	currentMysqlRestartCount := currentMysqlCont.RestartCount
	expectedMysqlRestartCount := mysqlCont.RestartCount + 1
	if currentMysqlRestartCount != expectedMysqlRestartCount {
		t.Fatalf("mysql container restart count is %d but expected %d", currentMysqlRestartCount, expectedMysqlRestartCount)
	}

	// ensure sidecar didn't restart
	currentSidecarCont, err := k8s.GetContainerStatus(pod, k8s.Sidecar)
	if err != nil {
		t.Fatal(err)
	}
	currentSidecarRestartCount := currentSidecarCont.RestartCount
	expectedSidecarRestartCount := sidecarCont.RestartCount
	if currentSidecarRestartCount != expectedSidecarRestartCount {
		t.Fatalf("sidecar container restart count is %d but expected %d", currentSidecarRestartCount, expectedSidecarRestartCount)
	}

	// wait for operator to restore it
	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 1,
	}
	err = unit_c1d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_c1d.WaitOnRouters("mycluster", 3); err != nil {
		t.Fatal(err)
	}

	if err := unit_c1d.AssertGotClusterEvent(t,
		"mycluster", sinceResourceVersion, "Normal",
		"Rebooting", "Restoring OFFLINE cluster"); err != nil {
		t.Fatal(err)
	}

	params := unit_c1d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 1
	params.Primary = 0
	params.Routers = 3
	params.RestartsExpected = true
	if _, err := suite.CheckAll(unit_c1d, params); err != nil {
		t.Fatal(err)
	}
}

func RecoverShutdown(t *testing.T) {
	pod, err := unit_c1d.Client.GetPod(unit_c1d.Namespace, "mycluster-0")
	if err != nil {
		t.Fatal(err)
	}
	mysqlCont, err := k8s.GetContainerStatus(pod, k8s.Mysql)
	if err != nil {
		t.Fatal(err)
	}
	sidecarCont, err := k8s.GetContainerStatus(pod, k8s.Sidecar)
	if err != nil {
		t.Fatal(err)
	}

	sinceResourceVersion, err := unit_c1d.GetInnoDBClusterResourceVersion("mycluster")
	if err != nil {
		t.Fatal(err)
	}

	podSession, err := mysql.NewSession(unit_c1d.Namespace, "mycluster-0", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := podSession.Exec("shutdown"); err != nil {
		t.Fatal(err)
	}

	// wait for operator to notice it gone
	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"OFFLINE", "OFFLINE_UNCERTAIN"},
		ExpectedNumOnline: 0,
	}
	err = unit_c1d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	// wait/ensure pod restarted
	pod, err = unit_c1d.Client.GetPod(unit_c1d.Namespace, "mycluster-0")
	if err != nil {
		t.Fatal(err)
	}

	currentMysqlCont, err := k8s.GetContainerStatus(pod, k8s.Mysql)
	if err != nil {
		t.Fatal(err)
	}
	currentMysqlRestartCount := currentMysqlCont.RestartCount
	expectedMysqlRestartCount := mysqlCont.RestartCount + 1
	if currentMysqlRestartCount != expectedMysqlRestartCount {
		t.Fatalf("mysql container restart count is %d but expected %d", currentMysqlRestartCount, expectedMysqlRestartCount)
	}

	// ensure sidecar didn't restart
	currentSidecarCont, err := k8s.GetContainerStatus(pod, k8s.Sidecar)
	if err != nil {
		t.Fatal(err)
	}
	currentSidecarRestartCount := currentSidecarCont.RestartCount
	expectedSidecarRestartCount := sidecarCont.RestartCount
	if currentSidecarRestartCount != expectedSidecarRestartCount {
		t.Fatalf("sidecar container restart count is %d but expected %d", currentSidecarRestartCount, expectedSidecarRestartCount)
	}

	// wait for operator to restore it
	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 1,
	}
	err = unit_c1d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_c1d.WaitOnRouters("mycluster", 3); err != nil {
		t.Fatal(err)
	}

	if err := unit_c1d.AssertGotClusterEvent(t,
		"mycluster", sinceResourceVersion, "Normal",
		"Rebooting", "Restoring OFFLINE cluster"); err != nil {
		t.Fatal(err)
	}

	params := unit_c1d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 1
	params.Primary = 0
	params.Routers = 3
	params.RestartsExpected = true
	if _, err := suite.CheckAll(unit_c1d, params); err != nil {
		t.Fatal(err)
	}
}

func RecoverDelete(t *testing.T) {
	err := unit_c1d.Client.DeletePodWithTimeout(unit_c1d.Namespace, "mycluster-0", 200)
	if err != nil {
		t.Fatal(err)
	}

	sinceResourceVersion, err := unit_c1d.GetInnoDBClusterResourceVersion("mycluster")
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"OFFLINE"},
		ExpectedNumOnline: 0,
	}
	err = unit_c1d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	// wait for operator to restore everything
	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 1,
	}
	err = unit_c1d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	pod1, err := unit_c1d.Client.GetPod(unit_c1d.Namespace, "mycluster-0")
	if err != nil {
		t.Fatal(err)
	}

	// the pod was deleted, so restarts resets to 0
	if _, err := suite.CheckPodContainer(unit_c1d.Client, pod1, k8s.Mysql, 0, true); err != nil {
		t.Fatal(err)
	}
	if _, err := suite.CheckPodContainer(unit_c1d.Client, pod1, k8s.Sidecar, 0, true); err != nil {
		t.Fatal(err)
	}

	if err := unit_c1d.AssertGotClusterEvent(t,
		"mycluster", sinceResourceVersion, "Normal",
		"Rebooting", "Restoring OFFLINE cluster"); err != nil {
		t.Fatal(err)
	}

	params := unit_c1d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 1
	params.Primary = 0
	params.Routers = 3
	if _, err := suite.CheckAll(unit_c1d, params); err != nil {
		t.Fatal(err)
	}
}

func RecoverStop(t *testing.T) {
	t.Skip("todo")
	podSessions0, err := mysql.NewSession(unit_c1d.Namespace, "mycluster-0", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := podSessions0.Exec("stop group_replication"); err != nil {
		t.Fatal(err)
	}

	// wait for operator to notice it OFFLINE
	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"OFFLINE"},
		ExpectedNumOnline: 0,
	}
	err = unit_c1d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_c1d.WaitOnRouters("mycluster", 3); err != nil {
		t.Fatal(err)
	}

	// wait for operator to restore everything
	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 1,
	}
	err = unit_c1d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	params := unit_c1d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 1
	params.Primary = 0
	params.Routers = 3
	params.RestartsExpected = true
	if _, err := suite.CheckAll(unit_c1d, params); err != nil {
		t.Fatal(err)
	}
}

func AfterCluster1Defaults(t *testing.T) {
	err := unit_c1d.Client.DeleteInnoDBCluster(unit_c1d.Namespace, "mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_c1d.WaitOnPodGone("mycluster-0")
	if err != nil {
		t.Error(err)
	}

	err = unit_c1d.WaitOnInnoDBClusterGone("mycluster")
	if err != nil {
		t.Error(err)
	}
}

func TestCluster1Defaults(t *testing.T) {
	const Namespace = "cluster1-defaults"
	var err error
	unit_c1d, err = suit.NewUnitSetup(Namespace)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("CreateClusterOneInstance=0", CreateClusterOneInstance)
	t.Run("CheckAccounts1=1", CheckAccounts1)
	// t.Run("BadChanges=2", BadChanges)
	t.Run("GrowTwoInstances=2", GrowTwoInstances)
	t.Run("AddRouters=2", AddRouters)
	t.Run("GrowThreeInstances=2", GrowThreeInstances)
	t.Run("ShrinkToOneInstance=2", ShrinkToOneInstance)
	t.Run("RecoverCrash=3", RecoverCrash)
	//t.Run("RecoverSidecarCrash=3", RecoverSidecarCrash)
	t.Run("RecoverRestart=3", RecoverRestart)
	t.Run("RecoverShutdown=3", RecoverShutdown)
	t.Run("RecoverDelete=3", RecoverDelete)
	// t.Run("RecoverStop=3", RecoverStop)
	t.Run("AfterCluster1Defaults=9", AfterCluster1Defaults)

	err = unit_c1d.Teardown()
	if err != nil {
		t.Error(err)
	}
}
