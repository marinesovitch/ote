// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package setup

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/marinesovitch/ote/test-suite/util/common"
	"github.com/marinesovitch/ote/test-suite/util/system"
)

const DefaultAutoDetectValue = "detect"

func findCommand(possibleCommands []string) (string, error) {
	for _, cmd := range possibleCommands {
		if system.DoesCommandExist(cmd) {
			return cmd, nil
		}
	}
	return "", fmt.Errorf("cannot find any of the following commands: %s", strings.Join(possibleCommands, ","))
}

func resolveCommand(cfgCmd string, possibleCommands []string) (string, error) {
	if cfgCmd == DefaultAutoDetectValue {
		return findCommand(possibleCommands)
	} else {
		if system.DoesCommandExist(cfgCmd) {
			return cfgCmd, nil
		} else {
			return "", fmt.Errorf("command not found: %s", cfgCmd)
		}
	}
}

func verifyPullPolicy(cfgPolicy string) (string, error) {
	allowedPolicies := []string{"Always", "IfNotPresent", "Never"}
	for _, allowedPolicy := range allowedPolicies {
		if strings.EqualFold(cfgPolicy, allowedPolicy) {
			return allowedPolicy, nil
		}
	}
	return cfgPolicy, fmt.Errorf("incorrect pull policy %s, allowed values are: %s", cfgPolicy, strings.Join(allowedPolicies, ","))
}

func getDefaultKubeConfigPath() string {
	const KubeConfigEnvVar = "KUBECONFIG"
	kubeConfigPath := os.Getenv(KubeConfigEnvVar)
	if kubeConfigPath != "" {
		return kubeConfigPath
	}

	const KubeConfigHomePath = "~/.kube/config"
	return KubeConfigHomePath
}

func resolveK8sKubeConfig(cfgK8sKubeConfig string, baseDir string) (string, error) {
	var kubeConfigPath string
	if cfgK8sKubeConfig == DefaultAutoDetectValue {
		kubeConfigPath = getDefaultKubeConfigPath()
	} else {
		kubeConfigPath = cfgK8sKubeConfig
	}
	return system.ResolveFile(baseDir, kubeConfigPath, true)
}

func resolveK8sEnvironment(cfgK8sEnv string) (string, error) {
	supportedEnvs := []string{common.EnvMinikube, common.EnvK3d}
	return resolveCommand(cfgK8sEnv, supportedEnvs)
}

func resolveImagesEngine(cfgImagesEngine string) (string, error) {
	supportedEngines := []string{"docker", "podman"}
	return resolveCommand(cfgImagesEngine, supportedEngines)
}

func removeSchemaPrefix(url string) string {
	schemaSeparatorPos := strings.Index(url, common.SchemaSeparator)
	if schemaSeparatorPos == -1 {
		return url
	}
	schemaSeparatorEndPos := schemaSeparatorPos + len(common.SchemaSeparator)
	return url[schemaSeparatorEndPos:]
}

func resolveMinikubeRegistryHost(cfg Configuration) (string, error) {
	cfgRegistry := cfg.GetRegistryUrl()
	if len(cfgRegistry) == 0 {
		return cfgRegistry, nil
	}

	url, err := url.Parse(cfgRegistry)
	if err != nil {
		return cfgRegistry, err
	}

	// check if registry host is localhost at the bottom
	hostname := url.Hostname()

	isLoopback, err := system.IsLoopback(hostname)
	if err != nil {
		return cfgRegistry, err
	}
	if !isLoopback {
		return cfgRegistry, nil
	}

	/*
		localhost registry will not work inside minikube as it runs in its own vm
		so resolve the host IP and override it in registry Url, it will work when
		called from inside of minikube (but not from the host)

		example, assume the following configuration
		- in /etc/hosts there was added host registry.localhost
			127.0.0.1 registry.localhost
		- create local registry registry.localhost:5000 (added registry.localhost in /etc/hosts)
		- get host ip, it may be e.g. 10.0.2.15
		- minikube start --insecure-registry=10.0.2.15:5000
		- minikube ssh && docker pull 10.0.2.15:5000/mysql/mysql-operator:8.0.25, should work
		- btw on the host the above pull cmd will not work if 10.0.2.15 is not added as insecure
			registry but it doesn't matter, because we will need it only inside minikube

		some more details also in the following article:
		https://hasura.io/blog/sharing-a-local-registry-for-minikube-37c7240d0615/
	*/
	hostIP, err := system.ResolveHostIP()
	if err != nil {
		return cfgRegistry, err
	}
	host := hostIP.String()
	port := url.Port()
	if port != "" {
		host += ":" + port
	}
	url.Host = host
	registryHost := url.String()
	return registryHost, nil
}

func resolveImagesRegistryHost(cfg Configuration) (string, error) {
	if cfg.K8s.Environment == common.EnvMinikube {
		registryHost, err := resolveMinikubeRegistryHost(cfg)
		return removeSchemaPrefix(registryHost), err
	}
	return cfg.Images.Registry, nil
}

func resolveSettings(cfg Configuration) (Configuration, error) {
	var err error

	suiteRootDirectory := cfg.TestSuite.RootDirectory

	// test suite
	cfg.TestSuite.E2eDirectory, err = system.ResolveDirectory(suiteRootDirectory, cfg.TestSuite.E2eDirectory, true)
	if err != nil {
		return cfg, err
	}

	cfg.TestSuite.DataDirectory, err = system.ResolveDirectory(suiteRootDirectory, cfg.TestSuite.DataDirectory, true)
	if err != nil {
		return cfg, err
	}

	cfg.TestSuite.OutputDirectory, err = system.ResolveDirectory(suiteRootDirectory, cfg.TestSuite.OutputDirectory, false)
	if err != nil {
		return cfg, err
	}

	// k8s
	cfg.K8s.KubeConfig, err = resolveK8sKubeConfig(cfg.K8s.KubeConfig, suiteRootDirectory)
	if err != nil {
		return cfg, err
	}

	cfg.K8s.Environment, err = resolveK8sEnvironment(cfg.K8s.Environment)
	if err != nil {
		return cfg, err
	}

	// images
	cfg.Images.Engine, err = resolveImagesEngine(cfg.Images.Engine)
	if err != nil {
		return cfg, err
	}

	cfg.Images.Registry, err = resolveImagesRegistryHost(cfg)
	if err != nil {
		return cfg, err
	}

	cfg.Images.PullPolicy, err = verifyPullPolicy(cfg.Images.PullPolicy)
	if err != nil {
		return cfg, err
	}

	// k3d
	cfg.K3d.RegistryConfig, err = system.ResolveFile(suiteRootDirectory, cfg.K3d.RegistryConfig, true)
	if err != nil {
		return cfg, err
	}

	// operator
	cfg.Operator.Directory, err = system.ResolveDirectory(suiteRootDirectory, cfg.Operator.Directory, true)
	if err != nil {
		return cfg, err
	}

	cfg.Operator.PullPolicy, err = verifyPullPolicy(cfg.Operator.PullPolicy)
	if err != nil {
		return cfg, err
	}

	cfg.Operator.Template, err = system.ResolveFile(suiteRootDirectory, cfg.Operator.Template, true)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}
