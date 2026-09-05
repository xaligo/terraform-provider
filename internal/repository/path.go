package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	localentity "github.com/xaligo/terraform-provider/internal/entity/local"
)

type PathRepository interface {
	Resolve(sourceDir, outputPath string) (localentity.Paths, error)
	ResolveLexical(sourceDir, outputPath string) (localentity.Paths, error)
	StableID(outputPath string) string
}

type pathRepository struct{}

func NewPathRepository() PathRepository {
	return &pathRepository{}
}

// ResolveLexical normalizes paths without requiring them to exist. It is used
// for disabled plans and refresh/destroy operations that must not scan source.
func (rcvr *pathRepository) ResolveLexical(sourceDir, outputPath string) (localentity.Paths, error) {
	if sourceDir == "" {
		return localentity.Paths{}, fmt.Errorf("source_dir must not be empty")
	}
	if outputPath == "" {
		return localentity.Paths{}, fmt.Errorf("output_path must not be empty")
	}
	sourceAbs, err := filepath.Abs(sourceDir)
	if err != nil {
		return localentity.Paths{}, fmt.Errorf("resolve source_dir: %w", err)
	}
	sourceAbs = filepath.Clean(sourceAbs)
	resolvedOutput := outputPath
	if !filepath.IsAbs(resolvedOutput) {
		resolvedOutput = filepath.Join(sourceAbs, resolvedOutput)
	}
	outputAbs, err := filepath.Abs(resolvedOutput)
	if err != nil {
		return localentity.Paths{}, fmt.Errorf("resolve output_path: %w", err)
	}
	outputAbs = filepath.Clean(outputAbs)
	if filepath.Ext(outputAbs) != ".xal" {
		return localentity.Paths{}, fmt.Errorf("output_path must use the lowercase .xal extension: %s", outputAbs)
	}
	return localentity.Paths{SourceDir: sourceAbs, OutputPath: outputAbs}, nil
}

func (rcvr *pathRepository) Resolve(sourceDir, outputPath string) (localentity.Paths, error) {
	lexical, err := rcvr.ResolveLexical(sourceDir, outputPath)
	if err != nil {
		return localentity.Paths{}, err
	}
	sourceReal, err := filepath.EvalSymlinks(lexical.SourceDir)
	if err != nil {
		return localentity.Paths{}, fmt.Errorf("resolve source_dir symlinks: %w", err)
	}
	sourceInfo, err := os.Stat(sourceReal)
	if err != nil {
		return localentity.Paths{}, fmt.Errorf("stat source_dir: %w", err)
	}
	if !sourceInfo.IsDir() {
		return localentity.Paths{}, fmt.Errorf("source_dir is not a directory: %s", sourceReal)
	}

	outputAbs := lexical.OutputPath
	if !filepath.IsAbs(outputPath) {
		outputAbs = filepath.Join(sourceReal, outputPath)
	}
	outputAbs = filepath.Clean(outputAbs)
	parentReal, err := filepath.EvalSymlinks(filepath.Dir(outputAbs))
	if err != nil {
		return localentity.Paths{}, fmt.Errorf("resolve output_path parent: %w", err)
	}
	parentInfo, err := os.Stat(parentReal)
	if err != nil {
		return localentity.Paths{}, fmt.Errorf("stat output_path parent: %w", err)
	}
	if !parentInfo.IsDir() {
		return localentity.Paths{}, fmt.Errorf("output_path parent is not a directory: %s", parentReal)
	}
	outputReal := filepath.Join(parentReal, filepath.Base(outputAbs))
	if info, statErr := os.Lstat(outputReal); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return localentity.Paths{}, fmt.Errorf("output_path must not be a symbolic link: %s", outputReal)
		}
	} else if !os.IsNotExist(statErr) {
		return localentity.Paths{}, fmt.Errorf("inspect output_path: %w", statErr)
	}
	return localentity.Paths{SourceDir: filepath.Clean(sourceReal), OutputPath: filepath.Clean(outputReal)}, nil
}

func (rcvr *pathRepository) StableID(outputPath string) string {
	digest := sha256.Sum256([]byte("xaligo_diagram\x00" + filepath.Clean(outputPath)))
	return "xaligo-diagram-" + hex.EncodeToString(digest[:])
}
