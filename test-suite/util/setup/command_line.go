// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package setup

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/marinesovitch/ote/test-suite/util/common"
)

const listOfCommands = "[start|stop|deploy]"

func parseCommand(args []string) (common.Command, error) {
	if len(args) != 1 {
		return common.Unknown, fmt.Errorf("expected a single command %s but got '%s'", listOfCommands, strings.Join(args, " "))
	}
	cmd := args[0]
	switch cmd {
	case "start":
		return common.Start, nil
	case "stop":
		return common.Stop, nil
	case "deploy":
		return common.Deploy, nil
	default:
		return common.Unknown, fmt.Errorf("unknown command %s - expected one of %s", cmd, listOfCommands)
	}
}

func applyEnvVariable(envar string, setting *string) {
	enval, ok := os.LookupEnv(envar)
	if ok {
		*setting = enval
	}
}

func applyEnvVariableBool(envar string, setting *bool) {
	if enval, ok := os.LookupEnv(envar); ok {
		var err error
		*setting, err = strconv.ParseBool(enval)
		if err != nil {
			*setting = false
		}
	}
}

func applyEnvironment(initCfg Configuration) (Configuration, error) {
	cfg := initCfg

	applyEnvVariable("OPERATOR_TEST_REGISTRY", &cfg.Images.Registry)
	applyEnvVariable("OPERATOR_TEST_REPOSITORY", &cfg.Images.Repository)
	applyEnvVariable("OPERATOR_TEST_PULL_POLICY", &cfg.Operator.PullPolicy)

	applyEnvVariable("OPERATOR_TEST_IMAGE_NAME", &cfg.Operator.Image)
	applyEnvVariable("OPERATOR_TEST_EE_IMAGE_NAME", &cfg.Operator.ImageEE)
	applyEnvVariable("OPERATOR_TEST_VERSION_TAG", &cfg.Operator.VersionTag)
	applyEnvVariable("OPERATOR_TEST_PULL_POLICY", &cfg.Operator.PullPolicy)

	applyEnvVariableBool("OPERATOR_TEST_ENABLE_ENTERPRISE", &cfg.Enterprise.Enable)

	applyEnvVariableBool("OPERATOR_TEST_ENABLE_OCI", &cfg.Oci.Enable)
	applyEnvVariable("OPERATOR_TEST_OCI_CONFIG_PATH", &cfg.Oci.ConfigPath)
	applyEnvVariable("OPERATOR_TEST_OCI_BUCKET", &cfg.Oci.BucketName)

	applyEnvVariable("OPERATOR_TEST_K8S_CLUSTER_NAME", &cfg.K8s.ClusterName)

	return cfg, nil
}

