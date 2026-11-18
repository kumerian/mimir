package local

import (
	"fmt"
	"os/exec"
	"strings"
)

func makePermissiveFileForTest(path string) error {
	output, err := exec.Command("icacls", path, "/grant", "*S-1-1-0:R").CombinedOutput()
	if err != nil {
		return fmt.Errorf("grant test read access: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
