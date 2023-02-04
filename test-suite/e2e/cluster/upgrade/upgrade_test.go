// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package upgrade_test

import (
	"os"
	"testing"

	"github.com/marinesovitch/ote/test-suite/util/log"
	"github.com/marinesovitch/ote/test-suite/util/suite"
)

var suit *suite.Suite = nil

func TestMain(m *testing.M) {
	var err error
	suit, err = suite.CreateSuite()
	if err != nil {
		log.Error.Fatalf("suite creation failed: %s", err)
	}
	os.Exit(m.Run())
}
