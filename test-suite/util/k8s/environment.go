// Generic OTE
// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package k8s

import (
	"errors"
	"strings"

	"github.com/marinesovitch/ote/test-suite/util/common"
	"github.com/marinesovitch/ote/test-suite/util/setup"
)

type K8sEnvironment interface {
	StartCluster() error
	StopCluster() error
	DeleteCluster() error
}

func GetEnvironment(cfg *setup.Configuration) (K8sEnvironment, error) {
	switch strings.ToLower(cfg.K8s.Environment) {
	case common.EnvMinikube:
		return NewMinikubeEnv(cfg)
	case common.EnvK3d:
		return NewK3dEnv(cfg)
	default:
		return nil, errors.New("unknown k8s environment " + cfg.K8s.Environment)
	}
}
