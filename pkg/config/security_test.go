package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecuritySettingsSurviveRestart(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("UI_PASSWORD", "BOOTSTRAP-OLD")
	t.Setenv("UI_USERNAME", "admin")
	c := LoadConfig()
	dto := c.GetSettings()
	dto.UIPassword = "NEW-TEST-PASSWORD"
	dto.AgeEncryptEnabled = true
	dto.AgePublicKey = "testkey"
	if err := c.UpdateSettings(dto); err != nil {
		t.Fatal(err)
	}
	c2 := LoadConfig()
	if c2.UIPassword != "NEW-TEST-PASSWORD" || !c2.AgeEncryptEnabled || c2.AgePublicKey != "testkey" {
		t.Fatal("settings lost on restart")
	}
	info, err := os.Stat(filepath.Join(c.DataDir, "settings.json"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("settings permissions", err)
	}
	if c.GetSettings().UIPassword != "" {
		t.Fatal("password exposed by GetSettings")
	}
}
func TestSecurityInitialPassword(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("UI_PASSWORD", "")
	t.Setenv("UI_USERNAME", "")
	t.Setenv("UI_HOST", "")
	c := LoadConfig()
	if c.UIPassword == "aimilivpn" || len(c.UIPassword) < 32 || c.UIHost != "127.0.0.1" {
		t.Fatal("insecure defaults")
	}
	if c.UIPassword != LoadConfig().UIPassword {
		t.Fatal("generated password did not persist")
	}
	info, err := os.Stat(filepath.Join(c.DataDir, "ui_password.txt"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("password permissions", err)
	}
}
func TestSecuritySettingsWriteFailure(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	c := LoadConfig()
	old := c.UIPassword
	if err := os.Mkdir(filepath.Join(c.DataDir, "settings.json"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := c.UpdateSettings(SettingsDTO{UIPassword: "new"}); err == nil {
		t.Fatal("write failure hidden")
	}
	if c.UIPassword != old {
		t.Fatal("failed password change modified memory")
	}
}
func TestSecurityCorruptSettingsFailClosed(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	c := LoadConfig()
	os.WriteFile(filepath.Join(c.DataDir, "settings.json"), []byte("broken"), 0600)
	defer func() {
		if recover() == nil {
			t.Fatal("corrupt saved settings silently ignored")
		}
	}()
	LoadConfig()
}

func TestSecuritySavedCredentialsDoNotRebootstrap(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("UI_PASSWORD", "")
	c := LoadConfig()
	if err := c.UpdateSettings(SettingsDTO{UIPassword: "saved-new-password"}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(c.DataDir, "ui_password.txt")); err != nil {
		t.Fatal(err)
	}
	if LoadConfig().UIPassword != "saved-new-password" {
		t.Fatal("saved password not authoritative")
	}
	if _, err := os.Stat(filepath.Join(c.DataDir, "ui_password.txt")); !os.IsNotExist(err) {
		t.Fatal("rebootstrapped despite saved credentials")
	}
}
