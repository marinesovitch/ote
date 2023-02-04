// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/marinesovitch/ote/test-suite/util/auxi"
	"github.com/marinesovitch/ote/test-suite/util/common"
)

type Configuration struct {
	TestSuite struct {
		RootDirectory   string
		E2eDirectory    string
		DataDirectory   string
		OutputDirectory string
	}

	K8s struct {
		KubeConfig    string
		Environment   string
		ClusterName   string
		DeleteAtStart bool
		DeleteAtStop  bool
	}

	Images struct {
		Engine                   string
		Registry                 string
		Repository               string
		PullPolicy               string
		DefaultVersionTag        string
		DefaultServerVersionTag  string
		MinSupportedMysqlVersion string
		MysqlServerImage         string
		MysqlRouterImage         string
		MysqlServerEEImage       string
		MysqlRouterEEImage       string
	}

	Minikube struct {
		RegistryInsecure bool
	}

	K3d struct {
		RegistryConfig string
	}

	Operator struct {
		Deploy     bool
		Directory  string
		Yamls      string
		Image      string
		ImageEE    string
		VersionTag string
		PullPolicy string
		Template   string
		DebugLevel int
	}

	Enterprise struct {
		Enable bool
	}

	Oci struct {
		Enable     bool
		ConfigPath string
		BucketName string
	}
}

func (c *Configuration) GetContextName() string {
	if c.K8s.Environment == "k3d" {
		return "k3d-" + c.K8s.ClusterName
	}
	return c.K8s.ClusterName
}

func (c *Configuration) HasRegistry() bool {
	return len(c.Images.Registry) > 0
}

func (c *Configuration) GetRegistryUrl() string {
	if !c.HasRegistry() {
		return ""
	}
	if strings.Contains(c.Images.Registry, common.SchemaSeparator) {
		return c.Images.Registry
	}
	const DefaultSchema = "http"
	return DefaultSchema + common.SchemaSeparator + c.Images.Registry
}

func (c *Configuration) GetRegistryHostPort() []string {
	const PortSeparator = ":"
	return strings.Split(c.Images.Registry, PortSeparator)
}

func (c *Configuration) GetRegistryHost() (string, error) {
	registryHostPort := c.GetRegistryHostPort()
	if len(registryHostPort) == 0 {
		return "", errors.New("can't get registry host:port from '" + c.Images.Registry + "'")
	}
	return registryHostPort[0], nil
}

func (c *Configuration) GetImageRegistryRepository() string {
	if c.HasRegistry() {
		if c.Images.Repository != "" {
			return c.Images.Registry + "/" + c.Images.Repository
		} else {
			return c.Images.Registry
		}
	} else {
		return c.Images.Repository
	}
}

func (c *Configuration) GetTestDataPath(fileName string) string {
	return filepath.Join(c.TestSuite.DataDirectory, fileName)
}

func (c *Configuration) GetTemplatePath(templateName string) string {
	const templateSubdir = "template"
	return filepath.Join(c.TestSuite.RootDirectory, templateSubdir, templateName)
}

func (c *Configuration) GetOutputPath(subpath string) string {
	return filepath.Join(c.TestSuite.OutputDirectory, subpath)
}

func (c *Configuration) CheckEnterpriseConfig() error {
	if !c.Enterprise.Enable {
		return fmt.Errorf("enterprise tests are skipped")
	}
	return nil
}

func (c *Configuration) CheckOCIConfig() error {
	if !c.Oci.Enable {
		return fmt.Errorf("OCI tests are skipped")
	}

	if len(c.Oci.BucketName) == 0 || len(c.Oci.ConfigPath) == 0 {
		return fmt.Errorf("incomplete OCI setup, BucketName: '%s', ConfigPath: '%s'", c.Oci.BucketName, c.Oci.ConfigPath)
	}

	return nil
}

func loadConfigFile(path string, initCfg Configuration) (Configuration, error) {
	cfgJson, err := os.ReadFile(path)
	if err != nil {
		return initCfg, err
	}

	cfg := initCfg
	err = json.Unmarshal(cfgJson, &cfg)
	if err != nil {
		return initCfg, auxi.FromJsonError(path, err)
	}

	return cfg, nil
}