func parseCommandLine(initCfg Configuration, ignoreCommand bool) (common.Command, Configuration, error) {
	e2eDirectory := flag.String("e2e-dir", initCfg.TestSuite.E2eDirectory, "directory with e2e tests")
	dataDirectory := flag.String("data-dir", initCfg.TestSuite.DataDirectory, "directory with e2e data")
	outputDirectory := flag.String("output-dir", initCfg.TestSuite.OutputDirectory, "output directory for log and tmp files")

	kubeConfig := flag.String("kubecfg", initCfg.K8s.KubeConfig, "kube config path (if 'detect' it first tries ${KUBECONFIG}, then path ~/.kube/config)")
	environment := flag.String("env", initCfg.K8s.Environment, "environment [detect|k3d|minikube]")
	clusterName := flag.String("cluster-name", initCfg.K8s.ClusterName, "cluster name used for testing")
	skipDeleteCluster := flag.Bool("skip-delete", false, "skip deleting cluster")

	containerEngine := flag.String("engine", initCfg.Images.Engine, "container engine [detect|docker|podman]")
	registry := flag.String("registry", initCfg.Images.Registry, "registry, e.g. registry.localhost:5000")
	repository := flag.String("repository", initCfg.Images.Repository, "repository, e.g. qa")
	pullPolicy := flag.String("pull-policy", initCfg.Images.PullPolicy, "pull policy [Always|IfNotPresent|Never]")

	minikubeRegistryInsecure := flag.Bool("minikube-registry-insecure", initCfg.Minikube.RegistryInsecure, "is minikube registry insecure")

	k3dRegistryConfig := flag.String("k3d-registry-cfg", initCfg.K3d.RegistryConfig, "path to k3d registry config yaml or its template")

	skipDeployOperator := flag.Bool("skip-deploy", !initCfg.Operator.Deploy, "skip deploying operator")
	operatorDir := flag.String("operator-dir", initCfg.Operator.Directory, "operator directory")
	operatorYamls := flag.String("operator-yamls", initCfg.Operator.Yamls, "operator yamls")
	operatorImage := flag.String("operator-image", initCfg.Operator.Image, "image custom config path")
	operatorImageEE := flag.String("operator-image-ee", initCfg.Operator.ImageEE, "enterprise edition image custom config path")
	operatorPullPolicy := flag.String("operator-pull-policy", initCfg.Operator.PullPolicy,
		"pull policy for operator [Always|IfNotPresent|Never]")
	operatorVersionTag := flag.String("operator-tag", initCfg.Operator.VersionTag, "version tag for operator image")
	operatorTemplate := flag.String("operator-template", initCfg.Operator.Template, "path to operator deploy yaml or its template")
	debugLevel := flag.Int("dbg", initCfg.Operator.DebugLevel, "debug level")

	enterpriseEnable := flag.Bool("enterprise", initCfg.Enterprise.Enable, "run enterprise tests")

	ociEnable := flag.Bool("oci", initCfg.Oci.Enable, "run OCI tests")
	ociConfigPath := flag.String("oci-cfg-path", initCfg.Oci.ConfigPath, "path to a file with OCI profiles")
	ociBucketName := flag.String("oci-bucket-name", initCfg.Oci.BucketName, "OCI bucket name")

	defaultUsage := flag.Usage
	flag.Usage = func() {
		defaultUsage()
		fmt.Fprintf(flag.CommandLine.Output(), "Command %s\n", listOfCommands)
	}

	flag.Parse()

	command := common.Unknown
	if !ignoreCommand {
		var err error
		command, err = parseCommand(flag.Args())
		if err != nil {
			return command, initCfg, err
		}
	}

	cfg := initCfg
	cfg.TestSuite.E2eDirectory = *e2eDirectory
	cfg.TestSuite.DataDirectory = *dataDirectory
	cfg.TestSuite.OutputDirectory = *outputDirectory

	cfg.K8s.KubeConfig = *kubeConfig
	cfg.K8s.Environment = *environment
	cfg.K8s.ClusterName = *clusterName
	if command == common.Start {
		cfg.K8s.DeleteAtStart = !*skipDeleteCluster
	} else if command == common.Stop {
		cfg.K8s.DeleteAtStop = !*skipDeleteCluster
	}

	cfg.Images.Engine = *containerEngine
	cfg.Images.Registry = *registry
	cfg.Images.Repository = *repository
	cfg.Images.PullPolicy = *pullPolicy

	cfg.Minikube.RegistryInsecure = *minikubeRegistryInsecure

	cfg.K3d.RegistryConfig = *k3dRegistryConfig

	cfg.Operator.Deploy = !*skipDeployOperator
	cfg.Operator.Directory = *operatorDir
	cfg.Operator.Yamls = *operatorYamls
	cfg.Operator.Image = *operatorImage
	cfg.Operator.ImageEE = *operatorImageEE
	cfg.Operator.VersionTag = *operatorVersionTag
	cfg.Operator.PullPolicy = *operatorPullPolicy
	cfg.Operator.Template = *operatorTemplate
	cfg.Operator.DebugLevel = *debugLevel

	cfg.Enterprise.Enable = *enterpriseEnable

	cfg.Oci.Enable = *ociEnable
	cfg.Oci.ConfigPath = *ociConfigPath
	cfg.Oci.BucketName = *ociBucketName

	return command, cfg, nil
}
