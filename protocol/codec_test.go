package protocol

import (
	"bytes"
	"testing"
)

func TestEncoderRoundTripAtHeaderBoundaries(t *testing.T) {
	for _, size := range []int{0, 1, 0x3f, 0x40, 0x3fff, 0x4000, 0x3fffff, 0x400000} {
		src := bytes.Repeat([]byte{0xa5}, size)
		encoded, err := Encoder(src)
		if err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
		payloadLength, frameLength, err := DecodeHead(encoded)
		if err != nil {
			t.Fatalf("size %d header: %v", size, err)
		}
		if int(payloadLength) != size || int(frameLength) != len(encoded) {
			t.Fatalf("size %d: payload=%d frame=%d encoded=%d", size, payloadLength, frameLength, len(encoded))
		}
		decoded, err := Decode(encoded)
		if err != nil || !bytes.Equal(decoded, src) {
			t.Fatalf("size %d did not round trip", size)
		}
	}
}
