// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package backup_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/marinesovitch/ote/test-suite/util/common"
	"github.com/marinesovitch/ote/test-suite/util/k8s"
	"github.com/marinesovitch/ote/test-suite/util/mysql"
	"github.com/marinesovitch/ote/test-suite/util/oci"
	"github.com/marinesovitch/ote/test-suite/util/suite"

	corev1 "k8s.io/api/core/v1"
)

var unit_dmp *suite.Unit

const ClusterName = "mycluster"
const BackupProfileNameVolume = "fulldump-vol"
const BackupProfileNameOci = "fulldump-oci"
const TestBackupVolumeName = "test-backup-storage"
const MbkToVolume = "dump-test-volume1"
const MbkToOci = "dump-test-oci1"
const OciStoragePrefix = "/e2etest/ote-mysql"
const OciCredentials = "backup-apikey"

var ociBucket string
var ociStorageOutput string

type GenerateDumpClusterData struct {
	BackupProfileNameVolume string
	BackupProfileNameOci    string
	BackupVolumeName        string
	BucketName              string
	BackupDir               string
	OciStoragePrefix        string
	OciCredentials          string
}

func Create(t *testing.T) {
	err := unit_dmp.Client.CreateUserSecrets(unit_dmp.Namespace, "mypwds", common.RootUser, common.DefaultHost, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}

	ociBucket = unit_dmp.Cfg.Oci.BucketName
	if ociBucket == "" {
		const UnsetBucket = "not-set"
		ociBucket = UnsetBucket
	}

	const DumpClusterTemplate = "dump-cluster.yaml"
	generateData := GenerateDumpClusterData{
		BackupProfileNameVolume: BackupProfileNameVolume,
		BackupProfileNameOci:    BackupProfileNameOci,
		BackupVolumeName:        TestBackupVolumeName,
		BucketName:              ociBucket,
		BackupDir:               "/tmp/backups",
		OciStoragePrefix:        OciStoragePrefix,
		OciCredentials:          OciCredentials,
	}

	err = unit_dmp.GenerateAndApply(DumpClusterTemplate, generateData)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_dmp.WaitOnPod("mycluster-0", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	err = unit_dmp.WaitOnPod("mycluster-1", corev1.PodRunning)
	if err != nil {
		t.Fatal(err)
	}

	waitParams := k8s.WaitOnInnoDBClusterParams{
		Name:              ClusterName,
		ExpectedStatus:    []string{"ONLINE"},
		ExpectedNumOnline: 2,
	}
	err = unit_dmp.WaitOnInnoDBCluster(waitParams)
	if err != nil {
		t.Fatal(err)
	}

	err = suite.LoadSakilaScript(unit_dmp, "mycluster-0", k8s.Mysql)
	if err != nil {
		t.Fatal(err)
	}

	podSession, err := mysql.NewSession(unit_dmp.Namespace, "mycluster-0", common.RootUser, common.RootPassword)
	if err != nil {
		t.Fatal(err)
	}

	commands := []string{
		"create schema excludeme",
		"create table excludeme.country like sakila.country",
		"insert into excludeme.country select * from sakila.country",
	}
	for _, command := range commands {
		if _, err := podSession.Exec(command); err != nil {
			t.Fatal(err)
		}
	}

	// create a test volume to store backups
	const DumpVolumeTemplate = "dump-volume.yaml"
	err = unit_dmp.GenerateAndApply(DumpVolumeTemplate, generateData)
	if err != nil {
		t.Fatal(err)
	}
}

type GenerateBackupData struct {
	BackupName        string
	BackupProfileName string
}

func BackupToVolume(t *testing.T) {
	const BackupToVolumeTemplate = "task-backup-to-volume.yaml"
	generateData := GenerateBackupData{
		BackupName:        MbkToVolume,
		BackupProfileName: BackupProfileNameVolume,
	}
	err := unit_dmp.GenerateAndApply(BackupToVolumeTemplate, generateData)
	if err != nil {
		t.Fatal(err)
	}

	var mbk *k8s.MySQLBackup
	checker := func(args ...interface{}) (bool, error) {
		var ok bool
		ok, mbk, err = suite.CheckMySQLBackup(unit_dmp.Client, unit_dmp.Namespace, generateData.BackupName)
		return ok, err
	}
	_, err = unit_dmp.Wait(checker, 300, 2)
	if err != nil {
		t.Fatal(err)
	}

	specClusterName := mbk.GetString("spec", "clusterName")
	if specClusterName != ClusterName {
		t.Fatalf("expected cluster name is %s but got %s", ClusterName, specClusterName)
	}

	mbkStatus := mbk.GetStatus()
	if mbkStatus != k8s.MBKStatusCompleted {
		t.Fatalf("expected mbk status is %s but got %s", k8s.MBKStatusCompleted, mbkStatus)
	}

	mbkStatusOutput := mbk.GetString("status", "output")
	expectedMbkPrefix := generateData.BackupName + "-"
	if !strings.HasPrefix(mbkStatusOutput, expectedMbkPrefix) {
		t.Fatalf("mbk output name should begin with %s but it doesn't (%s)", expectedMbkPrefix, mbkStatusOutput)
	}

	// check status in backup object
	mbk, err = unit_dmp.Client.GetMySQLBackup(unit_dmp.Namespace, generateData.BackupName)
	if err != nil {
		t.Fatal(err)
	}

	if !mbk.HasField("status", "startTime") {
		t.Fatal("mbk hasn't got status.startTime field")
	}
	if !mbk.HasField("status", "completionTime") {
		t.Fatal("mbk hasn't got status.completionTime field")
	}

	mbkCompletionTime := mbk.GetTimeStamp("status", "completionTime")
	mbkStartTime := mbk.GetTimeStamp("status", "startTime")
	if mbkCompletionTime < mbkStartTime {
		t.Fatalf("mbk %s completion time (%d) should be greater than or equal to the start time (%d)", mbk.GetName(), mbkCompletionTime, mbkStartTime)
	}

	if !mbk.HasField("status", "elapsedTime") {
		t.Fatal("mbk hasn't got status.elapsedTime field")
	}

	if !mbk.HasField("status", "spaceAvailable") || mbk.GetString("status", "spaceAvailable") == "" {
		t.Fatalf("mbk %s should have status.spaceAvailable but got '%s'", mbk.GetName(), mbk.GetString("status", "spaceAvailable"))
	}

	if !mbk.HasField("status", "size") || mbk.GetString("status", "size") == "" {
		t.Fatalf("mbk %s should have status.size but got '%s'", mbk.GetName(), mbk.GetString("status", "size"))
	}

	mbkMethod := mbk.GetString("status", "method")
	expectedMbkMethod := "dump-instance/volume"
	if mbkMethod != expectedMbkMethod {
		t.Fatalf("expected mbk method is %s but got %s", expectedMbkMethod, mbkMethod)
	}
}

func BackupToOciBucket(t *testing.T) {
	err := unit_dmp.Cfg.CheckOCIConfig()
	if err != nil {
		t.Skip(err)
	}

	err = unit_dmp.Client.CreateApikeySecret(unit_dmp.Namespace, OciCredentials, unit_dmp.Cfg.Oci.ConfigPath, common.OciProfileBackup)
	if err != nil {
		t.Fatal(err)
	}

	const BackupToVolumeTemplate = "task-backup-to-volume.yaml"
	generateData := GenerateBackupData{
		BackupName:        MbkToOci,
		BackupProfileName: BackupProfileNameOci,
	}
	err = unit_dmp.GenerateAndApply(BackupToVolumeTemplate, generateData)
	if err != nil {
		t.Fatal(err)
	}

	var mbk *k8s.MySQLBackup
	checker := func(args ...interface{}) (bool, error) {
		var ok bool
		ok, mbk, err = suite.CheckMySQLBackup(unit_dmp.Client, unit_dmp.Namespace, generateData.BackupName)
		return ok, err
	}
	_, err = unit_dmp.Wait(checker, 300, 2)
	if err != nil {
		t.Fatal(err)
	}

	specClusterName := mbk.GetString("spec", "clusterName")
	if specClusterName != ClusterName {
		t.Fatalf("expected cluster name is %s but got %s", ClusterName, specClusterName)
	}

	mbkStatus := mbk.GetStatus()
	if mbkStatus != k8s.MBKStatusCompleted {
		t.Fatalf("expected mbk status is %s but got %s", k8s.MBKStatusCompleted, mbkStatus)
	}

	mbkStatusOutput := mbk.GetString("status", "output")
	expectedMbkPrefix := generateData.BackupName + "-"
	if !strings.HasPrefix(mbkStatusOutput, expectedMbkPrefix) {
		t.Fatalf("mbk output name should begin with %s but it doesn't (%s)", expectedMbkPrefix, mbkStatusOutput)
	}

	if len(mbkStatusOutput) > 0 {
		ociStorageOutput = filepath.Join(OciStoragePrefix, mbkStatusOutput)
	}

	// check status in backup object
	mbk, err = unit_dmp.Client.GetMySQLBackup(unit_dmp.Namespace, generateData.BackupName)
	if err != nil {
		t.Fatal(err)
	}

	if !mbk.HasField("status", "startTime") {
		t.Fatal("mbk hasn't got status.startTime field")
	}
	if !mbk.HasField("status", "completionTime") {
		t.Fatal("mbk hasn't got status.completionTime field")
	}
	mbkCompletionTime := mbk.GetTimeStamp("status", "completionTime")
	mbkStartTime := mbk.GetTimeStamp("status", "startTime")
	if mbkCompletionTime < mbkStartTime {
		t.Fatalf("mbk %s completion time (%d) should be greater than or equal to the start time (%d)", mbk.GetName(), mbkCompletionTime, mbkStartTime)
	}

	if !mbk.HasField("status", "elapsedTime") {
		t.Fatal("mbk hasn't got status.elapsedTime field")
	}

	mbkMethod := mbk.GetString("status", "method")
	expectedMbkMethod := "dump-instance/oci-bucket"
	if mbkMethod != expectedMbkMethod {
		t.Fatalf("expected mbk method is %s but got %s", expectedMbkMethod, mbkMethod)
	}

	mbkBucket := mbk.GetString("status", "bucket")
	if mbkBucket != ociBucket {
		t.Fatalf("expected bucket name is %s but got %s", ociBucket, mbkBucket)
	}

	ociTenancy := mbk.GetString("status", "ociTenancy")
	if !strings.Contains(ociTenancy, "oci") || !strings.Contains(ociTenancy, "tenancy") {
		t.Fatalf("oci tenancy %s is incorrect", ociTenancy)
	}

	mbkSource := mbk.GetString("status", "source")
	if len(mbkSource) == 0 {
		t.Fatalf("mbk status.source is empty")
	}
}

func Destroy(t *testing.T) {
	err := unit_dmp.Client.DeleteInnoDBCluster(unit_dmp.Namespace, ClusterName)
	if err != nil {
		t.Error(err)
	}

	err = unit_dmp.WaitOnPodGone("mycluster-2")
	if err != nil {
		t.Error(err)
	}

	err = unit_dmp.WaitOnPodGone("mycluster-1")
	if err != nil {
		t.Error(err)
	}

	err = unit_dmp.WaitOnPodGone("mycluster-0")
	if err != nil {
		t.Error(err)
	}

	err = unit_dmp.WaitOnInnoDBClusterGone(ClusterName)
	if err != nil {
		t.Error(err)
	}

	if err = unit_dmp.Client.DeleteMySQLBackup(unit_dmp.Namespace, MbkToVolume); err != nil {
		t.Error(err)
	}

	ociEnabled := unit_dmp.Cfg.Oci.Enable
	if ociEnabled {
		if err = unit_dmp.Client.DeleteMySQLBackup(unit_dmp.Namespace, MbkToOci); err != nil {
			t.Error(err)
		}
	}

	if err = unit_dmp.Client.DeletePersistentVolumeClaim(unit_dmp.Namespace, TestBackupVolumeName); err != nil {
		t.Error(err)
	}

	if err = unit_dmp.Client.DeletePersistentVolume(unit_dmp.Namespace, TestBackupVolumeName); err != nil {
		t.Error(err)
	}

	if ociEnabled {
		err = unit_dmp.Client.DeleteSecret(unit_dmp.Namespace, OciCredentials)
		if err != nil {
			t.Error(err)
		}
	}

	err = unit_dmp.Client.DeleteSecret(unit_dmp.Namespace, "mypwds")
	if err != nil {
		t.Error(err)
	}

	if ociEnabled && len(ociStorageOutput) > 0 {
		err = oci.BulkDelete(unit_dmp.Cfg.Oci.ConfigPath, common.OciProfileDelete, ociBucket, ociStorageOutput)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestDumpInstance(t *testing.T) {
	const Namespace = "backup"
	var err error
	unit_dmp, err = suit.NewUnitSetup(Namespace)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Create=0", Create)
	t.Run("BackupToVolume=1", BackupToVolume)
	t.Run("BackupToOciBucket=1", BackupToOciBucket)
	t.Run("Destroy=9", Destroy)

	err = unit_dmp.Teardown()
	if err != nil {
		t.Error(err)
	}
}
