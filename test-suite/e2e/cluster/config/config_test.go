// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package config_test

import (
	"os"
	"testing"

	"github.com/marinesovitch/ote/test-suite/util/log"
	"github.com/marinesovitch/ote/test-suite/util/suite"
)

var suit *suite.Suite = nil

var oldVersionTag string

func TestMain(m *testing.M) {
	var err error
	suit, err = suite.CreateSuite()
	if err != nil {
		log.Error.Fatalf("suite creation failed: %s", err)
	}
	oldVersionTag = suit.Cfg.Images.MinSupportedMysqlVersion
	os.Exit(m.Run())
}
