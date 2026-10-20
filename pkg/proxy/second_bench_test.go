package proxy

import (
	"aimili-vpngate-go/pkg/tunnel"
	"fmt"
	"testing"
)

type optimizationFixedPool struct{ list []*tunnel.Tunnel }

func (p optimizationFixedPool) GetHealthyTunnels(_ []string) []*tunnel.Tunnel { return p.list }

var optimizationSink *tunnel.Tunnel

func BenchmarkSecondSelect(b *testing.B) {
	for _, n := range []int{8, 64} {
		for _, policy := range []PortPolicy{PolicyRoundRobin, PolicyRandom, PolicyLeastRTT} {
			b.Run(fmt.Sprintf("%d/%s", n, policy), func(b *testing.B) {
				p := optimizationFixedPool{}
				for i := 0; i < n; i++ {
					p.list = append(p.list, &tunnel.Tunnel{ID: fmt.Sprint(i), Status: tunnel.StatusConnected, LatencyMs: i + 1})
				}
				s := &DefaultScheduler{pool: p}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					optimizationSink = s.SelectTunnel(7928, nil, nil, policy, 300)
				}
			})
		}
	}
}
