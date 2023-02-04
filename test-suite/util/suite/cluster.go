// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package suite

import (
	"errors"

	"github.com/marinesovitch/ote/test-suite/util/mysql"
)

func CrossSyncGtids(namespace string, pods []string, user string, password string) error {
	var sessions []*mysql.PodSession

	defer func() {
		for _, s := range sessions {
			s.Close()
		}
	}()

	for _, pod := range pods {
		session, err := mysql.NewSession(namespace, pod, user, password)
		if err != nil {
			return err
		}
		sessions = append(sessions, session)
	}

	session0 := sessions[0]

	for _, session := range sessions[1:] {
		var gtidSet string
		row := session.QueryOne("select @@gtid_executed")
		if err := row.Scan(&gtidSet); err != nil {
			return err
		}

		var gtidSet0 string
		row = session0.QueryOne("select WAIT_FOR_EXECUTED_GTID_SET(?, 0)", gtidSet)
		if err := row.Scan(&gtidSet0); err != nil {
			return err
		}

		if gtidSet0 != "0" {
			return errors.New("wrong gtid_set")
		}
	}

	for _, session := range sessions[1:] {
		var gtidSet string
		row := session0.QueryOne("select @@gtid_executed")
		if err := row.Scan(&gtidSet); err != nil {
			return err
		}

		var gtidSet0 string
		row = session.QueryOne("select WAIT_FOR_EXECUTED_GTID_SET(?, 1)", gtidSet)
		if err := row.Scan(&gtidSet0); err != nil {
			return err
		}

		if gtidSet0 != "0" {
			return errors.New("wrong gtid_set")
		}
	}

	return nil
}
