package singboxscripts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractTo(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "singbox_test")

	err := ExtractTo(targetDir)
	if err != nil {
		t.Fatalf("ExtractTo failed: %v", err)
	}

	requiredFiles := []string{
		"sing-box.sh",
		"src/api.sh",
		"src/core.sh",
		"src/sub.sh",
		"src/init.sh",
		"src/systemd.sh",
		"src/dns.sh",
		"src/bbr.sh",
		"src/caddy.sh",
		"src/download.sh",
		"src/help.sh",
		"src/import.sh",
		"src/log.sh",
	}

	for _, rf := range requiredFiles {
		fullPath := filepath.Join(targetDir, rf)
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Errorf("expected extracted file %s does not exist: %v", rf, err)
			continue
		}

		if info.Size() == 0 {
			t.Errorf("extracted file %s is empty", rf)
		}

		// Check executable bit
		if info.Mode()&0111 == 0 {
			t.Errorf("extracted file %s is not executable, mode: %v", rf, info.Mode())
		}
	}
}
