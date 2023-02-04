// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

func isTemplate(path string) (bool, error) {
	rawContents, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	const PatternPrefix = "{{."
	contents := string(rawContents)
	return strings.Contains(contents, PatternPrefix), nil
}

func GenerateK3dRegistryConfig(cfg *Configuration) (string, error) {
	const k3dRegistryConfigFileName = "k3d-registry-config.yaml"
	// k3dRegistryConfigTemplatePath := GetTemplatePath(cfg, k3dRegistryConfigFileName)
	k3dRegistryConfigTemplatePath := cfg.K3d.RegistryConfig

	isTempl, err := isTemplate(k3dRegistryConfigTemplatePath)
	if err != nil {
		return k3dRegistryConfigTemplatePath, err
	}
	if !isTempl {
		// as it is not a template no need to generate, just use the file as it is
		return k3dRegistryConfigTemplatePath, nil
	}

	k3dRegistryConfigYamlPath := cfg.GetOutputPath(k3dRegistryConfigFileName)
	k3dRegistryConfigYamlFile, err := os.Create(k3dRegistryConfigYamlPath)
	if err != nil {
		return k3dRegistryConfigYamlPath, err
	}

	defer k3dRegistryConfigYamlFile.Close()

	type GenerateK3dRegistryConfigYamlData struct {
		Registry string
	}

	data := GenerateK3dRegistryConfigYamlData{
		cfg.Images.Registry,
	}

	tmpl := template.Must(template.ParseFiles(k3dRegistryConfigTemplatePath))
	err = tmpl.Execute(k3dRegistryConfigYamlFile, data)
	if err != nil {
		return k3dRegistryConfigYamlPath, fmt.Errorf("cannot generate %s: %s", k3dRegistryConfigYamlPath, err)
	}

	return k3dRegistryConfigYamlPath, nil
}

func prepareImageName(registryRepository string, image string, tag string) string {
	result := registryRepository
	if len(result) != 0 {
		result += "/"
	}
	result += image + ":" + tag
	return result
}

func GenerateDeployOperatorYaml(cfg *Configuration, destDir string) error {
	const deployOperatorFileName = "deploy-operator.yaml"
	// deployOperatorTemplatePath := GetTemplatePath(cfg, deployOperatorFileName)
	deployOperatorTemplatePath := cfg.Operator.Template

	isTempl, err := isTemplate(deployOperatorTemplatePath)
	if err != nil {
		return err
	}
	if !isTempl {
		// as it is not a template no need to generate, just use the file as it is
		return nil
	}

	deployOperatorYamlPath := filepath.Join(destDir, deployOperatorFileName)
	deployOperatorYamlFile, err := os.Create(deployOperatorYamlPath)
	if err != nil {
		return err
	}

	defer deployOperatorYamlFile.Close()

	type GenerateDeployOperatorYamlData struct {
		RegistryRepository string
		Image              string
		PullPolicy         string
		DebugLevel         int
	}

	data := GenerateDeployOperatorYamlData{
		cfg.GetImageRegistryRepository(),
		prepareImageName(cfg.GetImageRegistryRepository(), cfg.Operator.Image, cfg.Operator.VersionTag),
		cfg.Operator.PullPolicy,
		cfg.Operator.DebugLevel,
	}

	tmpl := template.Must(template.ParseFiles(deployOperatorTemplatePath))
	err = tmpl.Execute(deployOperatorYamlFile, data)
	if err != nil {
		return fmt.Errorf("cannot generate %s: %s", deployOperatorYamlPath, err)
	}

	return nil
}

type GenerateUserSecretsData struct {
	Name         string
	RootUser     string
	RootHost     string
	RootPassword string
}

func GenerateUserSecrets(cfg *Configuration, data *GenerateUserSecretsData) (string, error) {
	const userSecretsFileName = "user-secrets.yaml"
	userSecretsTemplatePath := cfg.GetTemplatePath(userSecretsFileName)

	userSecretsYamlPath := cfg.GetOutputPath(userSecretsFileName)
	userSecretsYamlFile, err := os.Create(userSecretsYamlPath)
	if err != nil {
		return userSecretsYamlPath, err
	}

	defer userSecretsYamlFile.Close()

	tmpl := template.Must(template.ParseFiles(userSecretsTemplatePath))
	err = tmpl.Execute(userSecretsYamlFile, data)
	if err != nil {
		return userSecretsYamlPath, fmt.Errorf("cannot generate %s: %s", userSecretsYamlPath, err)
	}

	return userSecretsYamlPath, nil
}

func GenerateFromGenericFile(cfg *Configuration, templatePath string, data interface{}) (string, error) {
	filename := filepath.Base(templatePath)
	yamlPath := cfg.GetOutputPath(filename)
	outputFile, err := os.Create(yamlPath)
	if err != nil {
		return yamlPath, err
	}
	defer outputFile.Close()

	tmpl := template.Must(template.ParseFiles(templatePath))
	err = tmpl.Execute(outputFile, data)
	if err != nil {
		return yamlPath, fmt.Errorf("cannot generate %s: %s", yamlPath, err)
	}

	return yamlPath, nil
}
