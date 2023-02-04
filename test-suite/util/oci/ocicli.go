package oci

import (
	"strings"

	"github.com/marinesovitch/ote/test-suite/util/log"
	"github.com/marinesovitch/ote/test-suite/util/system"
)

func Execute(ociCfgPath string, profile string, cmd string, subcmd []string, args []string) error {
	argv := []string{"--config-file", ociCfgPath, "--profile", profile, cmd}
	argv = append(argv, subcmd...)
	argv = append(argv, args...)
	log.Info.Printf("run oci %s\n", strings.Join(argv, " "))
	return system.Execute("oci", argv...)
}

func BulkDelete(ociCfgPath string, profile string, bucketName string, prefix string) error {
	cmd := "os"
	subcmd := []string{"object", "bulk-delete"}
	args := []string{"--force", "--bucket-name", bucketName, "--prefix", prefix}
	return Execute(ociCfgPath, profile, cmd, subcmd, args)
}
