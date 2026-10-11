package raknet_test

import (
	"bytes"
	"sync/atomic"
	"testing"
	"time"

	"github.com/helix-bedrock/raknet"
)

// pair starts a listener with setup applied, dials it and returns both ends.
func pair(t *testing.T, setup func(l *raknet.Listener)) (*raknet.Listener, *raknet.Conn, *raknet.Conn) {
	t.Helper()
	l, err := raknet.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })
	if setup != nil {
		setup(l)
	}
	accepted := make(chan *raknet.Conn, 1)
	go func() {
		c, err := l.Accept()
		if err != nil {
			close(accepted)
			return
		}
		accepted <- c.(*raknet.Conn)
	}()
	client, err := raknet.DialTimeout(l.Addr().String(), 5*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	select {
	case server, ok := <-accepted:
		if !ok {
			t.Fatal("accept failed")
		}
		t.Cleanup(func() { _ = server.Close() })
		return l, server, client
	case <-time.After(5 * time.Second):
		t.Fatal("accept timed out")
	}
	return nil, nil, nil
}

// packets starts one goroutine that reads every packet from c into the
// returned channel, so a timed-out read never steals a later packet.
func packets(c *raknet.Conn) <-chan []byte {
	ch := make(chan []byte, 16)
	go func() {
		defer close(ch)
		for {
			b, err := c.ReadPacket()
			if err != nil {
				return
			}
			ch <- b
		}
	}()
	return ch
}

// within returns the next packet from ch, or nil if none arrives in d.
func within(ch <-chan []byte, d time.Duration) []byte {
	select {
	case b := <-ch:
		return b
	case <-time.After(d):
		return nil
	}
}

func TestListenerHooksObserveHandshake(t *testing.T) {
	var rawIn, offline, frames, connects atomic.Int32
	var guid atomic.Int64
	_, server, client := pair(t, func(l *raknet.Listener) {
		l.OnRawIn(func(*raknet.RawEvent) raknet.Verdict { rawIn.Add(1); return raknet.Allow })
		l.OnOffline(func(*raknet.OfflineEvent) raknet.Verdict { offline.Add(1); return raknet.Allow })
		l.OnConnectionAttempt(func(e *raknet.AttemptEvent) raknet.Verdict { guid.Store(e.ClientGUID); return raknet.Allow })
		l.OnConnect(func(*raknet.Conn) { connects.Add(1) })
		l.OnFrameIn(func(*raknet.FrameEvent) raknet.Verdict { frames.Add(1); return raknet.Allow })
	})
	if _, err := client.Write([]byte("hello")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if b := within(packets(server), 5*time.Second); !bytes.Equal(b, []byte("hello")) {
		t.Fatalf("server read %q, want hello", b)
	}
	if rawIn.Load() == 0 || offline.Load() == 0 || frames.Load() == 0 {
		t.Fatalf("hooks not called: rawIn=%d offline=%d frames=%d", rawIn.Load(), offline.Load(), frames.Load())
	}
	if connects.Load() != 1 {
		t.Fatalf("connect hook called %d times, want 1", connects.Load())
	}
	if guid.Load() >= 0 {
		t.Fatalf("attempt hook saw GUID %d, want a negative client GUID", guid.Load())
	}
}

func TestPacketInHookCanReplaceData(t *testing.T) {
	_, server, client := pair(t, func(l *raknet.Listener) {
		l.OnPacketIn(func(e *raknet.PacketEvent) raknet.Verdict {
			if bytes.Equal(e.Data, []byte("original")) {
				e.Data = []byte("changed")
			}
			return raknet.Allow
		})
	})
	if _, err := client.Write([]byte("original")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if b := within(packets(server), 5*time.Second); !bytes.Equal(b, []byte("changed")) {
		t.Fatalf("server read %q, want changed", b)
	}
}

func TestConnPacketOutHookDropsAndRemoves(t *testing.T) {
	_, server, client := pair(t, nil)
	remove := server.OnPacketOut(func(*raknet.PacketEvent) raknet.Verdict { return raknet.Drop })
	if _, err := server.Write([]byte("dropped")); err != nil {
		t.Fatalf("write: %v", err)
	}
	in := packets(client)
	if b := within(in, 300*time.Millisecond); b != nil {
		t.Fatalf("client read %q, want nothing while the hook drops packets", b)
	}
	remove()
	remove() // removing twice is safe
	if _, err := server.Write([]byte("sent")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if b := within(in, 5*time.Second); !bytes.Equal(b, []byte("sent")) {
		t.Fatalf("client read %q, want sent after removing the hook", b)
	}
}

func TestAttemptHookCanReject(t *testing.T) {
	l, err := raknet.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })
	l.OnConnectionAttempt(func(*raknet.AttemptEvent) raknet.Verdict { return raknet.Drop })
	if c, err := raknet.DialTimeout(l.Addr().String(), time.Second); err == nil {
		_ = c.Close()
		t.Fatal("dial succeeded, want it rejected by the attempt hook")
	}
}

func TestPanickingHookDoesNotCrash(t *testing.T) {
	_, server, client := pair(t, func(l *raknet.Listener) {
		l.OnPacketIn(func(*raknet.PacketEvent) raknet.Verdict { panic("broken hook") })
	})
	if _, err := client.Write([]byte("still works")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if b := within(packets(server), 5*time.Second); !bytes.Equal(b, []byte("still works")) {
		t.Fatalf("server read %q, want the packet despite the panicking hook", b)
	}
}

func TestDisconnectHooks(t *testing.T) {
	var listenerSaw, connSaw atomic.Int32
	_, server, client := pair(t, func(l *raknet.Listener) {
		l.OnDisconnect(func(*raknet.Conn) { listenerSaw.Add(1) })
	})
	server.OnDisconnect(func(*raknet.Conn) { connSaw.Add(1) })
	_ = client.Close()
	_ = server.Close()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && (listenerSaw.Load() == 0 || connSaw.Load() == 0) {
		time.Sleep(10 * time.Millisecond)
	}
	if listenerSaw.Load() != 1 || connSaw.Load() != 1 {
		t.Fatalf("disconnect hooks: listener=%d conn=%d, want 1 and 1", listenerSaw.Load(), connSaw.Load())
	}
}
