// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package suite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/marinesovitch/ote/test-suite/util/auxi"
	"github.com/marinesovitch/ote/test-suite/util/common"
	"github.com/marinesovitch/ote/test-suite/util/k8s"
	"github.com/marinesovitch/ote/test-suite/util/mysql"

	corev1 "k8s.io/api/core/v1"
)

func extractIndex(host string) (int, error) {
	hostPieces := strings.Split(host, ".")
	if len(hostPieces) == 0 {
		return -1, fmt.Errorf("incorrect host name %s", host)
	}
	prefix := hostPieces[0]
	prefixPieces := strings.Split(prefix, "-")
	if len(prefixPieces) == 0 {
		return -1, fmt.Errorf("incorrect host name prefix %s", prefix)
	}
	indexStr := prefixPieces[len(prefixPieces)-1]
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return -1, err
	}
	return index, nil
}

func CheckGroup(icobj *k8s.InnoDBCluster, allPods []*corev1.Pod, user string, password string) (map[string]int, error) {
	info := make(map[string]int)

	session, err := mysql.NewSession(icobj.GetNamespace(), icobj.GetName()+"-0", user, password)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	members, err := session.QueryAll(
		"SELECT member_id, member_host, member_port, member_state, member_role FROM performance_schema.replication_group_members ORDER BY member_host")
	if err != nil {
		return nil, err
	}
	defer members.Close()

	var primaries []string
	var mid, mhost, mport, mstate, mrole string
	var memberCounter int
	for members.Next() {
		err := members.Scan(&mid, &mhost, &mport, &mstate, &mrole)
		if err != nil {
			return nil, err
		}
		if mstate != "ONLINE" {
			return nil, fmt.Errorf("member %s:%s (%s) is not online", mhost, mport, mid)
		}
		if mrole == "PRIMARY" {
			primaries = append(primaries, mrole)
			index, err := extractIndex(mhost)
			if err != nil {
				return nil, fmt.Errorf("cannot extract an index from the host name %s", mhost)
			}
			info["primary"] = index
		}
		memberCounter++
	}
	err = members.Err()
	if err != nil {
		return nil, err
	}

	if len(primaries) != 1 {
		return nil, fmt.Errorf("there is expected one primary but got %d", len(primaries))
	}

	instancesCount := icobj.GetInt("spec", "instances")
	if memberCounter != instancesCount {
		return nil, fmt.Errorf("there are expected %d member(s) but got %d", instancesCount, memberCounter)
	}

	onlineInstancesCount := icobj.GetInt("status", "cluster", "onlineInstances")
	if memberCounter != onlineInstancesCount {
		return nil, fmt.Errorf("there are expected %d online member(s) but got %d", onlineInstancesCount, memberCounter)
	}

	return info, nil
}

func checkGroupReplicationVar(grvars common.StringToStringMap, grvar string, expectedValue string) error {
	value, ok := grvars[grvar]

	if !ok {
		return fmt.Errorf("no value for %s but expected is %s", grvar, expectedValue)
	}

	if value != expectedValue {
		return fmt.Errorf("expected value for %s is %s but got %s", grvar, expectedValue, value)
	}

	return nil
}

func checkGroupReplicationGroupSeeds(allPods []*corev1.Pod, grvars common.StringToStringMap, groupSeedset common.StringSet) error {
	GroupReplicationGroupSeeds := "group_replication_group_seeds"
	if len(allPods) == 1 {
		return checkGroupReplicationVar(grvars, GroupReplicationGroupSeeds, "")
	}

	grGroupSeedsVar, ok := grvars[GroupReplicationGroupSeeds]
	if !ok {
		return fmt.Errorf("%s not found", GroupReplicationGroupSeeds)
	}
	grGroupSeeds := strings.Split(grGroupSeedsVar, ",")

	groupSeeds := groupSeedset.ToSortedSlice()
	if !auxi.AreStringSlicesEqual(groupSeeds, grGroupSeeds) {
		return fmt.Errorf("%s should be %v but got %v", GroupReplicationGroupSeeds, groupSeeds, grGroupSeeds)
	}

	return nil
}

