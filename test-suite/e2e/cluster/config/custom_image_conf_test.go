// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package config_test

import (
	"testing"

	"github.com/marinesovitch/ote/test-suite/util/auxi"
	"github.com/marinesovitch/ote/test-suite/util/common"
	"github.com/marinesovitch/ote/test-suite/util/k8s"
	"github.com/marinesovitch/ote/test-suite/util/suite"

	corev1 "k8s.io/api/core/v1"
)

var unit_ci *suite.Unit

type GenerateCustomImageConfData struct {
	ServerVersion string
}

func CreateCustomImageConf(t *testing.T) {
	// Checks:
	// - imagePullSecrets is propagated
	// - version is propagated
	err := unit_ci.Client.CreateUserSecrets(
		unit_ci.Namespace, "mypwds", common.AdminUser, common.DefaultHost, common.AdminPassword)
	if err != nil {
		t.Fatal(err)
	}

	// create cluster with mostly default configs but a specific server version
	generateData := GenerateCustomImageConfData{
		ServerVersion: oldVersionTag,
	}
	err = unit_ci.GenerateAndApply("custom-image-conf.yaml", generateData)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_ci.WaitOnPod("mycluster-0", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 1,
	}
	err = unit_ci.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_ci.WaitOnRouters("mycluster", 1); err != nil {
		t.Fatal(err)
	}

	params := unit_ci.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 1
	params.Routers = 1
	params.Primary = 0
	params.User = common.AdminUser
	params.Password = "secret"
	if _, err := suite.CheckAll(unit_ci, params); err != nil {
		t.Fatal(err)
	}

	// check server pod
	Pod0Name := "mycluster-0"
	pod, err := unit_ci.Client.GetPod(unit_ci.Namespace, Pod0Name)
	if err != nil {
		t.Fatal(err)
	}

	// hardcoded value according to applied yaml
	icSpecPullSecrets := []string{"pullsecrets"}

	podSpecImagePullSecrets := pod.Spec.ImagePullSecrets
	podSpecPullSecretNames := make([]string, len(podSpecImagePullSecrets))
	for i, podSpecPullSecret := range podSpecImagePullSecrets {
		podSpecPullSecretNames[i] = podSpecPullSecret.Name
	}

	if !auxi.AreStringSlicesEqual(icSpecPullSecrets, podSpecPullSecretNames) {
		t.Fatalf("ic spec imagePullSecrets %v are different than pod spec %s imagePullSecrets %v",
			icSpecPullSecrets, Pod0Name, podSpecPullSecretNames)
	}

	mysqlCont, err := suite.CheckPodContainer(unit_ci.Client, pod, k8s.Mysql, suite.NoRestarts, true)
	if err != nil {
		t.Fatal(err)
	}
	mysqlContImage := mysqlCont.Container.Image
	expectedMysqlContImage := unit_ci.GetServerImage(oldVersionTag)
	if mysqlContImage != expectedMysqlContImage {
		t.Fatalf("expected mysql container image in pod %s is %s but got %s", Pod0Name, expectedMysqlContImage, mysqlContImage)
	}

	sidecarCont, err := suite.CheckPodContainer(unit_ci.Client, pod, k8s.Sidecar, suite.NoRestarts, true)
	if err != nil {
		t.Fatal(err)
	}
	sidecarContImage := sidecarCont.Container.Image
	expectedSidecarContImage := unit_ci.GetOperatorImage(unit_ci.Cfg.Operator.VersionTag)
	if sidecarContImage != expectedSidecarContImage {
		t.Fatalf("expected sidecar container image in pod %s is %s but got %s", Pod0Name, expectedSidecarContImage, sidecarContImage)
	}

	// check router pod
	routers, err := unit_ci.Client.ListPodsWithFilter(unit_ci.Namespace, "mycluster-.*-router")
	if err != nil {
		t.Fatal(err)
	}

	for _, router := range routers.Items {
		routerName := router.GetName()

		routerSpecImagePullSecrets := router.Spec.ImagePullSecrets
		routerSpecPullSecretNames := make([]string, len(routerSpecImagePullSecrets))
		for i, routerSpecPullSecret := range routerSpecImagePullSecrets {
			routerSpecPullSecretNames[i] = routerSpecPullSecret.Name
		}

		if !auxi.AreStringSlicesEqual(icSpecPullSecrets, routerSpecPullSecretNames) {
			t.Fatalf("ic spec imagePullSecrets %v are different than router spec %s imagePullSecrets %v",
				icSpecPullSecrets, routerName, routerSpecPullSecretNames)
		}

		routerCont, err := suite.CheckPodContainer(unit_ci.Client, &router, k8s.Router, suite.NoRestarts, true)
		if err != nil {
			t.Fatal(err)
		}
		routerContainerImage := routerCont.Container.Image
		expectedRouterContainerImage := unit_ci.GetDefaultRouterImage()
		if routerContainerImage != expectedRouterContainerImage {
			t.Fatalf("expected container image for router %s is %s but got %s", routerName, expectedRouterContainerImage, routerContainerImage)
		}
	}
}

func DestroyCustomImageConf(t *testing.T) {
	err := unit_ci.Client.DeleteInnoDBCluster(unit_ci.Namespace, "mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_ci.WaitOnPodGone("mycluster-0")
	if err != nil {
		t.Error(err)
	}

	err = unit_ci.WaitOnInnoDBClusterGone("mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_ci.Client.DeleteSecret(unit_ci.Namespace, "mypwds")
	if err != nil {
		t.Error(err)
	}
}

func TestClusterCustomImageConf(t *testing.T) {
	const Namespace = "custom-image-conf"
	var err error
	unit_ci, err = suit.NewUnitSetup(Namespace)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("CreateCustomImageConf=0", CreateCustomImageConf)
	t.Run("DestroyCustomImageConf=1", DestroyCustomImageConf)

	err = unit_ci.Teardown()
	if err != nil {
		t.Error(err)
	}
}
