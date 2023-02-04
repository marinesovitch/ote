// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package initdb_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/marinesovitch/ote/test-suite/util/auxi"
	"github.com/marinesovitch/ote/test-suite/util/common"
	"github.com/marinesovitch/ote/test-suite/util/k8s"
	"github.com/marinesovitch/ote/test-suite/util/mysql"
	"github.com/marinesovitch/ote/test-suite/util/oci"
	"github.com/marinesovitch/ote/test-suite/util/suite"

	corev1 "k8s.io/api/core/v1"
)

// Create cluster and initialize from a shell dump stored in an OCI bucket.

var unit_fdo *suite.Unit

const ClusterName = "mycluster"
const NewClusterName = "newcluster"
const DumpName = "cluster-from-dump-test-oci1"
const BackupProfileName = "fulldump-oci"
const OciStoragePrefix = "/e2etest/ote-mysql"
const OciCredentialsBackup = "backup-apikey"
const OciCredentialsRestore = "restore-apikey"

type GenerateDumpOCIData struct {
	BackupProfileName     string
	DumpName              string
	BucketName            string
	OciStoragePrefix      string
	OciCredentialsBackup  string
	OciCredentialsRestore string
}

var originalTables *mysql.Records
var generateData GenerateDumpOCIData
var ociStorageOutput string

func BeforeFromDumpOCI(t *testing.T) {
	err := unit_fdo.Client.CreateUserSecrets(
		unit_fdo.Namespace, "mypwds", common.RootUser, common.DefaultHost, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}

	// create a secret with the api key to access the bucket, which should be
	// stored in the path given in the environment variable
	configPath := unit_fdo.Cfg.Oci.ConfigPath
	err = unit_fdo.Client.CreateApikeySecret(unit_fdo.Namespace, OciCredentialsBackup, configPath, common.OciProfileBackup)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_fdo.Client.CreateApikeySecret(unit_fdo.Namespace, OciCredentialsRestore, configPath, common.OciProfileRestore)
	if err != nil {
		t.Fatal(err)
	}

	generateData = GenerateDumpOCIData{
		BackupProfileName:     BackupProfileName,
		DumpName:              DumpName,
		BucketName:            unit_fdo.Cfg.Oci.BucketName,
		OciStoragePrefix:      OciStoragePrefix,
		OciCredentialsBackup:  OciCredentialsBackup,
		OciCredentialsRestore: OciCredentialsRestore,
	}

	err = unit_fdo.GenerateAndApply("cluster-oci-dump.yaml", generateData)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_fdo.WaitOnPod("mycluster-0", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              ClusterName,
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 1,
	}
	err = unit_fdo.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	err = suite.LoadSakilaScript(unit_fdo, "mycluster-0", k8s.Mysql)
	if err != nil {
		t.Fatal(err)
	}

	podSession, err := mysql.NewSession(unit_fdo.Namespace, "mycluster-0", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}

	originalTables, err = podSession.FetchAll("show tables in sakila")
	if err != nil {
		t.Fatal(err)
	}

	// create a dump in a bucket
	if err := unit_fdo.GenerateAndApply("dump-into-bucket.yaml", generateData); err != nil {
		t.Fatal(err)
	}

	// wait for backup to be done
	var mbk *k8s.MySQLBackup
	checker := func(args ...interface{}) (bool, error) {
		var ok bool
		ok, mbk, err = suite.CheckMySQLBackup(unit_fdo.Client, unit_fdo.Namespace, generateData.DumpName)
		return ok, err
	}
	ok, err := unit_fdo.Wait(checker, 300, 2)
	if err != nil {
		t.Fatal(err)
	}

	if ok && mbk.GetName() == generateData.DumpName {
		specClusterName := mbk.GetString("spec", "clusterName")
		if specClusterName != ClusterName {
			t.Fatalf("expected cluster name is %s but got %s", ClusterName, specClusterName)
		}

		mbkStatus := mbk.GetStatus()
		if mbkStatus != k8s.MBKStatusCompleted {
			t.Fatalf("expected mbk status is %s but got %s", k8s.MBKStatusCompleted, mbkStatus)
		}

		mbkStatusOutput := mbk.GetString("status", "output")
		expectedMbkPrefix := generateData.DumpName + "-"
		if !strings.HasPrefix(mbkStatusOutput, expectedMbkPrefix) {
			t.Fatalf("mbk output name should begin with %s but it doesn't (%s)", expectedMbkPrefix, mbkStatusOutput)
		}

		if len(mbkStatusOutput) > 0 {
			ociStorageOutput = filepath.Join(OciStoragePrefix, mbkStatusOutput)
		}
	}

	// destroy the test cluster
	err = unit_fdo.Client.DeleteInnoDBCluster(unit_fdo.Namespace, ClusterName)
	if err != nil {
		t.Error(err)
	}

	err = unit_fdo.WaitOnPodGone("mycluster-0")
	if err != nil {
		t.Error(err)
	}

	err = unit_fdo.WaitOnInnoDBClusterGone(ClusterName)
	if err != nil {
		t.Error(err)
	}

	// delete pv and pvc related to mycluster
	err = unit_fdo.DeleteAllPersistentVolumeClaims()
	if err != nil {
		t.Error(err)
	}

	err = unit_fdo.Client.DeleteSecret(unit_fdo.Namespace, "mypwds")
	if err != nil {
		t.Error(err)
	}
}

