package local

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestOfflineWholeDirectoryBackupRestorePreservesReplay(t *testing.T) {
	root, source, stateDir, candidate := setupLocalRun(t, "backup-v1", `{"id":1}`)
	if _, err := Activate(context.Background(), candidate, stateDir, "local-user", true); err != nil {
		t.Fatal(err)
	}
	first, err := Run(context.Background(), localRunOptions(source, stateDir))
	if err != nil {
		t.Fatal(err)
	}
	backup, restored := filepath.Join(root, "backup"), filepath.Join(root, "restored")
	copyPrivateTree(t, stateDir, backup)
	copyPrivateTree(t, backup, restored)
	again, err := Run(context.Background(), localRunOptions(source, restored))
	if err != nil {
		t.Fatal(err)
	}
	if again.RunID != first.RunID || !again.AlreadyCommitted {
		t.Fatalf("restored run = %+v, original = %+v; want same reconciled run", again, first)
	}
	batches, err := ReadCommitted(filepath.Join(restored, "data", "destination.jsonl"))
	if err != nil || len(batches) != 1 {
		t.Fatalf("restored destination = %+v, %v; want one committed batch", batches, err)
	}
}

func copyPrivateTree(t *testing.T, source, destination string) {
	t.Helper()
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.Mkdir(target, 0o700)
		}
		if !entry.Type().IsRegular() {
			t.Fatalf("backup includes unsupported non-regular file %s", relative)
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if err != nil {
		t.Fatal(err)
	}
}
