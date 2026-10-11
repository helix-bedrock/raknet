package raknet

import (
	"net"
	"testing"
	"time"
)

func TestVersionTable(t *testing.T) {
	tab := newVersionTable()
	now := time.Now()
	a := &net.UDPAddr{IP: net.IPv4(1, 2, 3, 4), Port: 1}
	if v := tab.take(a, now); v != protocolVersion {
		t.Fatalf("unknown addr: %d", v)
	}
	tab.put(a, 10, now)
	if v := tab.take(a, now); v != 10 {
		t.Fatalf("take = %d, want 10", v)
	}
	if v := tab.take(a, now); v != protocolVersion {
		t.Fatal("take did not forget the entry")
	}
	tab.put(a, 10, now)
	tab.put(a, 11, now) // a retry with the current version wins
	if v := tab.take(a, now); v != protocolVersion {
		t.Fatalf("take after retry = %d", v)
	}
	tab.put(a, 10, now)
	if v := tab.take(a, now.Add(versionTTL+time.Second)); v != protocolVersion {
		t.Fatal("expired entry was used")
	}
}

func TestVersionTableBounded(t *testing.T) {
	tab := newVersionTable()
	now := time.Now()
	for i := range versionTableMax + 100 {
		tab.put(&net.UDPAddr{IP: net.IPv4(10, byte(i>>16), byte(i>>8), byte(i)), Port: 1}, 10, now)
	}
	if n := len(tab.m); n > versionTableMax {
		t.Fatalf("table grew to %d entries", n)
	}
	// Once entries expire, new ones fit again.
	late := now.Add(versionTTL + time.Second)
	b := &net.UDPAddr{IP: net.IPv4(9, 9, 9, 9), Port: 1}
	tab.put(b, 10, late)
	if v := tab.take(b, late); v != 10 {
		t.Fatalf("new entry after expiry = %d", v)
	}
}
