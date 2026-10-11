package raknet

import (
	"log/slog"
	"net"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

// Verdict is returned by hooks that may stop what they observe.
type Verdict uint8

const (
	// Allow lets the data or action continue. It is the zero value.
	Allow Verdict = iota
	// Drop stops the data or action. Later hooks are not called.
	Drop
)

// RawEvent describes one UDP datagram, incoming or outgoing, before RakNet
// decodes it or after RakNet encodes it.
type RawEvent struct {
	// Addr is the remote address the datagram came from or is sent to.
	Addr net.Addr
	// Conn is the connection the datagram belongs to, or nil if no connection
	// exists for the address yet (offline messages, handshakes).
	Conn *Conn
	// Data is the datagram. It is only valid during the hook. A hook may
	// replace it with a different slice to change what is processed or sent;
	// it must not keep or modify the original slice after returning.
	Data []byte
}

// OfflineEvent describes an offline message: a datagram from an address that
// has no connection yet, such as a ping or a handshake request.
type OfflineEvent struct {
	// Addr is the address the message came from.
	Addr net.Addr
	// ID is the RakNet message ID, the first byte of Data.
	ID byte
	// Data is the full message, including ID. It is only valid during the
	// hook; a hook may replace it to change what is handled.
	Data []byte
}

// AttemptEvent describes a connection attempt that passed RakNet's own checks
// (cookie and GUID) and is about to be accepted.
type AttemptEvent struct {
	// Addr is the address of the peer trying to connect.
	Addr net.Addr
	// ClientGUID is the GUID the peer sent in Open Connection Request 2.
	ClientGUID int64
	// RequestedMTU is the MTU the peer asked for.
	RequestedMTU uint16
	// MTU is the MTU the listener will grant. A hook may lower it.
	MTU uint16
}

// PacketEvent describes one complete packet (after reassembly) sent or
// received over a connection: usually one Minecraft batch.
type PacketEvent struct {
	// Conn is the connection the packet belongs to.
	Conn *Conn
	// Data is the packet. It is only valid during the hook. A hook may replace
	// it to change what is delivered or sent; an empty replacement drops it.
	Data []byte
}

// FrameEvent describes one encapsulated frame read from an incoming datagram,
// before split frames are reassembled and before ordering.
type FrameEvent struct {
	// Conn is the connection the frame arrived on.
	Conn *Conn
	// Reliability is the RakNet reliability ID of the frame.
	Reliability byte
	// MessageIndex, SequenceIndex and OrderIndex are the frame's RakNet
	// indexes; they are only meaningful for reliabilities that use them.
	MessageIndex, SequenceIndex, OrderIndex uint32
	// Split reports whether the frame is one fragment of a larger packet.
	Split bool
	// SplitCount, SplitIndex and SplitID describe the fragment if Split is set.
	SplitCount, SplitIndex uint32
	SplitID                uint16
	// Payload is the frame's content. It is only valid during the hook; a hook
	// may replace it to change what is processed.
	Payload []byte
}

// hookList is a list of hooks that can be added to and removed from at any
// time. Reads are a single atomic load and cost nothing when the list is
// empty, so unused hooks add no overhead to the paths that check them.
type hookList[F any] struct {
	mu   sync.Mutex
	next uint64
	fns  atomic.Pointer[[]hookEntry[F]]
}

type hookEntry[F any] struct {
	id uint64
	fn F
}

// add registers fn and returns a function that removes it. The returned
// function is safe to call more than once and from any goroutine.
func (h *hookList[F]) add(fn F) (remove func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.next++
	id := h.next
	var cur []hookEntry[F]
	if p := h.fns.Load(); p != nil {
		cur = *p
	}
	updated := make([]hookEntry[F], len(cur), len(cur)+1)
	copy(updated, cur)
	updated = append(updated, hookEntry[F]{id: id, fn: fn})
	h.fns.Store(&updated)
	return sync.OnceFunc(func() { h.remove(id) })
}

func (h *hookList[F]) remove(id uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	p := h.fns.Load()
	if p == nil {
		return
	}
	updated := make([]hookEntry[F], 0, len(*p))
	for _, e := range *p {
		if e.id != id {
			updated = append(updated, e)
		}
	}
	if len(updated) == 0 {
		h.fns.Store(nil)
		return
	}
	h.fns.Store(&updated)
}

// get returns the current hooks, or nil if there are none.
func (h *hookList[F]) get() []hookEntry[F] {
	if p := h.fns.Load(); p != nil {
		return *p
	}
	return nil
}

// runVerdict calls each hook in each list in order until one returns Drop. A
// hook that panics is logged and treated as Allow, so a broken hook can never
// crash the server.
func runVerdict[E any](log *slog.Logger, e *E, lists ...[]hookEntry[func(*E) Verdict]) Verdict {
	for _, list := range lists {
		for _, h := range list {
			if callVerdict(log, h.fn, e) == Drop {
				return Drop
			}
		}
	}
	return Allow
}

func callVerdict[E any](log *slog.Logger, fn func(*E) Verdict, e *E) (v Verdict) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("raknet: hook panicked", "panic", r, "stack", string(debug.Stack()))
			v = Allow
		}
	}()
	return fn(e)
}

