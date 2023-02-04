// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package main

import (
	"github.com/marinesovitch/ote/test-suite/util/executor"
	"github.com/marinesovitch/ote/test-suite/util/log"
	"github.com/marinesovitch/ote/test-suite/util/setup"
)

func main() {
	cfg, cmd, err := setup.CreateConfiguration(false)
	if err != nil {
		log.Error.Fatal(err)
	}

	err = executor.RunCommand(cmd, &cfg)
	if err != nil {
		log.Error.Fatal(err)
	}
}
