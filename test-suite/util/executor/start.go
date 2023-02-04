// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package executor

import (
	"github.com/marinesovitch/ote/test-suite/util/k8s"
	"github.com/marinesovitch/ote/test-suite/util/setup"
)

func start(cfg *setup.Configuration) error {
	env, err := k8s.GetEnvironment(cfg)
	if err != nil {
		return err
	}

	if cfg.K8s.DeleteAtStart {
		err = env.DeleteCluster()
		if err != nil {
			return err
		}
	}

	err = env.StartCluster()
	if err != nil {
		return err
	}

	return deploy(cfg)
}
