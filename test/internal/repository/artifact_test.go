package repository_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	localentity "github.com/xaligo/terraform-provider/internal/entity/local"
	"github.com/xaligo/terraform-provider/internal/repository"
)

func TestArtifactStoreOwnershipDriftAndSafeDelete(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "diagram.xal")
	store := repository.NewArtifactRepository()

	if inspection, err := store.Inspect(path); err != nil || inspection.Exists || inspection.Digest != "" {
		t.Fatalf("initial Inspect() = %#v, %v", inspection, err)
	}
	if err := store.Write(path, []byte("first"), localentity.WriteOptions{}); err != nil {
		t.Fatalf("initial Write() error = %v", err)
	}
	firstDigest := digestOf([]byte("first"))
	if err := store.Write(path, []byte("second"), localentity.WriteOptions{ExpectedPreviousDigest: firstDigest}); err != nil {
		t.Fatalf("owned Write() error = %v", err)
	}
	secondDigest := digestOf([]byte("second"))
	if err := os.WriteFile(path, []byte("external"), 0o644); err != nil {
		t.Fatalf("modify output externally: %v", err)
	}
	if err := store.Write(path, []byte("third"), localentity.WriteOptions{ExpectedPreviousDigest: secondDigest}); err == nil || !strings.Contains(err.Error(), "externally modified") {
		t.Fatalf("drifted Write() error = %v", err)
	}
	if err := store.Write(path, []byte("third"), localentity.WriteOptions{Overwrite: true}); err != nil {
		t.Fatalf("overwrite Write() error = %v", err)
	}

	result, err := store.Delete(path, secondDigest)
	if err != nil || result.Deleted || result.Warning == "" {
		t.Fatalf("mismatched Delete() = %#v, %v", result, err)
	}
	thirdDigest := digestOf([]byte("third"))
	result, err = store.Delete(path, thirdDigest)
	if err != nil || !result.Deleted || result.Warning != "" {
		t.Fatalf("owned Delete() = %#v, %v", result, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("deleted output stat error = %v", err)
	}
	if matches, err := filepath.Glob(filepath.Join(directory, ".diagram.xal.tmp-*")); err != nil || len(matches) != 0 {
		t.Fatalf("temporary outputs = %v, %v", matches, err)
	}
}

func TestArtifactStoreRefusesSymbolicLinks(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	target := filepath.Join(directory, "target.xal")
	link := filepath.Join(directory, "link.xal")
	if err := os.WriteFile(target, []byte("target"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("create symbolic link: %v", err)
	}
	store := repository.NewArtifactRepository()
	if _, err := store.Inspect(link); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("Inspect(symlink) error = %v", err)
	}
	if err := store.Write(link, []byte("replacement"), localentity.WriteOptions{Overwrite: true}); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("Write(symlink) error = %v", err)
	}
	result, err := store.Delete(link, digestOf([]byte("target")))
	if err != nil || result.Deleted || !strings.Contains(result.Warning, "symbolic-link") {
		t.Fatalf("Delete(symlink) = %#v, %v", result, err)
	}
	if content, err := os.ReadFile(target); err != nil || string(content) != "target" {
		t.Fatalf("target after symlink operations = %q, %v", content, err)
	}
}

func TestArtifactStoreSerializesConcurrentSamePathWrites(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "diagram.xal")
	store := repository.NewArtifactRepository()
	payloads := []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot"}
	var wait sync.WaitGroup
	errors := make(chan error, len(payloads))
	for _, payload := range payloads {
		payload := payload
		wait.Add(1)
		go func() {
			defer wait.Done()
			errors <- store.Write(path, []byte(payload), localentity.WriteOptions{Overwrite: true})
		}()
	}
	wait.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatalf("concurrent Write() error = %v", err)
		}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read concurrent output: %v", err)
	}
	found := false
	for _, payload := range payloads {
		if string(content) == payload {
			found = true
		}
	}
	if !found {
		t.Fatalf("concurrent output %q is not one complete payload", content)
	}
}

func digestOf(content []byte) string {
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:])
}
