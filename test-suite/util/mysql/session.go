// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package mysql

import (
	"database/sql"
	"fmt"
	"log"
	"os/exec"
	"syscall"
	"time"

	"github.com/marinesovitch/ote/test-suite/util/k8s"

	_ "github.com/go-sql-driver/mysql"
)

type PodSession struct {
	Database  *sql.DB
	portFwCmd *exec.Cmd
}

func trySetupNewSession(namespace string, podName string, user string, password string) (*PodSession, error) {
	kubectl := k8s.Kubectl{}
	const DefaultPort = 3306
	portFwCmd, port, err := kubectl.PortForward(namespace, podName, DefaultPort)
	if err != nil {
		return nil, err
	}
	const DriverName = "mysql"
	const DefaultScheme = "mysql"
	dataSourceName := fmt.Sprintf("%s:%s@tcp(127.0.0.1:%d)/%s", user, password, port, DefaultScheme)
	db, err := sql.Open(DriverName, dataSourceName)
	if err != nil {
		portFwCmd.Process.Signal(syscall.SIGTERM)
		return nil, err
	}
	session := PodSession{
		Database:  db,
		portFwCmd: portFwCmd,
	}
	return &session, nil
}

func NewSession(namespace string, podName string, user string, password string) (session *PodSession, err error) {
	const MaxTrials = 5
	for i := 0; i < MaxTrials; i++ {
		session, err = trySetupNewSession(namespace, podName, user, password)
		if err == nil {
			break
		}
		err = fmt.Errorf("cannot setup a new session on %s/%s for %s@%s: %v", namespace, podName, user, password, err)
		log.Print(err)
		time.Sleep(2 * time.Second)
	}
	return session, err
}

func (p *PodSession) Close() {
	if p.Database != nil {
		p.Database.Close()
	}
	if p.portFwCmd != nil {
		p.portFwCmd.Process.Signal(syscall.SIGTERM)
	}
}

func (p *PodSession) Exec(statement string, args ...interface{}) (sql.Result, error) {
	return p.Database.Exec(statement, args...)
}

func (p *PodSession) QueryOne(query string, args ...interface{}) *sql.Row {
	return p.Database.QueryRow(query, args...)
}

func (p *PodSession) QueryAll(query string, args ...interface{}) (*sql.Rows, error) {
	return p.Database.Query(query, args...)
}

func (p *PodSession) FetchOne(query string, args ...interface{}) (*Record, error) {
	records, err := p.FetchAll(query, args...)
	if err != nil {
		return nil, err
	}

	if len(records.Rows) == 0 {
		return nil, sql.ErrNoRows
	}

	record := Record{
		Columns: records.Columns,
		Row:     records.Rows[0],
	}

	return &record, nil
}

func (p *PodSession) FetchAll(query string, args ...interface{}) (*Records, error) {
	rows, err := p.Database.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columnsCount := len(columns)
	var rawRows RawRows
	for rows.Next() {
		vals := make([]interface{}, columnsCount)
		for i := 0; i < columnsCount; i++ {
			vals[i] = new(string)
		}
		err := rows.Scan(vals...)
		if err != nil {
			return nil, err
		}

		rawRows = append(rawRows, vals)
	}

	records := Records{
		Columns: columns,
		Rows:    rawRows,
	}
	return &records, nil
}
