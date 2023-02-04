// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package setup

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/marinesovitch/ote/test-suite/util/common"
	"github.com/marinesovitch/ote/test-suite/util/log"
	"github.com/marinesovitch/ote/test-suite/util/system"
)

func resolveSuiteRootDirectory() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Error.Fatal(err)
	}

	const TestSuiteDirName = "test-suite"
	return system.FindDirInPath(workingDir, TestSuiteDirName)
}

func loadConfiguration(cfgFilename string, initCfg Configuration, obligatory bool) (Configuration, error) {
	cfgPath := filepath.Join(initCfg.TestSuite.RootDirectory, cfgFilename)
	if obligatory || system.DoesFileExist(cfgPath) {
		return loadConfigFile(cfgPath, initCfg)
	}
	return initCfg, nil
}

func ensureAllNecessaryItems(cfg Configuration) error {
	return system.EnsureDirExist(cfg.TestSuite.OutputDirectory)
}

// returns true if runs under 'test go'
func runsUnderTestGo() bool {
	return flag.Lookup("test.v") != nil
}

func CreateConfiguration(ignoreCommand bool) (Configuration, common.Command, error) {
	var cfg Configuration
	var err error

	cfg.TestSuite.RootDirectory, err = resolveSuiteRootDirectory()
	if err != nil {
		return cfg, common.Unknown, err
	}

	const DefaultConfigurationFile = "default.cfg"
	cfg, err = loadConfiguration(DefaultConfigurationFile, cfg, true)
	if err != nil {
		return cfg, common.Unknown, err
	}

	cfg, err = applyEnvironment(cfg)
	if err != nil {
		return cfg, common.Unknown, err
	}

	const CustomConfigurationFile = "custom.cfg"
	cfg, err = loadConfiguration(CustomConfigurationFile, cfg, false)
	if err != nil {
		return cfg, common.Unknown, err
	}

	var cmd common.Command = common.Unknown
	if !runsUnderTestGo() {
		cmd, cfg, err = parseCommandLine(cfg, ignoreCommand)
		if err != nil {
			return cfg, common.Unknown, err
		}
	}

	cfg, err = resolveSettings(cfg)
	if err != nil {
		return cfg, common.Unknown, err
	}

	err = ensureAllNecessaryItems(cfg)
	if err != nil {
		return cfg, common.Unknown, err
	}

	return cfg, cmd, nil
}
