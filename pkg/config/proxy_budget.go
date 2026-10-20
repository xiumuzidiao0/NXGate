package config

const DefaultProxyGlobalMaxConnections = 1024
const MaxProxyGlobalMaxConnections = 65536

// One budget lives on the shared Config, not on each port or listener generation.
// Updating the limit doesn't reset active leases. No queue of accepted idle sockets.
func (c *Config) TryAcquireProxyConnection() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	limit := c.ProxyGlobalMaxConnections
	if limit <= 0 {
		limit = DefaultProxyGlobalMaxConnections
	}
	if c.proxyActive >= limit {
		c.proxyRejected++
		return false
	}
	c.proxyActive++
	return true
}
func (c *Config) ReleaseProxyConnection() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.proxyActive > 0 {
		c.proxyActive--
	}
}
func (c *Config) ProxyConnectionStats() (active, limit int, rejected uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	limit = c.ProxyGlobalMaxConnections
	if limit <= 0 {
		limit = DefaultProxyGlobalMaxConnections
	}
	return c.proxyActive, limit, c.proxyRejected
}
