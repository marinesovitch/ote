// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package system

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/marinesovitch/ote/test-suite/util/log"
)

func DoesFileExist(path string) bool {
	if fileInfo, err := os.Stat(path); err == nil {
		return !fileInfo.IsDir()
	} else {
		return false
	}
}

func DoesDirExist(path string) bool {
	if fileInfo, err := os.Stat(path); err == nil {
		return fileInfo.IsDir()
	} else {
		return false
	}
}

func VerifyFileExist(path string) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("path %s is a directory", path)
	}

	return nil
}

func VerifyDirExist(path string) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("directory %s doesn't exist", path)
	}

	return nil
}

func EnsureDirExist(path string) error {
	if DoesDirExist(path) {
		return nil
	}
	return os.MkdirAll(path, os.ModePerm)
}

func PathExpand(rawPath string) string {
	const HomeSymbol = "~"
	if !strings.Contains(rawPath, HomeSymbol) {
		return rawPath
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return rawPath
	}
	return strings.ReplaceAll(rawPath, HomeSymbol, homeDir)
}

func ResolveFile(baseDir string, cfgFile string, mustExist bool) (string, error) {
	file := PathExpand(cfgFile)
	if file == "" {
		return file, errors.New("file cannot be an empty value")
	}
	if !path.IsAbs(file) {
		file = filepath.Join(baseDir, file)
	}
	file = filepath.Clean(file)
	if mustExist && !DoesFileExist(file) {
		return file, fmt.Errorf("file %s doesn't exist", file)
	}
	return file, nil
}

func ResolveDirectory(baseDir string, cfgDir string, mustExist bool) (string, error) {
	dir := PathExpand(cfgDir)
	if dir == "" {
		return dir, errors.New("directory cannot be an empty value")
	}
	if !path.IsAbs(dir) {
		dir = filepath.Join(baseDir, dir)
	}
	dir = filepath.Clean(dir)
	if mustExist {
		err := VerifyDirExist(dir)
		if err != nil {
			return dir, err
		}
	}
	return dir, nil
}

func ResolveCommand(cmd string) (string, error) {
	return exec.LookPath(cmd)
}

func DoesCommandExist(cmd string) bool {
	path, err := ResolveCommand(cmd)
	if err != nil {
		log.Error.Println(err)
		return false
	}
	return len(path) > 0
}

func FindDirInPath(path string, dirName string) (string, error) {
	originalPath := path
	for {
		dir, suffix := filepath.Split(path)
		if suffix == dirName {
			return path, nil
		}
		if dir == "" || suffix == "" {
			return "", fmt.Errorf("path '%s' doesn't contain subdirectory '%s'", originalPath, dirName)
		}
		path = filepath.Clean(dir)
	}
}

func CanUpdateFile(srcModTime time.Time, destPath string) (bool, error) {
	if !DoesFileExist(destPath) {
		return true, nil
	}

	destFileStat, err := os.Stat(destPath)
	if err != nil {
		return false, err
	}

	if !destFileStat.Mode().IsRegular() {
		return false, fmt.Errorf("%s is not a regular file", destFileStat)
	}

	return srcModTime.After(destFileStat.ModTime()), nil
}

func CopyOrUpdateFile(srcPath string, destPath string) error {
	srcFileStat, err := os.Stat(srcPath)
	if err != nil {
		return err
	}

	if !srcFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", srcPath)
	}

	if canUpdate, err := CanUpdateFile(srcFileStat.ModTime(), destPath); !canUpdate || err != nil {
		return err
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dest, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, src)
	return err
}

func CopyOrUpdateDir(srcDir string, destDir string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		relPath := strings.Replace(path, srcDir, "", 1)
		if relPath == "" {
			if DoesDirExist(destDir) {
				return nil
			}
			return os.Mkdir(destDir, info.Mode())
		}

		if info.IsDir() {
			destSubdir := filepath.Join(destDir, relPath)
			if DoesDirExist(destSubdir) {
				return nil
			}
			return os.Mkdir(destSubdir, info.Mode())
		}

		srcFilePath := filepath.Join(srcDir, relPath)
		destFilePath := filepath.Join(destDir, relPath)
		return CopyOrUpdateFile(srcFilePath, destFilePath)
	})
}
