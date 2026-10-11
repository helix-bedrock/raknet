package message

import (
	"encoding/binary"
	"io"
)

// ConnectedPing is sent over an established connection to measure latency and keep it alive.
type ConnectedPing struct {
	// PingTime represents the duration in milliseconds since the system or client was started.
	PingTime int64
}

// UnmarshalBinary decodes the message from its wire format.
func (pk *ConnectedPing) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return io.ErrUnexpectedEOF
	}
	pk.PingTime = int64(binary.BigEndian.Uint64(data))
	return nil
}

// MarshalBinary encodes the message into its wire format.
func (pk *ConnectedPing) MarshalBinary() (data []byte, err error) {
	b := make([]byte, 9)
	b[0] = IDConnectedPing
	binary.BigEndian.PutUint64(b[1:], uint64(pk.PingTime))
	return b, nil
}
