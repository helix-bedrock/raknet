package message

import (
	"io"
	"sync"
)

// OpenConnectionRequest1 is the first offline handshake message, padded to probe the MTU.
type OpenConnectionRequest1 struct {
	ClientProtocol byte
	MTU            uint16
}

var cachedOCR1 sync.Map

// MarshalBinary encodes the message into its wire format.
func (pk *OpenConnectionRequest1) MarshalBinary() (data []byte, err error) {
	// Cache OpenConnectionRequest1 data: it only depends on the MTU and the
	// protocol, and is pretty big.
	key := uint32(pk.MTU)<<8 | uint32(pk.ClientProtocol)
	if b, ok := cachedOCR1.Load(key); ok {
		return b.([]byte), nil
	}
	b := make([]byte, pk.MTU-20-8) // IP Header: 20 bytes, UDP Header: 8 bytes.
	b[0] = IDOpenConnectionRequest1
	copy(b[1:], unconnectedMessageSequence[:])
	b[17] = pk.ClientProtocol

	cachedOCR1.Store(key, b)
	return b, nil
}

// UnmarshalBinary decodes the message from its wire format.
func (pk *OpenConnectionRequest1) UnmarshalBinary(data []byte) error {
	if len(data) < 17 {
		return io.ErrUnexpectedEOF
	}
	// Magic: 16 bytes.
	pk.ClientProtocol = data[16]
	pk.MTU = uint16(len(data) + 20 + 8 + 1) // Headers + packet ID.
	return nil
}
