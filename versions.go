package raknet

import (
	"net"
	"slices"
	"sync"
	"time"
)

// RakNet protocol 10 is used by Minecraft before 1.19.30 (for example 1.16 and
// 1.18 clients). Its handshake and frames are the same as protocol 11; only the
// version byte differs, so accepting it only needs the version recorded.

// acceptsProtocol reports whether the listener accepts RakNet protocol v.
func (conf ListenConfig) acceptsProtocol(v byte) bool {
	if len(conf.AcceptedProtocols) == 0 {
		return v == protocolVersion
	}
	return slices.Contains(conf.AcceptedProtocols, v)
}

// versionTable remembers the RakNet protocol a peer asked for in Open
// Connection Request 1 until its Request 2 arrives, which does not repeat it.
// Only protocols other than the current one are stored, entries expire, and
// the table is bounded, so spoofed requests cannot grow it without limit.
type versionTable struct {
	mu sync.Mutex
	m  map[string]versionEntry
}

type versionEntry struct {
	v  byte
	at time.Time
}

const (
	versionTableMax = 4096
	versionTTL      = 10 * time.Second
)

func newVersionTable() *versionTable {
	return &versionTable{m: map[string]versionEntry{}}
}

// put records that addr asked for protocol v.
func (t *versionTable) put(addr net.Addr, v byte, now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	key := addr.String()
	if v == protocolVersion {
		delete(t.m, key)
		return
	}
	if len(t.m) >= versionTableMax {
		for k, e := range t.m {
			if now.Sub(e.at) > versionTTL {
				delete(t.m, k)
			}
		}
		if len(t.m) >= versionTableMax {
			return
		}
	}
	t.m[key] = versionEntry{v: v, at: now}
}

// take returns and forgets the protocol addr asked for, or the current
// protocol if nothing (or only an expired entry) is known.
func (t *versionTable) take(addr net.Addr, now time.Time) byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	key := addr.String()
	e, ok := t.m[key]
	delete(t.m, key)
	if !ok || now.Sub(e.at) > versionTTL {
		return protocolVersion
	}
	return e.v
}
