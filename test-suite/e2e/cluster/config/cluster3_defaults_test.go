// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package config_test

// test three instances cluster with default configs

import (
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

var unit_c3d *suite.Unit

func CreateClusterThreeInstances(t *testing.T) {
	err := unit_c3d.Client.CreateUserSecrets(
		unit_c3d.Namespace, "mypwds", common.RootUser, common.DefaultHost, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}

	// create cluster with mostly default configs
	err = unit_c3d.Apply("cluster3-defaults.yaml")
	if err != nil {
		t.Fatal(err)
	}

	// ensure router pods don't get created until the cluster is ONLINE
	if err := suite.CheckRouterPods(unit_c3d.Client, unit_c3d.Namespace, "mycluster", 0); err != nil {
		t.Fatal(err)
	}

	err = unit_c3d.WaitOnPod("mycluster-0", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 1,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err := suite.CheckRouterPods(unit_c3d.Client, unit_c3d.Namespace, "mycluster", 0); err != nil {
		t.Fatal(err)
	}

	err = unit_c3d.WaitOnPod("mycluster-1", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_c3d.WaitOnPod("mycluster-2", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 3,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_c3d.WaitOnRouters("mycluster", 2); err != nil {
		t.Fatal(err)
	}

	if err := suite.CheckRouterPods(unit_c3d.Client, unit_c3d.Namespace, "mycluster", 2); err != nil {
		t.Fatal(err)
	}

	params := unit_c3d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 3
	params.Routers = 2
	params.Primary = 0
	if _, err := suite.CheckAll(unit_c3d, params); err != nil {
		t.Fatal(err)
	}
}

func CheckVersion3(t *testing.T) {
	// ensure containers have the right version and edition
	pod, err := unit_c3d.Client.GetPod(unit_c3d.Namespace, "mycluster-0")
	if err != nil {
		t.Fatal(err)
	}

	initmysqlCont, err := k8s.GetContainer(pod, k8s.InitMysql)
	if err != nil {
		t.Fatal(err)
	}

	image := initmysqlCont.Image
	expectedVersionTag := ":" + unit_c3d.Cfg.Images.DefaultVersionTag
	if !strings.Contains(image, expectedVersionTag) {
		t.Fatalf("initmysql container image should contain %s but it doesn't %s", expectedVersionTag, image)
	}

	expectedImageName := unit_c3d.Cfg.Images.MysqlServerImage + ":"
	if !strings.Contains(image, expectedImageName) {
		t.Fatalf("initmysql container image should contain %s but it doesn't %s", expectedImageName, image)
	}

	initconfCont, err := k8s.GetContainer(pod, k8s.InitConf)
	if err != nil {
		t.Fatal(err)
	}
	image = initconfCont.Image
	expectedVersionTag = ":" + unit_c3d.Cfg.Operator.VersionTag
	if !strings.Contains(image, expectedVersionTag) {
		t.Fatalf("initconf container image should contain %s but it doesn't %s", expectedVersionTag, image)
	}

	expectedImageName = unit_c3d.Cfg.Operator.Image + ":"
	if !strings.Contains(image, expectedImageName) {
		t.Fatalf("initconf container image should contain %s but it doesn't %s", expectedImageName, image)
	}

	mysqlCont, err := k8s.GetContainer(pod, k8s.Mysql)
	if err != nil {
		t.Fatal(err)
	}
	image = mysqlCont.Image
	expectedVersionTag = ":" + unit_c3d.Cfg.Images.DefaultVersionTag
	if !strings.Contains(image, expectedVersionTag) {
		t.Fatalf("mysql container image should contain %s but it doesn't %s", expectedVersionTag, image)
	}

	expectedImageName = unit_c3d.Cfg.Images.MysqlServerImage + ":"
	if !strings.Contains(image, expectedImageName) {
		t.Fatalf("mysql container image should contain %s but it doesn't %s", expectedImageName, image)
	}

	sidecarCont, err := k8s.GetContainer(pod, k8s.Sidecar)
	if err != nil {
		t.Fatal(err)
	}
	image = sidecarCont.Image
	expectedVersionTag = ":" + unit_c3d.Cfg.Operator.VersionTag
	if !strings.Contains(image, expectedVersionTag) {
		t.Fatalf("sidecar container image should contain %s but it doesn't %s", expectedVersionTag, image)
	}

	expectedImageName = unit_c3d.Cfg.Operator.Image + ":"
	if !strings.Contains(image, expectedImageName) {
		t.Fatalf("sidecar container image should contain %s but it doesn't %s", expectedImageName, image)
	}

	// check router version and edition
	routers, err := unit_c3d.Client.ListPodsWithFilter(unit_c3d.Namespace, "mycluster-router-.*")
	if err != nil {
		t.Fatal(err)
	}
	if len(routers.Items) == 0 {
		t.Fatal("no routers found")
	}
	routerPod := &routers.Items[0]
	routerCont, err := k8s.GetContainer(routerPod, k8s.Router)
	if err != nil {
		t.Fatal(err)
	}
	image = routerCont.Image
	expectedVersionTag = ":" + unit_c3d.Cfg.Images.DefaultVersionTag
	if !strings.Contains(image, expectedVersionTag) {
		t.Fatalf("router container image should contain %s but it doesn't %s", expectedVersionTag, image)
	}

	expectedImageName = unit_c3d.Cfg.Images.MysqlRouterImage + ":"
	if !strings.Contains(image, expectedImageName) {
		t.Fatalf("router container image should contain %s but it doesn't %s", expectedImageName, image)
	}
}

func checkClusterAccounts3(t *testing.T, clusterName string) {
	accounts, err := suite.QuerySet(
		unit_c3d.Namespace, clusterName, "root", "sakila",
		"SELECT concat(user,'@',host) FROM mysql.user", 0)
	if err != nil {
		t.Fatal(err)
	}

	expectedAccounts := []string{"root@%",
		"localroot@localhost", "mysqladmin@%", "mysqlbackup@%", "mysqlrouter@%",
		"mysqlhealthchecker@localhost", "mysql_innodb_cluster_1000@%",
		"mysql_innodb_cluster_1001@%", "mysql_innodb_cluster_1002@%"}
	expectedAccountSet := suite.PrepareAccountSet(expectedAccounts, true)

	if !auxi.AreStringSetsEqual(accounts, expectedAccountSet) {
		t.Fatalf("expected accounts are %v but got %v", expectedAccountSet.ToSortedSlice(), accounts.ToSortedSlice())
	}
}

func CheckAccounts3(t *testing.T) {
	checkClusterAccounts3(t, "mycluster-0")
	checkClusterAccounts3(t, "mycluster-1")
	checkClusterAccounts3(t, "mycluster-2")
}

type GenerateRoutingData struct {
	Image string
}

func CheckRouting(t *testing.T) {
	// Check routing from a standalone pod in a different namespace
	// create a pod to connect from (as an app)
	const routingYaml = "cluster3-routing.yaml"
	generateData := GenerateRoutingData{
		Image: unit_c3d.GetDefaultOperatorImage(),
	}

	appNamespace := unit_c3d.AuxNamespace
	err := unit_c3d.Client.CreateNamespace(appNamespace)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_c3d.GenerateAndApplyInNamespace(appNamespace, routingYaml, generateData)
	if err != nil {
		t.Fatal(err)
	}
	err = unit_c3d.WaitOnPodInNamespace(appNamespace, "testpod", corev1.PodRunning)
	if err != nil {
		t.Error(err)
	}

	// TODO: add interactive session to connect all pods under various ports

	err = unit_c3d.Client.DeletePod(appNamespace, "testpod")
	if err != nil {
		t.Error(err)
	}

	err = unit_c3d.Client.DeleteNamespace(appNamespace)
	if err != nil {
		t.Error(err)
	}
}

func RecoverCrash1of3(t *testing.T) {
	sinceResourceVersion, err := unit_c3d.GetInnoDBClusterResourceVersion("mycluster")
	if err != nil {
		t.Fatal(err)
	}

	// kill mysqld (pid 1)
	err = unit_c3d.Client.Kill(unit_c3d.Namespace, "mycluster-0", k8s.Mysql, 11, 1)
	if err != nil {
		t.Fatal(err)
	}

	// wait for operator to notice it gone
	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:                 "mycluster",
		ExpectedStatus:       []string{"ONLINE_PARTIAL", "ONLINE_UNCERTAIN"},
		ExpectedNumOnline:    2,
		SinceResourceVersion: sinceResourceVersion,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	sinceResourceVersion, err = unit_c3d.GetInnoDBClusterResourceVersion("mycluster")
	if err != nil {
		t.Fatal(err)
	}

	// wait for operator to restore it
	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:                 "mycluster",
		ExpectedStatus:       []string{"ONLINE"},
		ExpectedNumOnline:    3,
		SinceResourceVersion: sinceResourceVersion,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err := unit_c3d.AssertGotClusterEvent(t,
		"mycluster", sinceResourceVersion, "Normal",
		"StatusChange", "Cluster status changed to ONLINE_PARTIAL. 2 member\\(s\\) ONLINE"); err != nil {
		t.Fatal(err)
	}
	if err := unit_c3d.AssertGotClusterEvent(t,
		"mycluster", sinceResourceVersion, "Normal",
		"StatusChange", "Cluster status changed to ONLINE. 3 member\\(s\\) ONLINE"); err != nil {
		t.Fatal(err)
	}

	params := unit_c3d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 3
	params.Primary = suite.NoPrimary
	params.RestartsExpected = true
	if _, err := suite.CheckAll(unit_c3d, params); err != nil {
		t.Fatal(err)
	}
}

func RecoverCrash2of3(t *testing.T) {
	sinceResourceVersion, err := unit_c3d.GetInnoDBClusterResourceVersion("mycluster")
	if err != nil {
		t.Fatal(err)
	}

	// kill mysqld (pid 1)
	err = unit_c3d.Client.Kill(unit_c3d.Namespace, "mycluster-1", k8s.Mysql, 11, 1)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_c3d.Client.Kill(unit_c3d.Namespace, "mycluster-0", k8s.Mysql, 11, 1)
	if err != nil {
		t.Fatal(err)
	}

	// wait for operator to notice them gone
	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"NO_QUORUM"},
		ExpectedNumOnline: -1,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	// wait for operator to restore it
	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 3,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err := unit_c3d.AssertGotClusterEvent(t,
		"mycluster", sinceResourceVersion, "Normal",
		"StatusChange", "Cluster status changed to NO_QUORUM. 0 member\\(s\\) ONLINE"); err != nil {
		t.Fatal(err)
	}

	if err := unit_c3d.AssertGotClusterEvent(t,
		"mycluster", sinceResourceVersion, "Normal",
		"RestoreQuorum", "Restoring quorum of cluster"); err != nil {
		t.Fatal(err)
	}

	if err := unit_c3d.AssertGotClusterEvent(t,
		"mycluster", sinceResourceVersion, "Normal",
		"StatusChange", "Cluster status changed to ONLINE_PARTIAL. 2 member\\(s\\) ONLINE"); err != nil {
		t.Fatal(err)
	}

	if err := unit_c3d.AssertGotClusterEvent(t,
		"mycluster", sinceResourceVersion, "Normal",
		"Rejoin", "Rejoining mycluster-0 to cluster"); err != nil {
		t.Fatal(err)
	}

	if err := unit_c3d.AssertGotClusterEvent(t,
		"mycluster", sinceResourceVersion, "Normal",
		"StatusChange", "Cluster status changed to ONLINE. 3 member\\(s\\) ONLINE"); err != nil {
		t.Fatal(err)
	}

	params := unit_c3d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 3
	params.Primary = 2
	params.RestartsExpected = true
	if _, err := suite.CheckAll(unit_c3d, params); err != nil {
		t.Fatal(err)
	}
}

func RecoverCrash3of3(t *testing.T) {
	sinceResourceVersion, err := unit_c3d.GetInnoDBClusterResourceVersion("mycluster")
	if err != nil {
		t.Fatal(err)
	}

	// kill mysqld (pid 1)
	if err := unit_c3d.Client.Kill(unit_c3d.Namespace, "mycluster-2", k8s.Mysql, 11, 1); err != nil {
		t.Fatal(err)
	}
	if err := unit_c3d.Client.Kill(unit_c3d.Namespace, "mycluster-1", k8s.Mysql, 11, 1); err != nil {
		t.Fatal(err)
	}
	if err := unit_c3d.Client.Kill(unit_c3d.Namespace, "mycluster-0", k8s.Mysql, 11, 1); err != nil {
		t.Fatal(err)
	}

	// wait for operator to notice them gone
	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:                 "mycluster",
		ExpectedStatus:       []string{"OFFLINE"},
		ExpectedNumOnline:    0,
		SinceResourceVersion: sinceResourceVersion,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	sinceResourceVersion, err = unit_c3d.GetInnoDBClusterResourceVersion("mycluster")
	if err != nil {
		t.Fatal(err)
	}

	// wait for operator to restore it
	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:                 "mycluster",
		ExpectedStatus:       []string{"ONLINE"},
		ExpectedNumOnline:    3,
		SinceResourceVersion: sinceResourceVersion,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_c3d.WaitOnRouters("mycluster", 2); err != nil {
		t.Fatal(err)
	}

	params := unit_c3d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 3
	params.Routers = 2
	params.Primary = suite.NoPrimary
	params.RestartsExpected = true
	if _, err := suite.CheckAll(unit_c3d, params); err != nil {
		t.Fatal(err)
	}
}

func RecoverDelete1of3(t *testing.T) {
	// delete the PRIMARY
	err := unit_c3d.Client.DeletePod(unit_c3d.Namespace, "mycluster-0")
	if err != nil {
		t.Error(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE_PARTIAL", "ONLINE_UNCERTAIN"},
		ExpectedNumOnline: 2,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	// wait for operator to restore everything
	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 3,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_c3d.WaitOnRouters("mycluster", 2); err != nil {
		t.Fatal(err)
	}

	pod0, err := unit_c3d.Client.GetPod(unit_c3d.Namespace, "mycluster-0")
	if err != nil {
		t.Fatal(err)
	}

	// the pod was deleted, so restarts resets to 0
	pod0RestartCount := pod0.Status.ContainerStatuses[0].RestartCount
	if pod0RestartCount != 0 {
		t.Fatalf("pod0 expected restart count is 0 but got %d", pod0RestartCount)
	}

	params := unit_c3d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 3
	params.Routers = 2
	params.Primary = suite.NoPrimary
	params.RestartsExpected = true
	if _, err := suite.CheckAll(unit_c3d, params); err != nil {
		t.Fatal(err)
	}

	err = unit_c3d.Client.Execute(unit_c3d.Namespace, "mycluster-0", k8s.Sidecar,
		"mysqlsh", "root:sakila@localhost", "--",
		"cluster", "set-primary-instance",
		fmt.Sprintf("mycluster-0.mycluster-instances.%s.svc.cluster.local:3306", unit_c3d.Namespace))
	if err != nil {
		t.Fatal(err)
	}

	if err := suite.CrossSyncGtids(
		unit_c3d.Namespace, []string{"mycluster-0", "mycluster-1", "mycluster-2"},
		"root", "sakila"); err != nil {
		t.Fatal(err)
	}

	params = unit_c3d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 3
	params.Primary = 0
	params.RestartsExpected = true
	all_pods, err := suite.CheckAll(unit_c3d, params)
	if err != nil {
		t.Fatal(err)
	}

	if err := suite.CheckData(all_pods, common.RootUser, common.RootPassword, 0); err != nil {
		t.Fatal(err)
	}
}

func RecoverDelete2of3(t *testing.T) {
	pod0, err := unit_c3d.Client.GetPod(unit_c3d.Namespace, "mycluster-0")
	if err != nil {
		t.Fatal(err)
	}
	p0ts := pod0.GetCreationTimestamp()

	pod1, err := unit_c3d.Client.GetPod(unit_c3d.Namespace, "mycluster-1")
	if err != nil {
		t.Fatal(err)
	}
	p1ts := pod1.GetCreationTimestamp()

	sinceResourceVersion, err := unit_c3d.GetInnoDBClusterResourceVersion("mycluster")
	if err != nil {
		t.Fatal(err)
	}

	pod0ResourceVersion := pod0.GetResourceVersion()
	err = unit_c3d.Client.DeletePodWithTimeout(unit_c3d.Namespace, "mycluster-0", 200)
	if err != nil {
		t.Fatal(err)
	}

	// extra timeout because the deletion of the 2nd pod will be blocked by
	// the busy handlers from the 1st deletion
	pod1ResourceVersion := pod1.GetResourceVersion()
	err = unit_c3d.Client.DeletePodWithTimeout(unit_c3d.Namespace, "mycluster-1", 200)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_c3d.WaitOnPodSince("mycluster-0", pod0ResourceVersion, corev1.PodPending)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_c3d.WaitOnPodSince("mycluster-1", pod1ResourceVersion, corev1.PodPending)
	if err != nil {
		t.Fatal(err)
	}

	// wait for operator to restore everything
	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:                 "mycluster",
		ExpectedStatus:       []string{"ONLINE_PARTIAL", "ONLINE_UNCERTAIN"},
		ExpectedNumOnline:    -1,
		SinceResourceVersion: sinceResourceVersion,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	sinceResourceVersion, err = unit_c3d.GetInnoDBClusterResourceVersion("mycluster")
	if err != nil {
		t.Fatal(err)
	}

	err = unit_c3d.WaitOnPod("mycluster-0", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_c3d.WaitOnPod("mycluster-1", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	// wait for operator to restore everything
	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:                 "mycluster",
		ExpectedStatus:       []string{"ONLINE"},
		ExpectedNumOnline:    3,
		SinceResourceVersion: sinceResourceVersion,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	// the pods were deleted, which means they would cleanly shutdown and
	// removed from the cluster
	if err := unit_c3d.AssertGotClusterEvent(t,
		"mycluster", sinceResourceVersion, "Normal",
		"Join", "Joining mycluster-0 to cluster"); err != nil {
		t.Fatal(err)
	}
	if err := unit_c3d.AssertGotClusterEvent(t,
		"mycluster", sinceResourceVersion, "Normal",
		"StatusChange",
		"Cluster status changed to ONLINE_PARTIAL. 2 member\\(s\\) ONLINE"); err != nil {
		t.Fatal(err)
	}
	if err := unit_c3d.AssertGotClusterEvent(t,
		"mycluster", sinceResourceVersion, "Normal",
		"Join", "Joining mycluster-1 to cluster"); err != nil {
		t.Fatal(err)
	}
	if err := unit_c3d.AssertGotClusterEvent(t,
		"mycluster", sinceResourceVersion, "Normal",
		"StatusChange",
		"Cluster status changed to ONLINE. 3 member\\(s\\) ONLINE"); err != nil {
		t.Fatal(err)
	}

	// make sure that the pods were actually deleted and recreated
	pod0, err = unit_c3d.Client.GetPod(unit_c3d.Namespace, "mycluster-0")
	if err != nil {
		t.Fatal(err)
	}
	pod0CreationTimestamp := pod0.GetCreationTimestamp()
	if p0ts.Time.After(pod0CreationTimestamp.Time) {
		t.Fatalf("pod %s should be recreated but its timestamp is %s greater than %s", pod0.GetName(), p0ts, pod0CreationTimestamp)
	}

	pod1, err = unit_c3d.Client.GetPod(unit_c3d.Namespace, "mycluster-1")
	if err != nil {
		t.Fatal(err)
	}
	pod1CreationTimestamp := pod1.GetCreationTimestamp()
	if p1ts.Time.After(pod1CreationTimestamp.Time) {
		t.Fatalf("pod %s should be recreated but its timestamp is %s greater than %s", pod1.GetName(), p1ts, pod1CreationTimestamp)
	}

	// the pod was deleted, so restarts resets to 0
	pod0RestartCount := pod0.Status.ContainerStatuses[0].RestartCount
	if pod0RestartCount != 0 {
		t.Fatalf("pod %s expected restart count is 0 but got %d", pod0.GetName(), pod0RestartCount)
	}

	pod1RestartCount := pod1.Status.ContainerStatuses[0].RestartCount
	if pod1RestartCount != 0 {
		t.Fatalf("pod %s expected restart count is 0 but got %d", pod1.GetName(), pod1RestartCount)
	}

	if err := suite.CrossSyncGtids(
		unit_c3d.Namespace, []string{"mycluster-2", "mycluster-0", "mycluster-1"},
		"root", "sakila"); err != nil {
		t.Error(err)
	}
	if err := suite.CrossSyncGtids(
		unit_c3d.Namespace, []string{"mycluster-1", "mycluster-2", "mycluster-0"},
		"root", "sakila"); err != nil {
		t.Error(err)
	}
	if err := suite.CrossSyncGtids(
		unit_c3d.Namespace, []string{"mycluster-0", "mycluster-2", "mycluster-1"},
		"root", "sakila"); err != nil {
		t.Error(err)
	}

	params := unit_c3d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 3
	params.Routers = suite.NoRouters
	params.Primary = 2
	params.RestartsExpected = true
	all_pods, err := suite.CheckAll(unit_c3d, params)
	if err != nil {
		t.Fatal(err)
	}

	if err := suite.CheckData(all_pods, common.RootUser, common.RootPassword, params.Primary); err != nil {
		t.Fatal(err)
	}

	err = unit_c3d.Client.Execute(unit_c3d.Namespace, "mycluster-0", k8s.Sidecar,
		"mysqlsh", "root:sakila@localhost", "--", "cluster", "set-primary-instance",
		fmt.Sprintf("mycluster-0.mycluster-instances.%s.svc.cluster.local:3306", unit_c3d.Namespace))
	if err != nil {
		t.Fatal(err)
	}
}

func RecoverDeleteAndWipe1of3(t *testing.T) {
	// delete the pv and pvc first, which will block because until the pod
	// is deleted

	// delete a secondary
	err := unit_c3d.Client.DeletePod(unit_c3d.Namespace, "mycluster-1")
	if err != nil {
		t.Error(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE_PARTIAL", "ONLINE_UNCERTAIN"},
		ExpectedNumOnline: 2,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	// wait for operator to restore everything
	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 3,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_c3d.WaitOnRouters("mycluster", 2); err != nil {
		t.Fatal(err)
	}

	pod1, err := unit_c3d.Client.GetPod(unit_c3d.Namespace, "mycluster-1")
	if err != nil {
		t.Fatal(err)
	}

	// the pod was deleted, so restarts resets to 0
	pod1RestartCount := pod1.Status.ContainerStatuses[0].RestartCount
	if pod1RestartCount != 0 {
		t.Fatalf("pod1 expected restart count is 0 but got %d", pod1RestartCount)
	}

	params := unit_c3d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 3
	params.Routers = 2
	params.Primary = 0
	params.RestartsExpected = true
	all_pods, err := suite.CheckAll(unit_c3d, params)
	if err != nil {
		t.Fatal(err)
	}

	if err := suite.CheckData(all_pods, common.RootUser, common.RootPassword, 0); err != nil {
		t.Fatal(err)
	}
}

func RecoverStop1of3(t *testing.T) {
	// Manually stop GR in 1 instance out of 3.
	t.Skip("TODO decide what to do, leave alone or restore?")
	podSessions0, err := mysql.NewSession(unit_c3d.Namespace, "mycluster-1", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer podSessions0.Close()
	if _, err := podSessions0.Exec("stop group_replication"); err != nil {
		t.Fatal(err)
	}

	// wait for operator to notice it OFFLINE
	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE_PARTIAL"},
		ExpectedNumOnline: 2,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	// restart GR and wait until everything is back to normal
	podSessions1, err := mysql.NewSession(unit_c3d.Namespace, "mycluster-1", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer podSessions1.Close()
	if _, err := podSessions1.Exec("start group_replication"); err != nil {
		t.Fatal(err)
	}

	// wait for operator to restore everything
	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 3,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	params := unit_c3d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 3
	params.Primary = 0
	if _, err := suite.CheckAll(unit_c3d, params); err != nil {
		t.Fatal(err)
	}
}

func RecoverStop2of3(t *testing.T) {
	t.Skip("under construction")
	s0, err := mysql.NewSession(unit_c3d.Namespace, "mycluster-0", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer s0.Close()

	s2, err := mysql.NewSession(unit_c3d.Namespace, "mycluster-2", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()

	if _, err := s0.Exec("stop group_replication"); err != nil {
		t.Fatal(err)
	}

	if _, err := s2.Exec("stop group_replication"); err != nil {
		t.Fatal(err)
	}

	// wait for operator to notice it ONLINE_PARTIAL
	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE_PARTIAL"},
		ExpectedNumOnline: 1,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	// wait for operator to restore everything
	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 3,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	params := unit_c3d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 3
	params.Primary = 1
	if _, err := suite.CheckAll(unit_c3d, params); err != nil {
		t.Fatal(err)
	}
}

func RecoverStop3of3(t *testing.T) {
	t.Skip("under construction")
	s0, err := mysql.NewSession(unit_c3d.Namespace, "mycluster-0", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer s0.Close()

	s1, err := mysql.NewSession(unit_c3d.Namespace, "mycluster-1", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer s1.Close()

	s2, err := mysql.NewSession(unit_c3d.Namespace, "mycluster-2", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()

	if _, err := s0.Exec("stop group_replication"); err != nil {
		t.Fatal(err)
	}

	if _, err := s1.Exec("stop group_replication"); err != nil {
		t.Fatal(err)
	}

	if _, err := s2.Exec("stop group_replication"); err != nil {
		t.Fatal(err)
	}

	// wait for operator to notice it OFFLINE
	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"OFFLINE"},
		ExpectedNumOnline: 0,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	// wait for operator to restore everything
	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 3,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	params := unit_c3d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 3
	params.Primary = 0
	if _, err := suite.CheckAll(unit_c3d, params); err != nil {
		t.Fatal(err)
	}
}

func RecoverRestart1of3(t *testing.T) {
	s0, err := mysql.NewSession(unit_c3d.Namespace, "mycluster-0", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer s0.Close()
	if _, err := s0.Exec("restart"); err != nil {
		t.Fatal(err)
	}

	// wait for operator to notice it OFFLINE
	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE_PARTIAL"},
		ExpectedNumOnline: -1,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	// wait for operator to restore everything
	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 3,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_c3d.WaitOnRouters("mycluster", 2); err != nil {
		t.Fatal(err)
	}

	params := unit_c3d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 3
	params.Routers = 2
	params.Primary = suite.NoPrimary
	params.RestartsExpected = true
	if _, err := suite.CheckAll(unit_c3d, params); err != nil {
		t.Fatal(err)
	}
}

func RecoverRestart2of3(t *testing.T) {
	s0, err := mysql.NewSession(unit_c3d.Namespace, "mycluster-0", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer s0.Close()
	if _, err := s0.Exec("restart"); err != nil {
		t.Fatal(err)
	}

	s2, err := mysql.NewSession(unit_c3d.Namespace, "mycluster-2", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	if _, err := s2.Exec("restart"); err != nil {
		t.Fatal(err)
	}

	// wait for operator to notice it ONLINE_PARTIAL
	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE_PARTIAL"},
		ExpectedNumOnline: 1,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	// check status of each pod

	// wait for operator to restore everything
	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 3,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	params := unit_c3d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 3
	params.Routers = suite.NoRouters
	params.Primary = 1
	params.RestartsExpected = true
	if _, err := suite.CheckAll(unit_c3d, params); err != nil {
		t.Fatal(err)
	}
}

func RecoverRestart3of3(t *testing.T) {
	s0, err := mysql.NewSession(unit_c3d.Namespace, "mycluster-0", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer s0.Close()
	if _, err := s0.Exec("restart"); err != nil {
		t.Fatal(err)
	}

	s1, err := mysql.NewSession(unit_c3d.Namespace, "mycluster-1", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer s1.Close()
	if _, err := s1.Exec("restart"); err != nil {
		t.Fatal(err)
	}

	s2, err := mysql.NewSession(unit_c3d.Namespace, "mycluster-2", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	if _, err := s2.Exec("restart"); err != nil {
		t.Fatal(err)
	}

	// wait for operator to notice it OFFLINE
	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"OFFLINE"},
		ExpectedNumOnline: 0,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	// wait for operator to restore everything
	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 3,
	}
	err = unit_c3d.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_c3d.WaitOnRouters("mycluster", 2); err != nil {
		t.Fatal(err)
	}

	params := unit_c3d.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 3
	params.Routers = 2
	params.Primary = suite.NoPrimary
	params.RestartsExpected = true
	all_pods, err := suite.CheckAll(unit_c3d, params)
	if err != nil {
		t.Fatal(err)
	}

	if err := suite.CheckData(all_pods, common.RootUser, common.RootPassword, 0); err != nil {
		t.Fatal(err)
	}
}

func AfterCluster3Defaults(t *testing.T) {
	err := unit_c3d.Client.DeleteInnoDBCluster(unit_c3d.Namespace, "mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_c3d.WaitOnPodGone("mycluster-2")
	if err != nil {
		t.Error(err)
	}

	err = unit_c3d.WaitOnPodGone("mycluster-1")
	if err != nil {
		t.Error(err)
	}

	err = unit_c3d.WaitOnPodGone("mycluster-0")
	if err != nil {
		t.Error(err)
	}

	err = unit_c3d.WaitOnInnoDBClusterGone("mycluster")
	if err != nil {
		t.Error(err)
	}
}

func TestCluster3Defaults(t *testing.T) {
	const Namespace = "cluster3-defaults"
	const NamespaceApp = "appns"
	var err error
	unit_c3d, err = suit.NewUnitSetupWithAuxNamespace(Namespace, NamespaceApp)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("CreateClusterThreeInstances=0", CreateClusterThreeInstances)
	t.Run("CheckVersion3=1", CheckVersion3)
	t.Run("CheckAccounts3=1", CheckAccounts3)
	t.Run("CheckRouting=2", CheckRouting)
	t.Run("RecoverCrash1of3=3", RecoverCrash1of3)
	t.Run("RecoverCrash2of3=3", RecoverCrash2of3)
	t.Run("RecoverCrash3of3=3", RecoverCrash3of3)
	t.Run("RecoverDelete1of3=3", RecoverDelete1of3)
	t.Run("RecoverDelete2of3=3", RecoverDelete2of3)
	t.Run("RecoverDeleteAndWipe1of3=3", RecoverDeleteAndWipe1of3)
	// t.Run("RecoverStop1of3=3", RecoverStop1of3)
	// t.Run("RecoverStop2of3=3", RecoverStop2of3)
	// t.Run("RecoverStop3of3=3", RecoverStop3of3)
	t.Run("RecoverRestart1of3=3", RecoverRestart1of3)
	t.Run("RecoverRestart2of3=3", RecoverRestart2of3)
	t.Run("RecoverRestart3of3=3", RecoverRestart3of3)
	t.Run("AfterCluster3Defaults=9", AfterCluster3Defaults)

	err = unit_c3d.Teardown()
	if err != nil {
		t.Error(err)
	}
}
