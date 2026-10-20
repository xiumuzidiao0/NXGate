package tunnel

import (
	"aimili-vpngate-go/pkg/config"
	"os"
	"testing"
)

func TestSecurityGroupPersistenceFailure(t *testing.T) {
	m := NewDynamicGroupManager(&config.Config{DataDir: t.TempDir()}, nil, nil)
	g := &DynamicGroup{ID: "test", Name: "original"}
	if err := m.SaveGroup(g); err != nil {
		t.Fatal(err)
	}
	os.Remove(m.filePath)
	if err := os.Mkdir(m.filePath, 0700); err != nil {
		t.Fatal(err)
	}
	if err := m.SaveGroup(&DynamicGroup{ID: "test", Name: "changed"}); err == nil {
		t.Fatal("persistence failure hidden")
	}
	if got := m.GetGroup("test"); got.Name != "original" {
		t.Fatal("failed write changed live group")
	}
	if err := m.SaveGroup(&DynamicGroup{ID: "new"}); err == nil {
		t.Fatal("new group failure hidden")
	}
	if m.GetGroup("new") != nil {
		t.Fatal("failed new group retained")
	}
}
