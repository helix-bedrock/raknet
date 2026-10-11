package raknet

import "testing"

// FuzzPacketRead feeds arbitrary bytes to the encapsulated packet decoder,
// which parses data from untrusted peers. It must never panic.
func FuzzPacketRead(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0x60, 0, 8, 0, 0, 0, 0, 0, 0, 0, 1})
	f.Add([]byte{0x70, 0xff, 0xff})
	f.Fuzz(func(t *testing.T, data []byte) {
		pk := &packet{}
		_, _ = pk.read(data)
	})
}

// FuzzAcknowledgementRead feeds arbitrary bytes to the ACK/NACK decoder.
func FuzzAcknowledgementRead(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0, 1, 1, 0, 0, 0})
	f.Add([]byte{0, 2, 0, 0, 0, 0, 0, 0, 0})
	f.Fuzz(func(t *testing.T, data []byte) {
		ack := &acknowledgement{}
		_ = ack.read(data)
	})
}
