// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package executor

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/marinesovitch/ote/test-suite/util/k8s"
	"github.com/marinesovitch/ote/test-suite/util/setup"
	"github.com/marinesovitch/ote/test-suite/util/system"
)

type Deployer struct {
	cfg              *setup.Configuration
	kustomizationDir string
}

func (d *Deployer) kustomizationBaseSubdir() string {
	const baseSubdir = "base"
	return filepath.Join(d.kustomizationDir, baseSubdir)
}

func (d *Deployer) kustomizationTestSuiteSubdir() string {
	const testSuiteSubdir = "test-suite"
	return filepath.Join(d.kustomizationDir, testSuiteSubdir)
}

// ---------------------------

func (d *Deployer) copyKustomizationSkeleton() error {
	const kustomizationSubdir = "kustomization"
	kustomizationSkeletonDir := d.cfg.GetTemplatePath(kustomizationSubdir)
	d.kustomizationDir = d.cfg.GetOutputPath(kustomizationSubdir)
	return system.CopyOrUpdateDir(kustomizationSkeletonDir, d.kustomizationDir)
}

// ---------------------------

func (d *Deployer) resolveYamlPath(cfgYamlPath string) (string, error) {
	if filepath.IsAbs(cfgYamlPath) && system.DoesFileExist(cfgYamlPath) {
		return cfgYamlPath, nil
	}

	yamlPath := filepath.Join(d.cfg.Operator.Directory, cfgYamlPath)
	if !system.DoesFileExist(yamlPath) {
		return yamlPath, fmt.Errorf("cannot find yaml file %s", yamlPath)
	}

	return yamlPath, nil
}

func (d *Deployer) copyBaseYaml(srcYamlPath string, kustomizationBaseDir string) error {
	yamlFileName := filepath.Base(srcYamlPath)
	destYamlPath := filepath.Join(kustomizationBaseDir, yamlFileName)
	return system.CopyOrUpdateFile(srcYamlPath, destYamlPath)
}

func (d *Deployer) copyBaseOperatorYamls() error {
	kustomizationBaseDir := d.kustomizationBaseSubdir()

	const YamlSeparator = ":"
	cfgYamlPaths := strings.Split(d.cfg.Operator.Yamls, YamlSeparator)
	for _, cfgYamlPath := range cfgYamlPaths {
		yamlPath, err := d.resolveYamlPath(cfgYamlPath)
		if err != nil {
			return err
		}

		if err := d.copyBaseYaml(yamlPath, kustomizationBaseDir); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------

func (d *Deployer) generateCustomOperatorYaml() error {
	kustomizationTestSuiteDir := d.kustomizationTestSuiteSubdir()
	return setup.GenerateDeployOperatorYaml(d.cfg, kustomizationTestSuiteDir)
}

// ---------------------------

func (d *Deployer) deployOperator() error {
	kustomizationTestSuiteDir := d.kustomizationTestSuiteSubdir()
	const deploymentFileName = "ote-deployment.yaml"
	deploymentPath := d.cfg.GetOutputPath(deploymentFileName)
	kubectl := k8s.Kubectl{}
	if err := kubectl.Kustomize(kustomizationTestSuiteDir, deploymentPath); err != nil {
		return err
	}

	if err := kubectl.Create(deploymentPath); err != nil {
		// temporary patch
		return kubectl.Apply(deploymentPath)
	}

	return nil
}

// ---------------------------

func (d *Deployer) run() error {
	if err := d.copyKustomizationSkeleton(); err != nil {
		return err
	}

	if err := d.copyBaseOperatorYamls(); err != nil {
		return err
	}

	if err := d.generateCustomOperatorYaml(); err != nil {
		return err
	}

	if err := d.deployOperator(); err != nil {
		return err
	}

	return nil
}

func deploy(cfg *setup.Configuration) error {
	if !cfg.Operator.Deploy {
		return nil
	}

	deployer := Deployer{cfg: cfg}
	return deployer.run()
}
