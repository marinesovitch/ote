// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package suite

import (
	"errors"
	"fmt"
	"os"

	"github.com/marinesovitch/ote/test-suite/util/auxi"
	"github.com/marinesovitch/ote/test-suite/util/common"
	"github.com/marinesovitch/ote/test-suite/util/k8s"
	"github.com/marinesovitch/ote/test-suite/util/mysql"
)

func hasFinalizers(client *k8s.Client, namespace string, resource k8s.Kind, name string) (bool, error) {
	switch resource {
	case k8s.CRDInnoDBCluster:
		ic, err := client.GetInnoDBCluster(namespace, name)
		if err != nil {
			return false, err
		}
		return len(ic.GetFinalizers()) > 0, nil
	case k8s.Pod:
		pod, err := client.GetPod(namespace, name)
		if err != nil {
			return false, err
		}
		return len(pod.GetFinalizers()) > 0, nil
	default:
		return false, errors.New("unsupported kind")
	}
}

func stripFinalizers(client *k8s.Client, namespace string, resource k8s.Kind, name string) error {
	hasFinalizers, err := hasFinalizers(client, namespace, resource, name)
	if err != nil {
		return err
	}
	if !hasFinalizers {
		return nil
	}

	const FinalizersPath = "/metadata/finalizers"
	patch := k8s.JsonPatch{
		Operation: k8s.PatchRemove,
		Path:      FinalizersPath,
	}
	switch resource {
	case k8s.CRDInnoDBCluster:
		return client.JSONPatchInnoDBCluster(namespace, name, patch)
	case k8s.Pod:
		return client.PatchPod(namespace, name, patch)
	default:
		return errors.New("unsupported resource kind " + resource.String())
	}
}

func LoadSakilaScript(unit *Unit, podName string, containerId k8s.ContainerId) error {
	sakilaSchemaSQLPath := unit.Cfg.GetTestDataPath("sql/sakila-schema.sql")
	sakilaSchema, err := os.ReadFile(sakilaSchemaSQLPath)
	if err != nil {
		return err
	}

	sakilaDataSQLPath := unit.Cfg.GetTestDataPath("sql/sakila-data.sql")
	sakilaData, err := os.ReadFile(sakilaDataSQLPath)
	if err != nil {
		return err
	}

	script := sakilaSchema
	script = append(script, sakilaData...)
	return mysql.LoadScript(unit.Namespace, podName, containerId, string(script))
}

func CheckMySQLBackup(client *k8s.Client, namespace string, name string) (bool, *k8s.MySQLBackup, error) {
	mbk, err := client.GetMySQLBackup(namespace, name)
	if err != nil {
		return false, nil, err
	}

	status, found, err := mbk.AskStatus()
	if err != nil {
		return false, nil, err
	}
	return found && status == k8s.MBKStatusCompleted, mbk, nil
}

func QuerySet(namespace string, podName string, user string, password string, query string, column int) (common.StringSet, error) {
	session, err := mysql.NewSession(namespace, podName, user, password)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	records, err := session.FetchAll(query)
	if err != nil {
		return nil, err
	}
	return records.ToStringSet(0), nil
}

func PrepareAccountSet(accounts []string, addDefaultAccounts bool) common.StringSet {
	accountSet := auxi.SliceToSet(accounts)
	if addDefaultAccounts {
		auxi.AddToSet(common.DefaultMysqlAccounts, accountSet)
	}
	return accountSet
}

func GetStringFromJSONTree(jsonTree map[string]interface{}, keyPath ...string) (string, error) {
	if len(keyPath) == 0 {
		return "", errors.New("internal error, no key found")
	}

	if len(keyPath) == 1 {
		switch v := jsonTree[keyPath[0]].(type) {
		case string:
			return v, nil
		default:
			return "", fmt.Errorf("cannot get string value from %q for key %q", jsonTree, keyPath[0])
		}
	}

	switch v := jsonTree[keyPath[0]].(type) {
	case map[string]interface{}:
		return GetStringFromJSONTree(v, keyPath[1:]...)
	default:
		return "", fmt.Errorf("expected json tree type is %T but got %T", jsonTree, v)
	}
}