func checkGroupReplicationIpWhitelist(grvars common.StringToStringMap, groupSeedset common.StringSet) error {
	// NIY
	// GroupReplicationIpWhitelist := "group_replication_ip_whitelist"
	// grIpWhitelistVar, ok := grvars[GroupReplicationIpWhitelist]
	// if !ok {
	// 	return fmt.Errorf("%s not found", GroupReplicationIpWhitelist)
	// }
	// grIpWhitelist := strings.Split(grIpWhitelistVar, ",")

	// groupSeeds := groupSeedset.ToSortedSlice()

	// if !auxi.AreStringSlicesEqual(groupSeeds, grIpWhitelist) {
	// 	return fmt.Errorf("%s should be %q but got %q", GroupReplicationIpWhitelist, groupSeeds, grIpWhitelist)
	// }

	return nil
}

func checkProcessList(session *mysql.PodSession, numSessions int) error {
	if numSessions == NoNumSessions {
		return nil
	}

	rows, err := session.QueryAll("show processlist")
	if err != nil {
		return err
	}

	ignoreUsers := []string{"event_scheduler", "system user"}
	ignoreCommands := []string{"Binlog Dump GTID"}

	var id, user, command string
	var host, db, time, state, info interface{}
	var countedSessions int
	for rows.Next() {
		err := rows.Scan(&id, &user, &host, &db, &command, &time, &state, &info)
		if err != nil {
			return err
		}

		if auxi.Contains(ignoreUsers, user) {
			continue
		}

		if auxi.Contains(ignoreCommands, command) {
			continue
		}

		countedSessions++
	}
	err = rows.Err()
	if err != nil {
		return err
	}

	expectedNumSessions := numSessions + 1
	if countedSessions != expectedNumSessions {
		return fmt.Errorf("expected number of sessions is %d but got %d", expectedNumSessions, countedSessions)
	}

	return nil
}

