package message

import "testing"

// decoders lists every message decoder. Each one must return an error for bad
// input, never panic: these messages arrive from untrusted clients.
var decoders = map[string]func() interface{ UnmarshalBinary([]byte) error }{
	"ConnectedPing":               func() interface{ UnmarshalBinary([]byte) error } { return &ConnectedPing{} },
	"ConnectedPong":               func() interface{ UnmarshalBinary([]byte) error } { return &ConnectedPong{} },
	"ConnectionRequest":           func() interface{ UnmarshalBinary([]byte) error } { return &ConnectionRequest{} },
	"IncompatibleProtocolVersion": func() interface{ UnmarshalBinary([]byte) error } { return &IncompatibleProtocolVersion{} },
	"NewIncomingConnection":       func() interface{ UnmarshalBinary([]byte) error } { return &NewIncomingConnection{} },
	"OpenConnectionReply1":        func() interface{ UnmarshalBinary([]byte) error } { return &OpenConnectionReply1{} },
	"OpenConnectionReply2":        func() interface{ UnmarshalBinary([]byte) error } { return &OpenConnectionReply2{} },
	"OpenConnectionRequest1":      func() interface{ UnmarshalBinary([]byte) error } { return &OpenConnectionRequest1{} },
	"OpenConnectionRequest2":      func() interface{ UnmarshalBinary([]byte) error } { return &OpenConnectionRequest2{} },
	"UnconnectedPing":             func() interface{ UnmarshalBinary([]byte) error } { return &UnconnectedPing{} },
	"UnconnectedPong":             func() interface{ UnmarshalBinary([]byte) error } { return &UnconnectedPong{} },
}

// FuzzMessages feeds arbitrary bytes to every decoder. The first byte picks
// the decoder so one fuzz target covers them all.
func FuzzMessages(f *testing.F) {
	names := make([]string, 0, len(decoders))
	for name := range decoders {
		names = append(names, name)
	}
	for i := range names {
		f.Add(byte(i), []byte{})
		f.Add(byte(i), []byte{0, 0, 0, 0})
		f.Add(byte(i), make([]byte, 64))
	}
	f.Fuzz(func(t *testing.T, which byte, data []byte) {
		for _, name := range names {
			if which == 0 {
				_ = decoders[name]().UnmarshalBinary(data)
				return
			}
			which--
		}
		_ = decoders[names[int(which)%len(names)]]().UnmarshalBinary(data)
	})
}

// FuzzAddr checks the address decoder used inside several messages.
func FuzzAddr(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{4, 1, 2, 3, 4, 0, 0})
	f.Add([]byte{6})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = addr(data)
		_ = addrSize(data)
	})
}
