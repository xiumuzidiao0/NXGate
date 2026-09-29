package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aimili-vpngate-go/pkg/config"
	"aimili-vpngate-go/pkg/nodes"
	"aimili-vpngate-go/pkg/proxy"
	"aimili-vpngate-go/pkg/tunnel"
)

func TestBackupExportAndImport(t *testing.T) {
	tempDir1 := t.TempDir()
	cfg1 := &config.Config{
		DataDir:   tempDir1,
		ProxyPort: 7928,
		UIHost:    "127.0.0.1",
		UIPort:    8787,
		UIPath:    "admin",
	}

	np1 := nodes.NewNodePool(cfg1)
	dm1 := tunnel.NewDynamicGroupManager(cfg1, nil, np1)
	pm1 := proxy.NewMultiPortManager(cfg1, nil, dm1)

	// Add test data
	np1.Favorites().Add("fav-node-101")
	np1.Blacklist().MarkManualWithOptions("perm-node-202", "1.2.3.4", "JP", "Manual block test", 0, "node", true)

	_ = dm1.SaveGroup(&tunnel.DynamicGroup{
		ID:             "dg-backup-test",
		Name:           "备份测试组",
		Country:        "JP",
		TargetCount:    2,
		FallbackPolicy: "favorites",
	})

	_ = pm1.ApplyRules([]proxy.PortRule{
		{
			Port:           1099,
			Enabled:        true,
			BoundGroupIDs:  []string{"dg-backup-test"},
			Policy:         proxy.PolicyRoundRobin,
			AuthMode:       "none",
		},
	})

	s1 := &Server{
		cfg:        cfg1,
		pool:       np1,
		dynamicMgr: dm1,
		portMgr:    pm1,
	}

	// 1. Export Backup
	reqExp := httptest.NewRequest("GET", "/api/system/backup/export", nil)
	wExp := httptest.NewRecorder()
	s1.handleExportBackup(wExp, reqExp)

	if wExp.Code != http.StatusOK {
		t.Fatalf("expected export 200, got %d, body: %s", wExp.Code, wExp.Body.String())
	}

	var backup GatewayBackupPackage
	if err := json.Unmarshal(wExp.Body.Bytes(), &backup); err != nil {
		t.Fatalf("failed to decode exported backup json: %v", err)
	}

	if len(backup.Favorites) != 1 || backup.Favorites[0] != "fav-node-101" {
		t.Errorf("exported favorites mismatch: %+v", backup.Favorites)
	}
	if len(backup.Blacklist) != 1 || backup.Blacklist[0].ID != "perm-node-202" {
		t.Errorf("exported blacklist mismatch: %+v", backup.Blacklist)
	}
	if len(backup.DynamicGroups) < 1 {
		t.Errorf("exported dynamic groups empty")
	}
	if len(backup.PortRules) != 1 || backup.PortRules[0].Port != 1099 {
		t.Errorf("exported port rules mismatch: %+v", backup.PortRules)
	}

	// 2. Import into a fresh Server
	pm1.StopAll()
	tempDir2 := t.TempDir()
	cfg2 := &config.Config{
		DataDir:   tempDir2,
		ProxyPort: 7928,
		UIHost:    "127.0.0.1",
		UIPort:    8787,
	}

	np2 := nodes.NewNodePool(cfg2)
	dm2 := tunnel.NewDynamicGroupManager(cfg2, nil, np2)
	pm2 := proxy.NewMultiPortManager(cfg2, nil, dm2)

	s2 := &Server{
		cfg:        cfg2,
		pool:       np2,
		dynamicMgr: dm2,
		portMgr:    pm2,
	}

	backupBytes, _ := json.Marshal(backup)
	reqImp := httptest.NewRequest("POST", "/api/system/backup/import", bytes.NewReader(backupBytes))
	wImp := httptest.NewRecorder()
	s2.handleImportBackup(wImp, reqImp)

	if wImp.Code != http.StatusOK {
		t.Fatalf("expected import 200, got %d, body: %s", wImp.Code, wImp.Body.String())
	}

	// Verify restored items in Server 2
	if !np2.Favorites().IsFavorite("fav-node-101") {
		t.Errorf("restored favorites missing fav-node-101")
	}
	if !np2.Blacklist().IsBlacklisted("perm-node-202") {
		t.Errorf("restored blacklist missing perm-node-202")
	}
	restoredGroup := dm2.GetGroup("dg-backup-test")
	if restoredGroup == nil || restoredGroup.FallbackPolicy != "favorites" {
		t.Errorf("restored dynamic group mismatch: %+v", restoredGroup)
	}
	restoredRule := pm2.GetRule(1099)
	if restoredRule == nil || !restoredRule.Enabled {
		t.Errorf("restored port rule 1099 missing or disabled: %+v", restoredRule)
	}
	pm2.StopAll()
}