func checkInstance(icobj *k8s.InnoDBCluster, allPods []*corev1.Pod, pod *corev1.Pod, isPrimary bool, numSessions int, version string, user string, password string) error {
	groupSeeds := make(common.StringSet)
	for _, p := range allPods {
		if p != pod {
			groupSeeds[p.GetName()+"."+icobj.GetName()+
				"-instances."+icobj.GetNamespace()+".svc.cluster.local:3306"] = struct{}{}
		}
	}

	name := pod.GetName()
	const DefaultBaseServerId = 1000
	baseId := icobj.GetOptionalInt(DefaultBaseServerId, "spec", "baseServerId")

	session, err := mysql.NewSession(pod.GetNamespace(), pod.GetName(), user, password)
	if err != nil {
		return err
	}

	// check that the Pod info matches
	var server_id int
	var server_uuid, report_host, ver string
	var sro bool
	row := session.QueryOne(
		"select @@server_id, @@server_uuid, @@report_host, @@super_read_only, @@version")
	err = row.Scan(&server_id, &server_uuid, &report_host, &sro, &ver)
	if err != nil {
		return err
	}
	if isPrimary {
		if sro {
			return fmt.Errorf("unexpected sro (%t) for primary %s", sro, name)
		}
	} else {
		if !sro {
			return fmt.Errorf("unexpected sro (%t) for secondary %s", sro, name)
		}
	}

	// membership-info can be missing if the instance is deleted
	// we check elsewhere if it's supposed to be there, so it's safe to ignore it here
	if minfoJson, ok := pod.GetAnnotations()["mysql.oracle.com/membership-info"]; ok {
		var minfo map[string]interface{}
		if err := json.Unmarshal([]byte(minfoJson), &minfo); err != nil {
			return err
		}

		memberId := minfo["memberId"]
		if memberId != server_uuid {
			return fmt.Errorf("memberId (%s) is different than expected (%s)", memberId, server_uuid)
		}
		splitName := strings.Split(pod.GetName(), "-")
		instanceIndex, err := strconv.Atoi(splitName[len(splitName)-1])
		if err != nil {
			return err
		}
		if baseId+instanceIndex != server_id {
			return fmt.Errorf("mismatch in indexing baseId+instanceIndex (%d) is different than expected (%d)", baseId+instanceIndex, server_id)
		}
	}

	if len(version) > 0 {
		if strings.Contains(version, "-") {
			extractedVersion := strings.Split(ver, "-")[0]
			if ver != extractedVersion {
				return fmt.Errorf("incorrect version %s, expected %s", extractedVersion, ver)
			}
		} else {
			if ver != version {
				return fmt.Errorf("incorrect version %s, expected %s", version, ver)
			}
		}
	}

	// check that the GR config is as expected
	rows, err := session.QueryAll("show global variables like 'group_replication%'")
	if err != nil {
		return err
	}
	var id string
	for rows.Next() {
		err := rows.Scan(&id, &name)
		if err != nil {
			return err
		}
		// log.Info.Print(id, name)
	}
	err = rows.Err()
	if err != nil {
		return err
	}

	grvarRows, err := session.FetchAll("show global variables like 'group_replication%'")
	if err != nil {
		return err
	}

	grvars := grvarRows.ToStringToStringMap(0, 1)

	if err := checkGroupReplicationVar(grvars, "group_replication_start_on_boot", "OFF"); err != nil {
		return err
	}

	if err := checkGroupReplicationVar(grvars, "group_replication_single_primary_mode", "ON"); err != nil {
		return err
	}

	if err := checkGroupReplicationVar(grvars, "group_replication_bootstrap_group", "OFF"); err != nil {
		return err
	}

	if err := checkGroupReplicationVar(grvars, "group_replication_ssl_mode", "REQUIRED"); err != nil {
		return err
	}

	if err := checkGroupReplicationGroupSeeds(allPods, grvars, groupSeeds); err != nil {
		return err
	}

	if len(allPods) > 1 {
		if err := checkGroupReplicationIpWhitelist(grvars, groupSeeds); err != nil {
			return err
		}
	}

	// check that SSL is enabled for recovery
	row = session.QueryOne(
		"select ssl_allowed, coalesce(tls_version, '') from performance_schema.replication_connection_configuration where channel_name='group_replication_recovery'")
	// there's no recovery channel in the seed
	if row != nil {
		var ssl_allowed string
		var tls_version string
		err = row.Scan(&ssl_allowed, &tls_version)
		if err != nil {
			if err != sql.ErrNoRows {
				return err
			}
		} else {
			if ssl_allowed != "YES" {
				return fmt.Errorf("ssl is not allowed for pod %s", name)
			}
			if len(tls_version) == 0 {
				return fmt.Errorf("tls version is empty for pod %s", name)
			}
		}
	}

	if err := checkProcessList(session, numSessions); err != nil {
		return err
	}

	return nil
}

type TableInfo struct {
	rowsCount int
	checksum  string
}

type TablesInfo map[string]TableInfo

func schemaReport(session *mysql.PodSession, schema string) (TablesInfo, error) {
	tablesInfo := make(TablesInfo)
	tables, err := session.FetchAll("SHOW TABLES IN !", schema)
	if err != nil {
		return nil, err
	}
	tableNames := tables.ToStringsSlice(0)
	for _, table := range tableNames {
		var tableInfo TableInfo
		err = session.QueryOne("SELECT count(*) FROM !.!", schema, table).Scan(&tableInfo.rowsCount)
		if err != nil {
			return nil, err
		}

		err = session.QueryOne("CHECKSUM TABLE !.!", schema, table).Scan(&tableInfo.checksum)
		if err != nil {
			return nil, err
		}

		tablesInfo[table] = tableInfo
	}
	return tablesInfo, nil
}

