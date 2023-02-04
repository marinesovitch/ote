// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package enterprise_test

import (
	"strings"
	"testing"

	"github.com/marinesovitch/ote/test-suite/util/auxi"
	"github.com/marinesovitch/ote/test-suite/util/common"
	"github.com/marinesovitch/ote/test-suite/util/k8s"
	"github.com/marinesovitch/ote/test-suite/util/mysql"
	"github.com/marinesovitch/ote/test-suite/util/suite"

	corev1 "k8s.io/api/core/v1"
)

var unit_cct *suite.Unit

func Create(t *testing.T) {
	// Create cluster, check posted events.

	err := unit_cct.Client.CreateUserSecrets(unit_cct.Namespace, "mypwds", common.RootUser, common.DefaultHost, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}

	// create cluster with mostly default configs
	err = unit_cct.Apply("enterprise-cluster.yaml")
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"PENDING", "INITIALIZING"},
		ExpectedNumOnline: 0,
	}
	err = unit_cct.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_cct.WaitOnPod("mycluster-0", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_cct.WaitOnPod("mycluster-1", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_cct.WaitOnPod("mycluster-2", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams = k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 3,
	}
	err = unit_cct.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_cct.WaitOnRouters("mycluster", 2); err != nil {
		t.Fatal(err)
	}

	params := unit_cct.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 3
	params.Routers = 2
	params.Primary = 0
	if _, err := suite.CheckAll(unit_cct, params); err != nil {
		t.Fatal(err)
	}

	if err := unit_cct.AssertGotClusterEvent(t, "mycluster", common.AnyResourceVersion,
		"Normal", "ResourcesCreated", "Dependency resources created, switching status to PENDING"); err != nil {
		t.Fatal(err)
	}
	if err := unit_cct.AssertGotClusterEvent(t, "mycluster", common.AnyResourceVersion,
		"Normal", "StatusChange", "Cluster status changed to INITIALIZING. 0 member\\(s\\) ONLINE"); err != nil {
		t.Fatal(err)
	}
	if err := unit_cct.AssertGotClusterEvent(t, "mycluster", common.AnyResourceVersion,
		"Normal", "StatusChange", "Cluster status changed to ONLINE. 1 member\\(s\\) ONLINE"); err != nil {
		t.Fatal(err)
	}
}

func CheckAccounts(t *testing.T) {
	session, err := mysql.NewSession(unit_cct.Namespace, "mycluster-0", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	rawAccounts, err := session.FetchAll("SELECT concat(user,'@',host) FROM mysql.user")
	if err != nil {
		t.Fatal(err)
	}
	accounts := rawAccounts.ToStringsSlice(0)

	expectedAccounts := []string{
		"root@%", "localroot@localhost", "mysqladmin@%",
		"mysqlbackup@%", "mysqlrouter@%",
		"mysqlhealthchecker@localhost", "mysql_innodb_cluster_1000@%",
		"mysql_innodb_cluster_1001@%", "mysql_innodb_cluster_1002@%"}
	expectedAccounts = append(expectedAccounts, common.DefaultMysqlAccounts...)

	if !auxi.AreStringSlicesEqual(accounts, expectedAccounts) {
		t.Fatalf("expected accounts are %v but got %v", expectedAccounts, accounts)
	}
}

func CheckVersion(t *testing.T) {
	// ensure containers have the right version and edition
	pod, err := unit_cct.Client.GetPod(unit_cct.Namespace, "mycluster-0")
	if err != nil {
		t.Fatal(err)
	}

	initmysqlCont, err := k8s.GetContainer(pod, k8s.InitMysql)
	if err != nil {
		t.Fatal(err)
	}
	image := initmysqlCont.Image
	expectedVersionTag := ":" + unit_cct.Cfg.Images.DefaultVersionTag
	if !strings.Contains(image, expectedVersionTag) {
		t.Fatalf("initmysql container image should contain %s but it doesn't %s", expectedVersionTag, image)
	}

	expectedImageName := unit_cct.Cfg.Images.MysqlServerEEImage + ":"
	if !strings.Contains(image, expectedImageName) {
		t.Fatalf("initmysql container image should contain %s but it doesn't %s", expectedImageName, image)
	}

	initconfCont, err := k8s.GetContainer(pod, k8s.InitConf)
	if err != nil {
		t.Fatal(err)
	}
	image = initconfCont.Image
	expectedVersionTag = ":" + unit_cct.Cfg.Operator.VersionTag
	if !strings.Contains(image, expectedVersionTag) {
		t.Fatalf("initconf container image should contain %s but it doesn't %s", expectedVersionTag, image)
	}

	expectedImageName = unit_cct.Cfg.Operator.ImageEE + ":"
	if !strings.Contains(image, expectedImageName) {
		t.Fatalf("initconf container image should contain %s but it doesn't %s", expectedImageName, image)
	}

	mysqlCont, err := k8s.GetContainer(pod, k8s.Mysql)
	if err != nil {
		t.Fatal(err)
	}
	image = mysqlCont.Image
	expectedVersionTag = ":" + unit_cct.Cfg.Images.DefaultVersionTag
	if !strings.Contains(image, expectedVersionTag) {
		t.Fatalf("mysql container image should contain %s but it doesn't %s", expectedVersionTag, image)
	}

	expectedImageName = unit_cct.Cfg.Images.MysqlServerEEImage + ":"
	if !strings.Contains(image, expectedImageName) {
		t.Fatalf("mysql container image should contain %s but it doesn't %s", expectedImageName, image)
	}

	sidecarCont, err := k8s.GetContainer(pod, k8s.Sidecar)
	if err != nil {
		t.Fatal(err)
	}
	image = sidecarCont.Image
	expectedVersionTag = ":" + unit_cct.Cfg.Operator.VersionTag
	if !strings.Contains(image, expectedVersionTag) {
		t.Fatalf("sidecar container image should contain %s but it doesn't %s", expectedVersionTag, image)
	}

	expectedImageName = unit_cct.Cfg.Operator.ImageEE + ":"
	if !strings.Contains(image, expectedImageName) {
		t.Fatalf("sidecar container image should contain %s but it doesn't %s", expectedImageName, image)
	}

	// check router version and edition
	routers, err := unit_cct.Client.ListPodsWithFilter(unit_cct.Namespace, "mycluster-router-.*")
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
	expectedVersionTag = ":" + unit_cct.Cfg.Images.DefaultVersionTag
	if !strings.Contains(image, expectedVersionTag) {
		t.Fatalf("router container image should contain %s but it doesn't %s", expectedVersionTag, image)
	}

	expectedImageName = unit_cct.Cfg.Images.MysqlRouterEEImage + ":"
	if !strings.Contains(image, expectedImageName) {
		t.Fatalf("router container image should contain %s but it doesn't %s", expectedImageName, image)
	}
}

func Destroy(t *testing.T) {
	err := unit_cct.Client.DeleteInnoDBCluster(unit_cct.Namespace, "mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_cct.WaitOnPodGone("mycluster-2")
	if err != nil {
		t.Error(err)
	}

	err = unit_cct.WaitOnPodGone("mycluster-1")
	if err != nil {
		t.Error(err)
	}

	err = unit_cct.WaitOnPodGone("mycluster-0")
	if err != nil {
		t.Error(err)
	}

	err = unit_cct.WaitOnInnoDBClusterGone("mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_cct.Client.DeleteSecret(unit_cct.Namespace, "mypwds")
	if err != nil {
		t.Error(err)
	}
}

func TestClusterEnterprise(t *testing.T) {
	const Namespace = "cluster-enterprise"
	var err error
	unit_cct, err = suit.NewUnitSetup(Namespace)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_cct.Cfg.CheckEnterpriseConfig()
	if err != nil {
		t.Skip(err)
	}

	t.Run("Create=0", Create)
	t.Run("CheckAccounts=1", CheckAccounts)
	t.Run("CheckVersion=1", CheckVersion)
	t.Run("Destroy=9", Destroy)

	err = unit_cct.Teardown()
	if err != nil {
		t.Error(err)
	}
}
