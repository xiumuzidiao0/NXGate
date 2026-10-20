package config

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestOptimizationGlobalBudgetAndSettings(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("LOCAL_PROXY_GLOBAL_MAX_CONNECTIONS", "2")
	c := LoadConfig()
	if !c.TryAcquireProxyConnection() || !c.TryAcquireProxyConnection() || c.TryAcquireProxyConnection() {
		t.Fatal("budget not enforced")
	}
	one := 1
	if e := c.UpdateSettings(SettingsDTO{ProxyGlobalMaxConnections: &one}); e != nil {
		t.Fatal(e)
	}
	if c.TryAcquireProxyConnection() {
		t.Fatal("lowering budget lost active leases")
	}
	c.ReleaseProxyConnection()
	if c.TryAcquireProxyConnection() {
		t.Fatal("existing at limit")
	}
	c.ReleaseProxyConnection()
	if !c.TryAcquireProxyConnection() {
		t.Fatal("permit leaked")
	}
	c.ReleaseProxyConnection()
	if a, l, r := c.ProxyConnectionStats(); a != 0 || l != 1 || r != 3 {
		t.Fatal(a, l, r)
	}
	if l := LoadConfig().GetSettings().ProxyGlobalMaxConnections; l == nil || *l != 1 {
		t.Fatal("restart lost limit")
	}
	bad := 0
	if e := c.UpdateSettings(SettingsDTO{ProxyGlobalMaxConnections: &bad}); e == nil {
		t.Fatal("invalid limit accepted")
	}
	if e := c.UpdateSettings(SettingsDTO{}); e != nil {
		t.Fatal(e)
	}
	if *c.GetSettings().ProxyGlobalMaxConnections != 1 {
		t.Fatal("old client erased limit")
	}
}
func TestOptimizationGlobalBudgetConcurrent(t *testing.T) {
	c := &Config{ProxyGlobalMaxConnections: 4}
	var active atomic.Int32
	var wg sync.WaitGroup
	for w := 0; w < 32; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				if c.TryAcquireProxyConnection() {
					if n := active.Add(1); n > 4 {
						t.Errorf("over limit: %d", n)
					}
					active.Add(-1)
					c.ReleaseProxyConnection()
				}
			}
		}()
	}
	wg.Wait()
	if a, _, _ := c.ProxyConnectionStats(); a != 0 {
		t.Fatal("leaked lease", a)
	}
}
