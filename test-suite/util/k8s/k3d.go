// K3d
// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package k8s

import (
	"github.com/marinesovitch/ote/test-suite/util/container"
	"github.com/marinesovitch/ote/test-suite/util/setup"
	"github.com/marinesovitch/ote/test-suite/util/system"
)

type k3dEnv struct {
	cfg             *setup.Configuration
	containerEngine container.Engine
}

func (k *k3dEnv) executeCmd(args []string) error {
	return system.Execute(k.cfg.K8s.Environment, args...)
}

func (k *k3dEnv) prepareRegistryArgs(args []string) ([]string, error) {
	if k.cfg.HasRegistry() {
		registryConfigPath, err := setup.GenerateK3dRegistryConfig(k.cfg)
		if err != nil {
			return args, err
		}
		args = append(args, "--registry-config", registryConfigPath)
	}

	return args, nil
}

func (k *k3dEnv) connectNetwork() error {
	if !k.cfg.HasRegistry() {
		return nil
	}

	registryHost, err := k.cfg.GetRegistryHost()
	if err != nil {
		return err
	}

	if loopback, err := system.IsLoopback(registryHost); !loopback || err != nil {
		return err
	}

	networkName := k.cfg.GetContextName()
	networkExists, err := k.containerEngine.DoesNetworkExist(networkName)
	if err != nil {
		return err
	}

	if networkExists {
		isRegistryHostConnected, err := k.containerEngine.IsNetworkConnectedTo(networkName, registryHost)
		if err != nil {
			return err
		}

		if isRegistryHostConnected {
			return nil
		}
	}

	return k.containerEngine.ConnectNetwork(networkName, registryHost)
}

func (k *k3dEnv) StartCluster() error {
	args := []string{
		"cluster",
		"create",
		k.cfg.K8s.ClusterName,
	}

	args, err := k.prepareRegistryArgs(args)
	if err != nil {
		return err
	}

	err = k.executeCmd(args)
	if err != nil {
		return err
	}

	return k.connectNetwork()
}

func (k *k3dEnv) StopCluster() error {
	args := []string{
		"cluster",
		"stop",
		k.cfg.K8s.ClusterName,
	}
	return k.executeCmd(args)
}

func (k *k3dEnv) DeleteCluster() error {
	args := []string{
		"cluster",
		"delete",
		k.cfg.K8s.ClusterName,
	}
	return k.executeCmd(args)
}

func NewK3dEnv(cfg *setup.Configuration) (K8sEnvironment, error) {
	containerEngine, err := container.GetEngine(cfg.Images.Engine)
	if err != nil {
		return nil, err
	}
	return &k3dEnv{cfg: cfg, containerEngine: containerEngine}, nil
}
