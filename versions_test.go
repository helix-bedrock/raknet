package raknet_test

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"github.com/helix-bedrock/raknet"
)

// dialV listens with conf, dials with protocol v and returns both ends, or the
// dial error.
func dialV(t *testing.T, conf raknet.ListenConfig, v byte) (*raknet.Conn, *raknet.Conn, error) {
	t.Helper()
	l, err := conf.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })
	var gotV byte
	l.OnConnectionAttempt(func(e *raknet.AttemptEvent) raknet.Verdict { gotV = e.Protocol; return raknet.Allow })
	accepted := make(chan net.Conn, 1)
	go func() {
		if c, err := l.Accept(); err == nil {
			accepted <- c
		}
	}()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	client, err := raknet.Dialer{Protocol: v}.DialContext(ctx, l.Addr().String())
	if err != nil {
		return nil, nil, err
	}
	t.Cleanup(func() { _ = client.Close() })
	select {
	case c := <-accepted:
		server := c.(*raknet.Conn)
		t.Cleanup(func() { _ = server.Close() })
		if gotV != server.ProtocolVersion() {
			t.Fatalf("attempt hook saw protocol %d, conn reports %d", gotV, server.ProtocolVersion())
		}
		return client, server, nil
	case <-time.After(5 * time.Second):
		t.Fatal("accept timed out")
	}
	return nil, nil, nil
}

func TestProtocol10RejectedByDefault(t *testing.T) {
	if _, _, err := dialV(t, raknet.ListenConfig{}, 10); err == nil {
		t.Fatal("protocol 10 client connected to a listener that only accepts 11")
	}
}

func TestProtocol10Accepted(t *testing.T) {
	conf := raknet.ListenConfig{AcceptedProtocols: []byte{10, 11}}
	for _, v := range []byte{10, 11} {
		client, server, err := dialV(t, conf, v)
		if err != nil {
			t.Fatalf("protocol %d: dial: %v", v, err)
		}
		if server.ProtocolVersion() != v || client.ProtocolVersion() != v {
			t.Fatalf("protocol %d: server reports %d, client reports %d", v, server.ProtocolVersion(), client.ProtocolVersion())
		}
		if _, err := client.Write([]byte("hello")); err != nil {
			t.Fatal(err)
		}
		b, err := server.ReadPacket()
		if err != nil || !bytes.Equal(b, []byte("hello")) {
			t.Fatalf("protocol %d: read %q, %v", v, b, err)
		}
	}
}

func TestOnlyProtocol10(t *testing.T) {
	if _, _, err := dialV(t, raknet.ListenConfig{AcceptedProtocols: []byte{10}}, 11); err == nil {
		t.Fatal("protocol 11 client connected to a listener that only accepts 10")
	}
}
