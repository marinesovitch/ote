package suite

import (
	"os"

	"github.com/marinesovitch/ote/test-suite/util/k8s"
	"github.com/marinesovitch/ote/test-suite/util/setup"

	"k8s.io/client-go/tools/clientcmd"
)

type Suite struct {
	Cfg    setup.Configuration
	Client *k8s.Client
	Dir    string
}

func CreateSuite() (*Suite, error) {
	suiteDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	cfg, _, err := setup.CreateConfiguration(true)
	if err != nil {
		return nil, err
	}

	kubeCfg, kubeErr := clientcmd.BuildConfigFromFlags("", cfg.K8s.KubeConfig)
	if kubeErr != nil {
		return nil, kubeErr
	}

	client, err := k8s.NewClient(&cfg, kubeCfg)
	if err != nil {
		return nil, err
	}

	suite := Suite{
		Cfg:    cfg,
		Client: client,
		Dir:    suiteDir,
	}

	return &suite, nil
}

func (s *Suite) NewUnitSetup(namespace string) (*Unit, error) {
	return s.NewUnitSetupWithAuxNamespace(namespace, "")
}

func (s *Suite) NewUnitSetupWithAuxNamespace(namespace string, auxNamespace string) (*Unit, error) {
	unit := Unit{
		Cfg:          s.Cfg,
		Client:       s.Client,
		Namespace:    namespace,
		AuxNamespace: auxNamespace,
		SuiteDir:     s.Dir,
	}

	return &unit, unit.Setup()
}
