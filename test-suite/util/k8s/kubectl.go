// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package k8s

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/marinesovitch/ote/test-suite/util/system"
)

type Kubectl struct {
}

const TryOnce = 1
const MaxTrials = 5

func tryRun(f func(...string) error, maxTrials int, args ...string) (err error) {
	for i := 0; i < maxTrials; i++ {
		err = f(args...)
		if err == nil {
			break
		}
		// err = fmt.Errorf("execution error: kubectl %v -> %v", args, err)
		// log.Print(err)
		time.Sleep(2 * time.Second)
	}
	return err
}

func (k Kubectl) run(trials int, args ...string) error {
	return tryRun(func(args ...string) error {
		return system.Execute("kubectl", args...)
	}, trials, args...)
}

func (k Kubectl) runGetOutput(args ...string) (output string, err error) {
	err = tryRun(func(args ...string) error {
		output, err = system.ExecuteGetOutput("kubectl", args...)
		return err
	}, 1, args...)
	return output, err
}

func (k Kubectl) runRedirectOutput(outputPath string, args ...string) error {
	return tryRun(func(args ...string) error {
		return system.ExecuteRedirectOutput(outputPath, "kubectl", args...)
	}, MaxTrials, args...)
}

func (k Kubectl) runWithInput(input string, args ...string) error {
	return tryRun(func(args ...string) error {
		return system.ExecuteWithInput(input, "kubectl", args...)
	}, MaxTrials, args...)
}

func (k Kubectl) Kustomize(kustomizationDir string, outputPath string) error {
	return k.runRedirectOutput(outputPath, "kustomize", "--reorder=none", kustomizationDir)
}

func (k Kubectl) Create(path string) error {
	return k.run(TryOnce, "create", "-f", path)
}

func (k Kubectl) Apply(path string) error {
	return k.run(TryOnce, "apply", "-f", path)
}

func (k Kubectl) ApplyInNamespace(namespace string, path string) error {
	return k.run(TryOnce, "apply", "-f", path, "-n", namespace)
}

func (k Kubectl) ApplyGetOutput(namespace string, path string) (string, error) {
	return k.runGetOutput("apply", "-f", path, "-n", namespace)
}

func (k Kubectl) Describe(namespace string, resource Kind, name string) (string, error) {
	return k.runGetOutput("describe", resource.String(), name, "-n", namespace)
}

func (k Kubectl) Logs(namespace string, name string, containerId ContainerId) (string, error) {
	containerName := GetContainerName(containerId)
	return k.runGetOutput("logs", name, "-c", containerName, "-n", namespace)
}

func (k Kubectl) Execute(namespace string, name string, containerId ContainerId, args ...string) error {
	containerName := GetContainerName(containerId)
	cmdLine := []string{"exec", name, "-c", containerName, "-n", namespace, "--"}
	cmdLine = append(cmdLine, args...)
	return k.run(MaxTrials, cmdLine...)
}

func (k Kubectl) ExecuteGetOutput(namespace string, name string, containerId ContainerId, args ...string) (string, error) {
	containerName := GetContainerName(containerId)
	cmdLine := []string{"exec", name, "-c", containerName, "-n", namespace, "--"}
	cmdLine = append(cmdLine, args...)
	return k.runGetOutput(cmdLine...)
}

func (k Kubectl) ExecuteWithInput(input string, namespace string, name string, containerId ContainerId, args ...string) error {
	containerName := GetContainerName(containerId)
	cmdLine := []string{"exec", name, "-c", containerName, "-n", namespace}
	cmdLine = append(cmdLine, args...)
	return k.runWithInput(input, cmdLine...)
}

func (k Kubectl) Run(args ...string) error {
	return k.run(MaxTrials, args...)
}

func (k Kubectl) PortForward(namespace string, podName string, podPort int) (*exec.Cmd, int, error) {
	cmd := exec.Command(
		"kubectl", "port-forward",
		fmt.Sprintf("pod/%s", podName),
		fmt.Sprintf(":%d", podPort),
		"--address", "127.0.0.1",
		"-n", namespace)

	reader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, -1, err
	}
	err = cmd.Start()
	if err != nil {
		return nil, -1, err
	}

	var output []byte = make([]byte, 1024)
	_, err = reader.Read(output)
	if err != nil {
		return nil, -1, err
	}

	line := string(output)
	parsedOutput := strings.Split(line, "->")
	if len(parsedOutput) == 0 {
		return nil, -1, err
	}

	rawPorts := parsedOutput[0]
	parsedPorts := strings.Split(rawPorts, ":")
	if len(parsedPorts) == 0 {
		return nil, -1, err
	}

	portStr := strings.Trim(parsedPorts[len(parsedPorts)-1], " ")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, -1, err
	}

	return cmd, port, nil
}