// runConnHooks calls hooks that only observe a connection. Panics are logged.
func runConnHooks(log *slog.Logger, c *Conn, lists ...[]hookEntry[func(*Conn)]) {
	for _, list := range lists {
		for _, h := range list {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Error("raknet: hook panicked", "panic", r, "stack", string(debug.Stack()))
					}
				}()
				h.fn(c)
			}()
		}
	}
}

// connHooks are hooks that can be set on a Listener (for every connection) or
// on a single Conn.
type connHooks struct {
	rawIn, rawOut       hookList[func(*RawEvent) Verdict]
	packetIn, packetOut hookList[func(*PacketEvent) Verdict]
	frameIn             hookList[func(*FrameEvent) Verdict]
	disconnect          hookList[func(*Conn)]
}

// listenerHooks are hooks only a Listener has: they fire before a connection
// exists.
type listenerHooks struct {
	connHooks
	offline hookList[func(*OfflineEvent) Verdict]
	attempt hookList[func(*AttemptEvent) Verdict]
	connect hookList[func(*Conn)]
}

// OnRawIn registers fn to be called for every datagram the listener receives,
// from any address, before it is decoded. Returning Drop discards it. It
// returns a function that removes the hook.
func (listener *Listener) OnRawIn(fn func(*RawEvent) Verdict) (remove func()) {
	return listener.hooks.rawIn.add(fn)
}

// OnRawOut registers fn to be called for every datagram the listener sends,
// including handshake replies, after it is encoded. Returning Drop discards it.
func (listener *Listener) OnRawOut(fn func(*RawEvent) Verdict) (remove func()) {
	return listener.hooks.rawOut.add(fn)
}

// OnOffline registers fn to be called for every offline message (pings and
// handshake requests from addresses without a connection). Returning Drop
// ignores the message.
func (listener *Listener) OnOffline(fn func(*OfflineEvent) Verdict) (remove func()) {
	return listener.hooks.offline.add(fn)
}

// OnConnectionAttempt registers fn to be called when a peer passes RakNet's
// handshake checks and is about to get a connection. Returning Drop rejects
// the attempt silently; a hook may also lower the granted MTU.
func (listener *Listener) OnConnectionAttempt(fn func(*AttemptEvent) Verdict) (remove func()) {
	return listener.hooks.attempt.add(fn)
}

// OnConnect registers fn to be called when a connection completes the RakNet
// handshake, just before Accept returns it.
func (listener *Listener) OnConnect(fn func(*Conn)) (remove func()) {
	return listener.hooks.connect.add(fn)
}

// OnDisconnect registers fn to be called when any connection of the listener
// closes.
func (listener *Listener) OnDisconnect(fn func(*Conn)) (remove func()) {
	return listener.hooks.disconnect.add(fn)
}

// OnPacketIn registers fn to be called for every complete packet received on
// any connection, before it reaches ReadPacket. Returning Drop discards it.
func (listener *Listener) OnPacketIn(fn func(*PacketEvent) Verdict) (remove func()) {
	return listener.hooks.packetIn.add(fn)
}

// OnPacketOut registers fn to be called for every packet written to any
// connection, before it is split and queued. Returning Drop discards it.
func (listener *Listener) OnPacketOut(fn func(*PacketEvent) Verdict) (remove func()) {
	return listener.hooks.packetOut.add(fn)
}

// OnFrameIn registers fn to be called for every frame read from an incoming
// datagram on any connection. Returning Drop discards the frame.
func (listener *Listener) OnFrameIn(fn func(*FrameEvent) Verdict) (remove func()) {
	return listener.hooks.frameIn.add(fn)
}

// OnRawIn registers fn for every datagram received on this connection only.
func (conn *Conn) OnRawIn(fn func(*RawEvent) Verdict) (remove func()) {
	return conn.hooks.rawIn.add(fn)
}

// OnRawOut registers fn for every datagram sent on this connection only.
func (conn *Conn) OnRawOut(fn func(*RawEvent) Verdict) (remove func()) {
	return conn.hooks.rawOut.add(fn)
}

