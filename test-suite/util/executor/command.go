// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package executor

import (
	"errors"

	"github.com/marinesovitch/ote/test-suite/util/common"
	"github.com/marinesovitch/ote/test-suite/util/setup"
)

func RunCommand(cmd common.Command, cfg *setup.Configuration) error {
	switch cmd {
	case common.Start:
		return start(cfg)
	case common.Stop:
		return stop(cfg)
	case common.Deploy:
		return deploy(cfg)
	default:
		return errors.New("internal error: unknown command")
	}
}
