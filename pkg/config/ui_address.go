package config

import (
	"fmt"
	"net"
	"strings"
)

// WebAccessURL distinguishes wildcard listeners from an address clients can use.
// No outbound IP-discovery request is required during application startup.
func WebAccessURL(host string, port int, path string) string {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "<SERVER-IP>"
	}
	return fmt.Sprintf("http://%s/%s/", net.JoinHostPort(host, fmt.Sprint(port)), strings.Trim(path, "/"))
}
