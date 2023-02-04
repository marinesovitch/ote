// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package initdb_test

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

var unit_fc *suite.Unit

const NamespaceClone = "clone"

func BeforeFromClone(t *testing.T) {
	err := unit_fc.Client.CreateUserSecrets(
		unit_fc.Namespace, "mypwds", common.RootUser, common.DefaultHost, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_fc.Apply("from-clone-cluster.yaml")
	if err != nil {
		t.Fatal(err)
	}

	err = unit_fc.WaitOnPod("mycluster-0", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_fc.WaitOnPod("mycluster-1", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_fc.WaitOnPod("mycluster-2", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              "mycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 3,
	}
	err = unit_fc.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	err = suite.LoadSakilaScript(unit_fc, "mycluster-0", k8s.Mysql)
	if err != nil {
		t.Fatal(err)
	}

	podSession, err := mysql.NewSession(unit_fc.Namespace, "mycluster-0", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer podSession.Close()
	if _, err = podSession.Exec("create user clone@'%' identified by 'clonepass'"); err != nil {
		t.Fatal(err)
	}
	if _, err = podSession.Exec("grant backup_admin on *.* to clone@'%'"); err != nil {
		t.Fatal(err)
	}
}

func CreateClone(t *testing.T) {
	if err := unit_fc.Client.CreateNamespace(NamespaceClone); err != nil {
		t.Fatal(err)
	}
	err := unit_fc.Client.CreateUserSecrets(
		NamespaceClone, "pwds", common.RootUser, common.DefaultHost, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	err = unit_fc.Client.CreateUserSecrets(
		NamespaceClone, "donorpwds", common.RootUser, common.DefaultHost, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}

	// create cluster with mostly default configs
	err = unit_fc.ApplyInNamespace(NamespaceClone, "create-clone.yaml")
	if err != nil {
		t.Fatal(err)
	}

	err = unit_fc.WaitOnPodInNamespace(NamespaceClone, "copycluster-0", corev1.PodRunning)
	if err != nil {
		t.Error(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Namespace:         NamespaceClone,
		Name:              "copycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 1,
		Timeout:           300,
	}
	err = unit_fc.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_fc.WaitOnRoutersInNamespace(NamespaceClone, "copycluster", 1); err != nil {
		t.Fatal(err)
	}

	podSession, err := mysql.NewSession(unit_fc.Namespace, "mycluster-0", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer podSession.Close()

	originalTables, err := podSession.FetchAll("show tables in sakila")
	if err != nil {
		t.Fatal(err)
	}

	cloneSession, err := mysql.NewSession(NamespaceClone, "copycluster-0", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer cloneSession.Close()

	clonedTables, err := cloneSession.FetchAll("show tables in sakila")
	if err != nil {
		t.Fatal(err)
	}

	// add some data with binlog disabled to make sure that all members of this
	// cluster are cloned
	commands := []string{
		"set autocommit=1",
		"set session sql_log_bin=0",
		"create schema unlogged_db",
		"create table unlogged_db.tbl (a int primary key)",
		"insert into unlogged_db.tbl values (42)",
		"set session sql_log_bin=1",
		"set autocommit=0",
	}
	for _, command := range commands {
		if _, err := cloneSession.Exec(command); err != nil {
			t.Fatal(err)
		}
	}

	originalTableNames := originalTables.ToStringsSlice(0)
	clonedTableNames := clonedTables.ToStringsSlice(0)
	if !auxi.AreStringSlicesEqual(originalTableNames, clonedTableNames) {
		t.Fatalf("expected tables: %v but got: %v", originalTableNames, clonedTableNames)
	}

	if err := suite.CheckRouterPods(unit_fc.Client, NamespaceClone, "copycluster", 1); err != nil {
		t.Fatal(err)
	}
}

func Grow(t *testing.T) {
	patch := k8s.JsonPatch{
		Operation: k8s.PatchReplace,
		Path:      "/spec/instances",
		Value:     2,
	}
	err := unit_fc.Client.JSONPatchInnoDBCluster(NamespaceClone, "copycluster", patch)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_fc.WaitOnPodInNamespace(NamespaceClone, "copycluster-1", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Namespace:         NamespaceClone,
		Name:              "copycluster",
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 2,
	}
	err = unit_fc.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	// check that the new instance was cloned
	cloneSession, err := mysql.NewSession(NamespaceClone, "copycluster-1", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer cloneSession.Close()

	result, err := cloneSession.FetchAll("select * from unlogged_db.tbl")
	if err != nil {
		t.Fatal(err)
	}
	records := result.ToStrings()
	if len(records) == 0 {
		t.Fatal("cannot get data from unlogged_db.tbl")
	}
	resultStr := strings.Join(records[0], "")
	expectedResultStr := "42"
	if resultStr != expectedResultStr {
		t.Fatalf("expected records [%s] but got [%s]", expectedResultStr, resultStr)
	}
}

func AfterFromClone(t *testing.T) {
	err := unit_fc.Client.DeleteInnoDBCluster(NamespaceClone, "copycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_fc.WaitOnPodGoneInNamespace(NamespaceClone, "copycluster-1")
	if err != nil {
		t.Error(err)
	}

	err = unit_fc.WaitOnPodGoneInNamespace(NamespaceClone, "copycluster-0")
	if err != nil {
		t.Error(err)
	}

	err = unit_fc.WaitOnInnoDBClusterGoneInNamespace(NamespaceClone, "copycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_fc.Client.DeleteInnoDBCluster(unit_fc.Namespace, "mycluster")
	if err != nil {
		t.Error(err)
	}

	err = unit_fc.WaitOnPodGone("mycluster-2")
	if err != nil {
		t.Error(err)
	}

	err = unit_fc.WaitOnPodGone("mycluster-1")
	if err != nil {
		t.Error(err)
	}

	err = unit_fc.WaitOnPodGone("mycluster-0")
	if err != nil {
		t.Error(err)
	}

	err = unit_fc.WaitOnInnoDBClusterGone("mycluster")
	if err != nil {
		t.Error(err)
	}
}

func TestClusterFromClone(t *testing.T) {
	const Namespace = "cluster-from-clone"
	var err error
	unit_fc, err = suit.NewUnitSetupWithAuxNamespace(Namespace, NamespaceClone)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("BeforeFromClone=0", BeforeFromClone)
	t.Run("CreateClone=1", CreateClone)
	t.Run("Grow=1", Grow)
	t.Run("AfterFromClone=9", AfterFromClone)

	err = unit_fc.Teardown()
	if err != nil {
		t.Error(err)
	}
}
