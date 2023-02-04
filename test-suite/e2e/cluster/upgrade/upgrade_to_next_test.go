// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package upgrade_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/marinesovitch/ote/test-suite/util/common"
	"github.com/marinesovitch/ote/test-suite/util/k8s"
	"github.com/marinesovitch/ote/test-suite/util/suite"

	corev1 "k8s.io/api/core/v1"
)

// upgrade to a newer version
var unit_utn *suite.Unit

type GenerateUpgradeToNextData struct {
	OldVersionTag string
}

var oldVersionTag string
var defaultVersionTag string

func BeforeUpgradeToNext(t *testing.T) {
	oldVersionTag = unit_utn.Cfg.Images.MinSupportedMysqlVersion
	defaultVersionTag = unit_utn.Cfg.Images.DefaultVersionTag

	err := unit_utn.Client.CreateUserSecrets(
		unit_utn.Namespace, "mypwds", common.RootUser, common.DefaultHost, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}

	generateData := GenerateUpgradeToNextData{
		OldVersionTag: oldVersionTag,
	}
	err = unit_utn.GenerateAndApply("upgrade-to-next.yaml", generateData)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_utn.WaitOnPod("mycluster-0", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_utn.WaitOnPod("mycluster-1", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_utn.WaitOnPod("mycluster-2", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 3,
	}
	err = unit_utn.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Error(err)
	}

	if err = unit_utn.WaitOnRouters("mycluster", 2); err != nil {
		t.Fatal(err)
	}

	params := unit_utn.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 3
	params.Routers = 2
	params.Primary = 0
	params.Version = oldVersionTag
	if _, err := suite.CheckAll(unit_utn, params); err != nil {
		t.Fatal(err)
	}

	for _, podName := range []string{"mycluster-0", "mycluster-1", "mycluster-2"} {
		pod, err := unit_utn.Client.GetPod(unit_utn.Namespace, podName)
		if err != nil {
			t.Fatal(err)
		}
		mysqlCont, err := suite.CheckPodContainer(unit_utn.Client, pod, k8s.Mysql, 0, true)
		if err != nil {
			t.Fatal(err)
		}
		mysqlContImage := mysqlCont.Container.Image
		expectedMysqlContImage := unit_utn.GetServerImage(oldVersionTag)
		if mysqlContImage != expectedMysqlContImage {
			t.Fatalf("expected mysql container image in pod %s is %s but got %s", podName, expectedMysqlContImage, mysqlContImage)
		}

		sidecarCont, err := suite.CheckPodContainer(unit_utn.Client, pod, k8s.Sidecar, suite.NoRestarts, true)
		if err != nil {
			t.Fatal(err)
		}
		sidecarContImage := sidecarCont.Container.Image
		expectedSidecarContImage := unit_utn.GetDefaultOperatorImage()
		if sidecarContImage != expectedSidecarContImage {
			t.Fatalf("expected sidecar container image in pod %s is %s but got %s", podName, expectedSidecarContImage, sidecarContImage)
		}
	}
}

func UpgradeToNext(t *testing.T) {
	// version is now 8.0.{VERSION} but we upgrade it to 8.0.{VERSION+1}
	// This will upgrade MySQL only, not the Router since it's already the latest.
	patch := k8s.JsonPatch{
		Operation: k8s.PatchReplace,
		Path:      "/spec/version",
		Value:     defaultVersionTag,
	}
	err := unit_utn.Client.JSONPatchInnoDBCluster(unit_utn.Namespace, "mycluster", patch)
	if err != nil {
		t.Fatal(err)
	}

	checkDone := func(args ...interface{}) (bool, error) {
		name := args[0].(string)
		pod, err := unit_utn.Client.GetPod(unit_utn.Namespace, name)
		if err != nil {
			return false, err
		}
		annotations := pod.GetAnnotations()
		const membershipInfoKey = "mysql.oracle.com/membership-info"
		jsonMembershipInfo, ok := annotations[membershipInfoKey]
		if !ok {
			return false, nil
		}

		var membershipInfo map[string]interface{}
		err = json.Unmarshal([]byte(jsonMembershipInfo), &membershipInfo)
		if err != nil {
			return false, nil
		}
		version := membershipInfo["version"].(string)
		if !strings.HasPrefix(version, defaultVersionTag) {
			return false, nil
		}

		return true, nil
	}

	if _, err := unit_utn.Wait(checkDone, 150, 10, "mycluster-2"); err != nil {
		t.Fatal(err)
	}
	if _, err := unit_utn.Wait(checkDone, 150, 10, "mycluster-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := unit_utn.Wait(checkDone, 150, 10, "mycluster-0"); err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 3,
	}
	err = unit_utn.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Error(err)
	}

	if err = unit_utn.WaitOnRouters("mycluster", 2); err != nil {
		t.Fatal(err)
	}

	params := unit_utn.GetDefaultCheckParams()
	params.Name = "mycluster"
	params.Instances = 3
	params.Routers = 2
	params.Version = defaultVersionTag
	if _, err := suite.CheckAll(unit_utn, params); err != nil {
		t.Fatal(err)
	}

	for _, podName := range []string{"mycluster-0", "mycluster-1", "mycluster-2"} {
		pod, err := unit_utn.Client.GetPod(unit_utn.Namespace, podName)
		if err != nil {
			t.Fatal(err)
		}
		mysqlCont, err := suite.CheckPodContainer(unit_utn.Client, pod, k8s.Mysql, suite.NoRestarts, true)
		if err != nil {
			t.Fatal(err)
		}

		mysqlContImage := mysqlCont.Container.Image
		expectedMysqlContImage := unit_utn.GetDefaultServerImage()
		if mysqlContImage != expectedMysqlContImage {
			t.Fatalf("expected mysql container image in pod %s is %s but got %s", podName, expectedMysqlContImage, mysqlContImage)
		}

		sidecarCont, err := suite.CheckPodContainer(unit_utn.Client, pod, k8s.Sidecar, suite.NoRestarts, true)
		if err != nil {
			t.Fatal(err)
		}
		sidecarContImage := sidecarCont.Container.Image
		expectedSidecarContImage := unit_utn.GetDefaultOperatorImage()
		if sidecarContImage != expectedSidecarContImage {
			t.Fatalf("expected sidecar container image in pod %s is %s but got %s", podName, expectedSidecarContImage, sidecarContImage)
		}
	}
}

func AfterUpgradeToNext(t *testing.T) {
	err := unit_utn.Client.DeleteInnoDBCluster(unit_utn.Namespace, "mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_utn.WaitOnPodGone("mycluster-2")
	if err != nil {
		t.Error(err)
	}

	err = unit_utn.WaitOnPodGone("mycluster-1")
	if err != nil {
		t.Error(err)
	}

	err = unit_utn.WaitOnPodGone("mycluster-0")
	if err != nil {
		t.Error(err)
	}

	if err := unit_utn.WaitOnInnoDBClusterGone("mycluster"); err != nil {
		t.Error(err)
	}

	if err := unit_utn.Client.DeleteSecret(unit_utn.Namespace, "mypwds"); err != nil {
		t.Error(err)
	}
}

func TestUpgradeToNext(t *testing.T) {
	const Namespace = "upgrade-to-next"
	var err error
	unit_utn, err = suit.NewUnitSetup(Namespace)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("BeforeUpgradeToNext", BeforeUpgradeToNext)
	t.Run("UpgradeToNext", UpgradeToNext)
	t.Run("AfterUpgradeToNext", AfterUpgradeToNext)

	err = unit_utn.Teardown()
	if err != nil {
		t.Error(err)
	}
}