// OnPacketIn registers fn for every complete packet received on this
// connection only.
func (conn *Conn) OnPacketIn(fn func(*PacketEvent) Verdict) (remove func()) {
	return conn.hooks.packetIn.add(fn)
}

// OnPacketOut registers fn for every packet written to this connection only.
func (conn *Conn) OnPacketOut(fn func(*PacketEvent) Verdict) (remove func()) {
	return conn.hooks.packetOut.add(fn)
}

// OnFrameIn registers fn for every frame received on this connection only.
func (conn *Conn) OnFrameIn(fn func(*FrameEvent) Verdict) (remove func()) {
	return conn.hooks.frameIn.add(fn)
}

// OnDisconnect registers fn to be called when this connection closes.
func (conn *Conn) OnDisconnect(fn func(*Conn)) (remove func()) {
	return conn.hooks.disconnect.add(fn)
}

// lhooks returns the hooks of the listener that owns conn, or nil for dialed
// connections.
func (conn *Conn) lhooks() *connHooks {
	if conn.listener == nil {
		return nil
	}
	return &conn.listener.hooks.connHooks
}

// runRaw runs raw hooks of the listener and the connection. It returns the
// (possibly replaced) data and false if a hook dropped it.
func (conn *Conn) runRaw(in bool, addr net.Addr, b []byte) ([]byte, bool) {
	var l, c []hookEntry[func(*RawEvent) Verdict]
	if lh := conn.lhooks(); lh != nil {
		if in {
			l = lh.rawIn.get()
		} else {
			l = lh.rawOut.get()
		}
	}
	if in {
		c = conn.hooks.rawIn.get()
	} else {
		c = conn.hooks.rawOut.get()
	}
	if l == nil && c == nil {
		return b, true
	}
	e := &RawEvent{Addr: addr, Conn: conn, Data: b}
	if runVerdict(conn.handler.log(), e, l, c) == Drop {
		return nil, false
	}
	return e.Data, true
}

// runPacket runs packet hooks of the listener and the connection.
func (conn *Conn) runPacket(in bool, b []byte) ([]byte, bool) {
	var l, c []hookEntry[func(*PacketEvent) Verdict]
	if lh := conn.lhooks(); lh != nil {
		if in {
			l = lh.packetIn.get()
		} else {
			l = lh.packetOut.get()
		}
	}
	if in {
		c = conn.hooks.packetIn.get()
	} else {
		c = conn.hooks.packetOut.get()
	}
	if l == nil && c == nil {
		return b, true
	}
	e := &PacketEvent{Conn: conn, Data: b}
	if runVerdict(conn.handler.log(), e, l, c) == Drop {
		return nil, false
	}
	return e.Data, true
}

// runFrameIn runs frame hooks for pk. It returns false if a hook dropped it,
// and applies a replaced payload to pk.
func (conn *Conn) runFrameIn(pk *packet) bool {
	var l []hookEntry[func(*FrameEvent) Verdict]
	if lh := conn.lhooks(); lh != nil {
		l = lh.frameIn.get()
	}
	c := conn.hooks.frameIn.get()
	if l == nil && c == nil {
		return true
	}
	e := &FrameEvent{
		Conn:          conn,
		Reliability:   byte(pk.reliability),
		MessageIndex:  uint32(pk.messageIndex),
		SequenceIndex: uint32(pk.sequenceIndex),
		OrderIndex:    uint32(pk.orderIndex),
		Split:         pk.split,
		SplitCount:    pk.splitCount,
		SplitIndex:    pk.splitIndex,
		SplitID:       pk.splitID,
		Payload:       pk.content,
	}
	if runVerdict(conn.handler.log(), e, l, c) == Drop {
		return false
	}
	pk.content = e.Payload
	return true
}

// runDisconnect runs the disconnect hooks of the connection and its listener.
func (conn *Conn) runDisconnect() {
	var l []hookEntry[func(*Conn)]
	if lh := conn.lhooks(); lh != nil {
		l = lh.disconnect.get()
	}
	c := conn.hooks.disconnect.get()
	if l == nil && c == nil {
		return
	}
	runConnHooks(conn.handler.log(), conn, l, c)
}

// writeOffline sends an offline reply through the listener's raw-out hooks.
func (listener *Listener) writeOffline(b []byte, addr net.Addr) error {
	if l := listener.hooks.rawOut.get(); l != nil {
		e := &RawEvent{Addr: addr, Data: b}
		if runVerdict(listener.conf.ErrorLog, e, l) == Drop {
			return nil
		}
		b = e.Data
	}
	_, err := listener.conn.WriteTo(b, addr)
	return err
}
