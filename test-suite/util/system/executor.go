// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package system

import (
	"os"
	"os/exec"
	"strings"

	"github.com/marinesovitch/ote/test-suite/util/log"
)

func ExecuteGetOutput(name string, args ...string) (string, error) {
	log.Info.Printf("%s %s", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmdStdoutStderr, err := cmd.CombinedOutput()
	return string(cmdStdoutStderr), err
}

func Execute(name string, args ...string) error {
	cmdStdoutStderr, err := ExecuteGetOutput(name, args...)
	if cmdStdoutStderr != "" {
		if err != nil {
			log.Error.Print(cmdStdoutStderr)
		} else {
			log.Info.Print(cmdStdoutStderr)
		}
	}
	return err
}

// execute a process with a redirected output, just like in case of a shell pipe 'app [args...] > output'
func ExecuteRedirectOutput(outputPath string, name string, args ...string) error {
	log.Info.Printf("%s %s > %s", name, strings.Join(args, " "), outputPath)
	cmd := exec.Command(name, args...)

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	cmd.Stdout = outputFile

	return cmd.Run()
}

// execute a process with a redirected input, just like in case of a shell pipe 'input | app [args...]'
func ExecuteWithInput(input string, name string, args ...string) error {
	log.Info.Printf("input | %s %s", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(input)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