func CreateFromDump(t *testing.T) {
	// Create cluster using a shell dump stored in an OCI bucket.
	err := unit_fdo.Client.CreateUserSecrets(
		unit_fdo.Namespace, "newpwds", common.RootUser, common.DefaultHost, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}

	// create cluster with mostly default configs
	generateData.OciStoragePrefix = ociStorageOutput
	err = unit_fdo.GenerateAndApply("create-from-oci-dump.yaml", generateData)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_fdo.WaitOnPod("newcluster-0", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              NewClusterName,
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 1,
		Timeout:           600,
	}
	err = unit_fdo.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_fdo.WaitOnRouters(NewClusterName, 1); err != nil {
		t.Fatal(err)
	}

	podSession, err := mysql.NewSession(unit_fdo.Namespace, "newcluster-0", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}

	tables, err := podSession.FetchAll("show tables in sakila")
	if err != nil {
		t.Fatal(err)
	}

	originalTableNames := originalTables.ToStringsSlice(0)
	tableNames := tables.ToStringsSlice(0)
	if !auxi.AreStringSlicesEqual(originalTableNames, tableNames) {
		t.Fatalf("expected tables: %v but got: %v", originalTableNames, tableNames)
	}

	// add some data with binlog disabled to allow testing that new
	// members added to this cluster use clone for provisioning
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
		if _, err := podSession.Exec(command); err != nil {
			t.Fatal(err)
		}
	}

	if err := suite.CheckRouterPods(unit_fdo.Client, unit_fdo.Namespace, NewClusterName, 1); err != nil {
		t.Fatal(err)
	}
}

func GrowClusterFromDump(t *testing.T) {
	//         Ensures that a cluster created from a dump can be scaled up properly
	patch := k8s.JsonPatch{
		Operation: k8s.PatchReplace,
		Path:      "/spec/instances",
		Value:     2,
	}
	err := unit_fdo.Client.JSONPatchInnoDBCluster(unit_fdo.Namespace, NewClusterName, patch)
	if err != nil {
		t.Error(err)
	}

	err = unit_fdo.WaitOnPod("newcluster-1", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              NewClusterName,
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 2,
	}
	err = unit_fdo.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	// check that the new instance was provisioned through clone and not incremental
	// podSession, err := mysql.NewSession(unit_fdo.Namespace, "newcluster-1", common.RootUser, common.RootPassword)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// result, err := podSession.FetchAll("select * from unlogged_db.tbl")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// records := result.ToStrings()
	// if len(records) == 0 {
	// 	t.Fatal("cannot get data from unlogged_db.tbl")
	// }
	// resultStr := strings.Join(records[0], "")
	// expectedResultStr := "42"
	// if resultStr != expectedResultStr {
	// 	t.Fatalf("expected records [%s] but got [%s]", expectedResultStr, resultStr)
	// }
}

