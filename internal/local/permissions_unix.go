//go:build !windows

package local

import (
	"fmt"
	"os"
)

func createPrivateDirectory(path string) error {
	return os.Mkdir(path, 0o700)
}

func isPrivateDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir() && info.Mode().Perm()&0o077 == 0
}

func isPrivateFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode().Perm()&0o077 == 0
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync directory: %w", err)
	}
	return nil
}
