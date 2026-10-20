package tunnel

import (
	"fmt"
	"testing"
)

func BenchmarkSecondHealthyPool(b *testing.B) {
	for _, n := range []int{8, 64} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			p := &Pool{tunnels: map[string]*Tunnel{}}
			for i := 0; i < n; i++ {
				id := fmt.Sprintf("t-%d", i)
				p.tunnels[id] = &Tunnel{ID: id, Status: StatusConnected}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = p.GetHealthyTunnels(nil)
			}
		})
	}
}