func CheckData(allPods []*corev1.Pod, user string, password string, primary int) error {

	ignoreSchemas := []string{"mysql", "information_schema", "performance_schema", "sys"}

	primaryName := allPods[primary].GetName()
	primaryIndex := strconv.Itoa(primary)
	if !strings.HasSuffix(primaryName, primaryIndex) {
		return fmt.Errorf("primary pod name is %s but expected index is %s", primaryName, primaryIndex)
	}

	primaryPodSession, err := mysql.NewSession(
		allPods[primary].GetNamespace(), allPods[primary].GetName(),
		user, password)
	if err != nil {
		return err
	}
	defer primaryPodSession.Close()

	var gtidSet0 string
	if err := primaryPodSession.QueryOne("SELECT @@gtid_executed").Scan(&gtidSet0); err != nil {
		return err
	}

	primaryPodSchemas, err := primaryPodSession.FetchAll("SHOW SCHEMAS")
	if err != nil {
		return err
	}

	primaryPodSchemaNames := primaryPodSchemas.ToStringsSlice(0)

	schemaTableInfo0 := make(map[string]TablesInfo)
	for _, schema := range primaryPodSchemaNames {
		if !auxi.Contains(ignoreSchemas, schema) {
			schemaTablesInfo, err := schemaReport(primaryPodSession, schema)
			if err != nil {
				return nil
			}
			schemaTableInfo0[schema] = schemaTablesInfo
		}
	}

	for i, pod := range allPods {
		if i == primary {
			continue
		}
		podSession, err := mysql.NewSession(
			pod.GetNamespace(), pod.GetName(),
			user, password)

		if err != nil {
			return err
		}
		defer podSession.Close()

		waitForExecutedGtidSet, err := podSession.Exec("select WAIT_FOR_EXECUTED_GTID_SET(?, 1)", gtidSet0)
		if err != nil {
			return err
		}
		executedGtidSetRowsAffected, err := waitForExecutedGtidSet.RowsAffected()
		if err != nil {
			return nil
		}
		if executedGtidSetRowsAffected != 0 {
			return fmt.Errorf("WAIT_FOR_EXECUTED_GTID_SET for %s is %d but should be 0", gtidSet0, executedGtidSetRowsAffected)
		}

		// check for missing GTIDs
		var missingGtids string
		if err := podSession.QueryOne(fmt.Sprintf("select gtid_subtract(%s, @@gtid_executed)", gtidSet0)).Scan(&missingGtids); err != nil {
			return err
		}
		if len(missingGtids) > 0 {
			return fmt.Errorf("none missing gtids expected but found %s", missingGtids)
		}

		// check for errant GTIDs comparing against a more recent GTID set from primary
		var gtidSet string
		if err := podSession.QueryOne("SELECT @@gtid_executed").Scan(&gtidSet); err != nil {
			return err
		}

		var errants string
		if err := primaryPodSession.QueryOne(fmt.Sprintf("select gtid_subtract(%s, @@gtid_executed)", gtidSet0)).Scan(&errants); err != nil {
			return err
		}
		if len(errants) > 0 {
			return fmt.Errorf("none errants expected but found %s", errants)
		}

		podSchemas, err := podSession.FetchAll("SHOW SCHEMAS")
		if err != nil {
			return nil
		}
		podSchemaNames := podSchemas.ToStringsSlice(0)
		if auxi.AreStringSlicesEqual(podSchemaNames, primaryPodSchemaNames) {
			return fmt.Errorf("primary pod schemas (%v) are different than pod %s schemas (%v)", primaryPodSchemaNames, pod.GetName(), podSchemaNames)
		}

		for _, schema := range primaryPodSchemaNames {
			if auxi.Contains(ignoreSchemas, schema) {
				continue
			}

			schemaTablesInfo, err := schemaReport(podSession, schema)
			if err != nil {
				return err
			}

			expectedSchemaTablesInfo := schemaTableInfo0[schema]
			if !reflect.DeepEqual(expectedSchemaTablesInfo, schemaTablesInfo) {
				return fmt.Errorf("expected schema tables info is %v but got %v", expectedSchemaTablesInfo, schemaTablesInfo)
			}
		}
	}
	return nil
}
