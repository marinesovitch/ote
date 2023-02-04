// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package badspec_test

// specification errors verified during admission

import (
	"strings"
	"testing"

	"github.com/marinesovitch/ote/test-suite/util/k8s"
	"github.com/marinesovitch/ote/test-suite/util/suite"
)

var unit_ac *suite.Unit

func assertApplyFails(t *testing.T, yamlFilename string, expectedErrMsg string) {
	msg, err := unit_ac.ApplyGetOutput(yamlFilename)
	if !strings.Contains(msg, expectedErrMsg) {
		t.Log(err)
		t.Fatalf("got: '%s', it doesn't contain: '%s'", msg, expectedErrMsg)
	}
}

func InvalidField(t *testing.T) {
	assertApplyFails(t,
		"invalid-field.yaml",
		"ValidationError(InnoDBCluster.spec): unknown field \"bogus\" in com.oracle.mysql.v2.InnoDBCluster.spec")
}

func NameTooLong(t *testing.T) {
	// a cluster name cannot be longer than allowed in innodb cluster (by default 40 chars)
	assertApplyFails(t,
		"name-too-long.yaml",
		"The InnoDBCluster \"veryveryveryveryveryveryveryverylongnamex\" is invalid: "+
			"metadata.name: Invalid value: \"veryveryveryveryveryveryveryverylongnamex\": "+
			"metadata.name in body should be at most 40 chars long")
	// in newer k8 versions:
	// "The InnoDBCluster \"veryveryveryveryveryveryveryverylongnamex\" is invalid: "+
	// 	"metadata.name: Too long: may not be longer than 40")
}

func LackOfName(t *testing.T) {
	// a metadata.name is mandatory (blocked even before the schema validation)
	assertApplyFails(t, "lack-of-name.yaml", "resource name may not be empty")
}

func LackOfSpec(t *testing.T) {
	assertApplyFails(t,
		"lack-of-spec.yaml",
		"ValidationError(InnoDBCluster): missing required field \"spec\" in com.oracle.mysql.v2.InnoDBCluste")
}

func LackOfSecret(t *testing.T) {
	// a spec.secretName is obligatory
	assertApplyFails(t,
		"lack-of-secret.yaml",
		"error validating data: ValidationError(InnoDBCluster.spec): missing required field \"secretName\"")
}

func WrongInstances(t *testing.T) {
	// check invalid values for spec.instances (too small, too big, not a number)
	assertApplyFails(t,
		"zero-instances.yaml",
		"spec.instances: Invalid value: 0: spec.instances in body should be greater than or equal to 1")

	assertApplyFails(t,
		"too-many-instances.yaml",
		"spec.instances: Invalid value: 14: spec.instances in body should be less than or equal to 9")

	assertApplyFails(t,
		"wrong-instances.yaml",
		"ValidationError(InnoDBCluster.spec.instances): invalid type for com.oracle.mysql.v2.InnoDBCluster.spec.instances: got \"string\", expected \"integer\"")
}

func WrongMycnf(t *testing.T) {
	assertApplyFails(t,
		"wrong-mycnf.yaml", "spec.mycnf: Invalid value: \"integer\": spec.mycnf in body must be of type string: \"integer\"")
}

func admissionChecksTeardown(t *testing.T) {
	// none of the tests should create anything
	ics, err := unit_ac.Client.ListInnoDBClusters(unit_ac.Namespace)
	if len(ics.Items) > 0 {
		t.Errorf("unexpected %d ic cluster(s) found", len(ics.Items))
	}
	if err != nil {
		t.Error(err)
	}

	sts, err := unit_ac.Client.ListStatefulSets(unit_ac.Namespace)
	if len(ics.Items) > 0 {
		t.Errorf("unexpected %d statefulset(s) found", len(sts.Items))
	}
	if err != nil {
		t.Error(err)
	}

	pods, err := unit_ac.Client.ListPods(unit_ac.Namespace)
	if len(pods.Items) > 0 {
		t.Errorf("unexpected pod(s) found %v", k8s.GetPodNames(pods))
	}
	if err != nil {
		t.Error(err)
	}
}

func TestClusterSpecAdmissionChecks(t *testing.T) {
	const Namespace = "badspec-admissions"
	var err error
	unit_ac, err = suit.NewUnitSetup(Namespace)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("InvalidField=1", InvalidField)
	t.Run("NameTooLong=1", NameTooLong)
	t.Run("LackOfName=1", LackOfName)
	t.Run("LackOfSpec=1", LackOfSpec)
	t.Run("LackOfSecret=1", LackOfSpec)
	t.Run("WrongInstances=1", LackOfSpec)
	t.Run("WrongMycnf=1", LackOfSpec)

	admissionChecksTeardown(t)

	err = unit_ac.Teardown()
	if err != nil {
		t.Error(err)
	}
}
