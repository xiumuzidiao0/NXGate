package proxy

import (
	"bufio"
	"io"
	"net"
	"sync"
	"time"

	"aimili-vpngate-go/pkg/stats"
)

type bufferedConn struct {
	net.Conn
	br *bufio.Reader
}

func (b *bufferedConn) Read(p []byte) (int, error) {
	return b.br.Read(p)
}

func (b *bufferedConn) CloseWrite() error {
	if tcp, ok := b.Conn.(*net.TCPConn); ok {
		return tcp.CloseWrite()
	}
	return b.Conn.Close()
}

type countingReader struct {
	reader io.Reader
	onRead func(n int)
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.reader.Read(p)
	if n > 0 && c.onRead != nil {
		c.onRead(n)
	}
	return n, err
}

func closeWrite(c net.Conn) {
	if bc, ok := c.(*bufferedConn); ok {
		_ = bc.CloseWrite()
		return
	}
	if tcp, ok := c.(*net.TCPConn); ok {
		_ = tcp.CloseWrite()
		return
	}
	_ = c.Close()
}

func clearDeadline(conn net.Conn) {
	_ = conn.SetDeadline(time.Time{})
}

var relayBufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, 32*1024)
		return &b
	},
}

func relay(client, upstream net.Conn) {
	tracker := stats.GetTrafficTracker()
	var wg sync.WaitGroup
	wg.Add(2)

	// Upstream -> Client (Download)
	go func() {
		defer wg.Done()
		defer closeWrite(client)

		cr := &countingReader{
			reader: upstream,
			onRead: func(n int) {
				if n > 0 {
					tracker.AddDownload(uint64(n))
				}
			},
		}
		bufPtr := relayBufferPool.Get().(*[]byte)
		defer relayBufferPool.Put(bufPtr)
		_, _ = io.CopyBuffer(client, cr, *bufPtr)
	}()

	// Client -> Upstream (Upload)
	go func() {
		defer wg.Done()
		defer closeWrite(upstream)

		cr := &countingReader{
			reader: client,
			onRead: func(n int) {
				if n > 0 {
					tracker.AddUpload(uint64(n))
				}
			},
		}
		bufPtr := relayBufferPool.Get().(*[]byte)
		defer relayBufferPool.Put(bufPtr)
		_, _ = io.CopyBuffer(upstream, cr, *bufPtr)
	}()

	wg.Wait()
	_ = client.Close()
	_ = upstream.Close()
}