func DestroyClusterFromDump(t *testing.T) {
	err := unit_fdo.Client.DeleteInnoDBCluster(unit_fdo.Namespace, NewClusterName)
	if err != nil {
		t.Error(err)
	}

	err = unit_fdo.WaitOnPodGone("newcluster-1")
	if err != nil {
		t.Error(err)
	}

	err = unit_fdo.WaitOnPodGone("newcluster-0")
	if err != nil {
		t.Error(err)
	}

	err = unit_fdo.WaitOnInnoDBClusterGone(NewClusterName)
	if err != nil {
		t.Error(err)
	}

	// delete pv and pvc related to newcluster
	err = unit_fdo.DeleteAllPersistentVolumeClaims()
	if err != nil {
		t.Error(err)
	}
}

func CreateFromDumpOptions(t *testing.T) {
	// Create cluster using a shell dump with additional options passed to the
	// load command.
	err := unit_fdo.GenerateAndApply("create-from-oci-options.yaml", generateData)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_fdo.WaitOnPod("newcluster-0", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              NewClusterName,
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 1,
		Timeout:           600,
	}
	err = unit_fdo.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	if err = unit_fdo.WaitOnRouters(NewClusterName, 1); err != nil {
		t.Fatal(err)
	}

	podSession, err := mysql.NewSession(unit_fdo.Namespace, "newcluster-0", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}

	tables, err := podSession.FetchAll("show tables in sakila")
	if err != nil {
		t.Fatal(err)
	}

	originalTableNames := originalTables.ToStringsSlice(0)
	tableNames := tables.ToStringsSlice(0)
	if !auxi.AreStringSlicesEqual(originalTableNames, tableNames) {
		t.Fatalf("expected tables: %v but got: %v", originalTableNames, tableNames)
	}

	if err := suite.CheckRouterPods(unit_fdo.Client, unit_fdo.Namespace, NewClusterName, 1); err != nil {
		t.Fatal(err)
	}
}

func AfterFromDumpOCI(t *testing.T) {
	// newcluster
	err := unit_fdo.Client.DeleteInnoDBCluster(unit_fdo.Namespace, NewClusterName)
	if err != nil {
		t.Error(err)
	}

	err = unit_fdo.WaitOnPodGone("newcluster-0")
	if err != nil {
		t.Error(err)
	}

	err = unit_fdo.WaitOnInnoDBClusterGone(NewClusterName)
	if err != nil {
		t.Error(err)
	}

	err = unit_fdo.Client.DeleteSecret(unit_fdo.Namespace, OciCredentialsRestore)
	if err != nil {
		t.Error(err)
	}

	err = unit_fdo.Client.DeleteSecret(unit_fdo.Namespace, OciCredentialsBackup)
	if err != nil {
		t.Error(err)
	}

	if len(ociStorageOutput) > 0 {
		err = oci.BulkDelete(unit_fdo.Cfg.Oci.ConfigPath, common.OciProfileDelete, unit_fdo.Cfg.Oci.BucketName, ociStorageOutput)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestClusterFromDumpOCI(t *testing.T) {
	const Namespace = "from-dump-oci"
	var err error
	unit_fdo, err = suit.NewUnitSetup(Namespace)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_fdo.Cfg.CheckOCIConfig()
	if err != nil {
		t.Skip(err)
	}

	t.Run("BeforeFromDumpOCI=0", BeforeFromDumpOCI)
	t.Run("CreateFromDump=1", CreateFromDump)
	t.Run("GrowClusterFromDump=2", GrowClusterFromDump)
	t.Run("DestroyClusterFromDump=3", DestroyClusterFromDump)
	t.Run("CreateFromDumpOptions=4", CreateFromDumpOptions)
	t.Run("AfterFromDumpOCI=9", AfterFromDumpOCI)

	err = unit_fdo.Teardown()
	if err != nil {
		t.Error(err)
	}
}
