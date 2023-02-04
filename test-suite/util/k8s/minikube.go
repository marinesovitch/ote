// Minikube
// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package k8s

import (
	"github.com/marinesovitch/ote/test-suite/util/setup"
	"github.com/marinesovitch/ote/test-suite/util/system"
)

type minikubeEnv struct {
	cfg *setup.Configuration
}

func (m *minikubeEnv) executeCmd(args []string) error {
	return system.Execute(m.cfg.K8s.Environment, args...)
}

func (m *minikubeEnv) prepareRegistryArgs(args []string) ([]string, error) {
	if m.cfg.HasRegistry() {
		registry := m.cfg.GetRegistryUrl()
		if m.cfg.Minikube.RegistryInsecure {
			args = append(args, "--insecure-registry")
		}
		args = append(args, registry)
	}

	return args, nil
}

func (m *minikubeEnv) StartCluster() error {
	args := []string{
		"start",
		"-p",
		m.cfg.K8s.ClusterName,
	}

	args, err := m.prepareRegistryArgs(args)
	if err != nil {
		return err
	}

	return m.executeCmd(args)
}

func (m *minikubeEnv) StopCluster() error {
	args := []string{
		"stop",
		"-p",
		m.cfg.K8s.ClusterName,
	}
	return m.executeCmd(args)
}

func (m *minikubeEnv) DeleteCluster() error {
	args := []string{
		"delete",
		"-p",
		m.cfg.K8s.ClusterName,
	}
	return m.executeCmd(args)
}

func NewMinikubeEnv(cfg *setup.Configuration) (K8sEnvironment, error) {
	return &minikubeEnv{cfg}, nil
}
