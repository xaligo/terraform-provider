package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	localentity "github.com/xaligo/terraform-provider/internal/entity/local"
)

type ArtifactRepository interface {
	Inspect(path string) (localentity.Inspection, error)
	Write(path string, content []byte, options localentity.WriteOptions) error
	Delete(path, expectedDigest string) (localentity.DeleteResult, error)
}

type artifactRepository struct {
	mutex      sync.Mutex
	locks      map[string]*sync.Mutex
	references map[string]int
}

func NewArtifactRepository() ArtifactRepository {
	return &artifactRepository{
		locks:      make(map[string]*sync.Mutex),
		references: make(map[string]int),
	}
}

func (rcvr *artifactRepository) Inspect(path string) (inspection localentity.Inspection, err error) {
	err = rcvr.withLock(path, func() error {
		var inspectErr error
		inspection.Digest, inspection.Exists, inspectErr = inspectRegular(path)
		return inspectErr
	})
	return inspection, err
}

func (rcvr *artifactRepository) Write(path string, content []byte, options localentity.WriteOptions) error {
	return rcvr.withLock(path, func() error {
		digest, exists, err := inspectRegular(path)
		if err != nil {
			return err
		}
		if exists && !options.Overwrite {
			if options.ExpectedPreviousDigest == "" {
				return fmt.Errorf("refusing to overwrite unowned output %s; set overwrite = true to take ownership", path)
			}
			if digest != options.ExpectedPreviousDigest {
				return fmt.Errorf("refusing to overwrite externally modified output %s; set overwrite = true to replace it", path)
			}
		}
		return atomicWrite(path, content, exists, digest)
	})
}

// Delete removes a regular file only when it still matches expectedDigest.
func (rcvr *artifactRepository) Delete(path, expectedDigest string) (result localentity.DeleteResult, err error) {
	if expectedDigest == "" {
		return result, nil
	}
	err = rcvr.withLock(path, func() error {
		info, statErr := os.Lstat(path)
		if os.IsNotExist(statErr) {
			return nil
		}
		if statErr != nil {
			return fmt.Errorf("inspect output before delete: %w", statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			result.Warning = fmt.Sprintf("preserved symbolic-link output %s", path)
			return nil
		}
		if !info.Mode().IsRegular() {
			result.Warning = fmt.Sprintf("preserved non-regular output %s", path)
			return nil
		}
		digest, _, digestErr := inspectRegular(path)
		if digestErr != nil {
			return digestErr
		}
		if digest != expectedDigest {
			result.Warning = fmt.Sprintf("preserved externally modified output %s", path)
			return nil
		}
		if removeErr := os.Remove(path); removeErr != nil {
			return fmt.Errorf("delete managed output: %w", removeErr)
		}
		result.Deleted = true
		return syncDirectory(filepath.Dir(path))
	})
	return result, err
}

func inspectRegular(path string) (string, bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("inspect output: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", false, fmt.Errorf("output must not be a symbolic link: %s", path)
	}
	if !info.Mode().IsRegular() {
		return "", false, fmt.Errorf("output is not a regular file: %s", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return "", false, fmt.Errorf("open output: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", false, fmt.Errorf("hash output: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), true, nil
}

func atomicWrite(path string, content []byte, initiallyExists bool, initialDigest string) (returnErr error) {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary output: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		if returnErr != nil {
			_ = temporary.Close()
		}
		_ = os.Remove(temporaryPath)
	}()
	if err := temporary.Chmod(0o644); err != nil {
		return fmt.Errorf("set temporary output permissions: %w", err)
	}
	if _, err := temporary.Write(content); err != nil {
		return fmt.Errorf("write temporary output: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary output: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary output: %w", err)
	}
	currentDigest, currentlyExists, err := inspectRegular(path)
	if err != nil {
		return fmt.Errorf("reinspect output before replacement: %w", err)
	}
	if initiallyExists != currentlyExists || (initiallyExists && currentDigest != initialDigest) {
		return fmt.Errorf("output changed while replacement was being prepared: %s", path)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace output atomically: %w", err)
	}
	return syncDirectory(directory)
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open output directory for sync: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync output directory: %w", err)
	}
	return nil
}

func (rcvr *artifactRepository) withLock(path string, operation func() error) error {
	cleanPath := filepath.Clean(path)
	rcvr.mutex.Lock()
	lock := rcvr.locks[cleanPath]
	if lock == nil {
		lock = &sync.Mutex{}
		rcvr.locks[cleanPath] = lock
	}
	rcvr.references[cleanPath]++
	rcvr.mutex.Unlock()

	lock.Lock()
	err := operation()
	lock.Unlock()

	rcvr.mutex.Lock()
	rcvr.references[cleanPath]--
	if rcvr.references[cleanPath] == 0 {
		delete(rcvr.locks, cleanPath)
		delete(rcvr.references, cleanPath)
	}
	rcvr.mutex.Unlock()
	return err
}
