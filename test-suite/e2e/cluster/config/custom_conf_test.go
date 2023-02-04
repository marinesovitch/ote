// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package config_test

import (
	"testing"

	"github.com/marinesovitch/ote/test-suite/util/common"
	"github.com/marinesovitch/ote/test-suite/util/k8s"
	"github.com/marinesovitch/ote/test-suite/util/mysql"
	"github.com/marinesovitch/ote/test-suite/util/suite"

	corev1 "k8s.io/api/core/v1"
)

var unit_cc *suite.Unit

type GenerateCustomConfData struct {
	ClusterName   string
	ServerVersion string
}

const clusterName = "myvalid-cluster-name-28-char"

func CreateCustomConf(t *testing.T) {
	// Checks:
	// - cluster name can be 28chars long
	// - root user name and host can be customized
	// - base server id can be changed
	// - version can be customized
	// - mycnf can be specified
	err := unit_cc.Client.CreateUserSecrets(unit_cc.Namespace, "mypwds", common.AdminUser, common.DefaultHost, common.AdminPassword)
	if err != nil {
		t.Fatal(err)
	}

	// create cluster with mostly default configs but a specific server version
	generateData := GenerateCustomConfData{
		ClusterName:   clusterName,
		ServerVersion: oldVersionTag,
	}
	err = unit_cc.GenerateAndApply("custom-conf.yaml", generateData)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_cc.WaitOnPod(clusterName+"-0", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_cc.WaitOnPod(clusterName+"-1", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              clusterName,
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 2,
	}
	err = unit_cc.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_cc.WaitOnRouters(clusterName, 1); err != nil {
		t.Fatal(err)
	}

	params := unit_cc.GetDefaultCheckParams()
	params.Name = clusterName
	params.Instances = 2
	params.Routers = 1
	params.Primary = 0
	params.User = common.AdminUser
	params.Password = "secret"
	if _, err := suite.CheckAll(unit_cc, params); err != nil {
		t.Fatal(err)
	}

	session, err := mysql.NewSession(unit_cc.Namespace, clusterName+"-0", common.AdminUser, common.AdminPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	var aport, sid int
	var ver string
	err = session.QueryOne("select @@admin_port, @@server_id, @@version").Scan(&aport, &sid, &ver)
	if err != nil {
		t.Fatal(err)
	}

	const expectedAdminPort = 3333
	if aport != expectedAdminPort {
		t.Fatalf("expected admin port is %d but got %d", expectedAdminPort, aport)
	}

	expectedServerId := 3210
	if sid != expectedServerId {
		t.Fatalf("expected server id is %d but got %d", expectedServerId, sid)
	}

	if ver != oldVersionTag {
		t.Fatalf("expected version tag is %s but got %s", oldVersionTag, ver)
	}

	users, err := session.FetchAll("select user,host from mysql.user where user='root'")
	if err != nil {
		t.Fatal(err)
	}
	if len(users.Rows) != 0 {
		t.Fatalf("expected no users but got %d", len(users.Rows))
	}

	session, err = mysql.NewSession(unit_cc.Namespace, clusterName+"-1", common.AdminUser, common.AdminPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	err = session.QueryOne(
		"select @@admin_port, @@server_id, @@version").Scan(&aport, &sid, &ver)
	if err != nil {
		t.Fatal(err)
	}
	if aport != expectedAdminPort {
		t.Fatalf("expected admin port is %d but got %d", expectedAdminPort, aport)
	}

	expectedServerId = 3211
	if sid != expectedServerId {
		t.Fatalf("expected server id is %d but got %d", expectedServerId, sid)
	}

	if ver != oldVersionTag {
		t.Fatalf("expected version tag is %s but got %s", oldVersionTag, ver)
	}

	users, err = session.FetchAll("select user,host from mysql.user where user='root'")
	if err != nil {
		t.Fatal(err)
	}
	if len(users.Rows) != 0 {
		t.Fatalf("expected no users but got %d", len(users.Rows))
	}

	pod, err := unit_cc.Client.GetPod(unit_cc.Namespace, clusterName+"-0")
	if err != nil {
		t.Fatal(err)
	}
	mysqlCont, err := suite.CheckPodContainer(unit_cc.Client, pod, k8s.Mysql, suite.NoRestarts, true)
	if err != nil {
		t.Fatal(err)
	}
	mysqlContImage := mysqlCont.Container.Image
	expectedMysqlContImage := unit_cc.GetServerImage(oldVersionTag)
	if mysqlContImage != expectedMysqlContImage {
		t.Fatalf("expected mysql container image in pod %s is %s but got %s", pod.GetName(), expectedMysqlContImage, mysqlContImage)
	}

	sidecarCont, err := suite.CheckPodContainer(unit_cc.Client, pod, k8s.Sidecar, suite.NoRestarts, true)
	if err != nil {
		t.Fatal(err)
	}
	sidecarContImage := sidecarCont.Container.Image
	expectedSidecarContImage := unit_cc.GetDefaultOperatorImage()
	if sidecarContImage != expectedSidecarContImage {
		t.Fatalf("expected sidecar container image in pod %s is %s but got %s", pod.GetName(), expectedSidecarContImage, sidecarContImage)
	}

	// check version of router images
	routers, err := unit_cc.Client.ListPodsWithFilter(unit_cc.Namespace, clusterName+"-.*-router")
	if err != nil {
		t.Fatal(err)
	}
	for _, router := range routers.Items {
		routerCont, err := suite.CheckPodContainer(unit_cc.Client, &router, k8s.Router, suite.NoRestarts, true)
		if err != nil {
			t.Fatal(err)
		}

		routerContainerImage := routerCont.Container.Image
		expectedRouterContainerImage := unit_cc.GetDefaultRouterImage()
		if routerContainerImage != expectedRouterContainerImage {
			t.Fatalf("expected container image for router %s is %s but got %s", router.GetName(), expectedRouterContainerImage, routerContainerImage)
		}
	}
}

func DestroyCustomConf(t *testing.T) {
	err := unit_cc.Client.DeleteInnoDBCluster(unit_cc.Namespace, clusterName)
	if err != nil {
		t.Error(err)
	}

	err = unit_cc.WaitOnPodGone(clusterName + "-1")
	if err != nil {
		t.Error(err)
	}

	err = unit_cc.WaitOnPodGone(clusterName + "-0")
	if err != nil {
		t.Error(err)
	}

	err = unit_cc.WaitOnInnoDBClusterGone(clusterName)
	if err != nil {
		t.Error(err)
	}
}

func TestClusterCustomConf(t *testing.T) {
	const Namespace = "custom-conf"
	var err error
	unit_cc, err = suit.NewUnitSetup(Namespace)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("CreateCustomConf=0", CreateCustomConf)
	t.Run("DestroyCustomConf=1", DestroyCustomConf)

	err = unit_cc.Teardown()
	if err != nil {
		t.Error(err)
	}
}
