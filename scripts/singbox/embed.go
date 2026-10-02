package singboxscripts

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed sing-box.sh src/*
var ScriptsFS embed.FS

// ExtractTo extracts all embedded sing-box scripts into the specified target directory.
// All created directories and files are ensured to have 0755 executable permissions.
func ExtractTo(targetDir string) error {
	if targetDir == "" {
		return fmt.Errorf("target directory cannot be empty")
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", targetDir, err)
	}

	return fs.WalkDir(ScriptsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == "." {
			return nil
		}

		destPath := filepath.Join(targetDir, path)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		content, err := ScriptsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
		}

		if err := os.WriteFile(destPath, content, 0755); err != nil {
			return fmt.Errorf("failed to write file %s: %w", destPath, err)
		}

		if err := os.Chmod(destPath, 0755); err != nil {
			return fmt.Errorf("failed to chmod 0755 on %s: %w", destPath, err)
		}

		return nil
	})
}
