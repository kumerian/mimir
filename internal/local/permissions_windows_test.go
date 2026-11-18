package local

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrivateDirectoryACL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private")
	if err := ensurePrivateDirectory(path); err != nil {
		t.Fatal(err)
	}
	if !isPrivateDirectory(path) {
		t.Fatal("created directory is not private to the local user")
	}
}

func makePermissiveFileForTest(path string) error {
	output, err := exec.Command("icacls", path, "/grant", "*S-1-1-0:R").CombinedOutput()
	if err != nil {
		return fmt.Errorf("grant test read access: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
