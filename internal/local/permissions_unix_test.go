//go:build !windows

package local

import "os"

func makePermissiveFileForTest(path string) error {
	return os.Chmod(path, 0o644)
}
