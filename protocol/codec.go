package protocol

import (
	"encoding/binary"
	"errors"
)

func DecodeHead(src []byte) (uint, uint, error) {
	if src == nil || len(src) == 0 {
		return 0, 0, errors.New("nil")
	}
	headLen := uint((src[0] & 0x3) + 1)
	if uint(len(src)) < headLen {
		return 0, 0, errors.New("dataLen<headLen")
	}
	n := uint(src[0])
	if headLen > 1 {
		n |= uint(src[1]) << 8
	}
	if headLen > 2 {
		n |= uint(src[2]) << 16
	}
	if headLen > 3 {
		n |= uint(src[3]) << 24
	}
	n >>= 2
	realLength := n
	if n <= 0x3F {
		realLength += 1
	} else if n <= 0x3FFF {
		realLength += 2
	} else if n <= 0x3FFFFF {
		realLength += 3
	} else if n <= 0x3FFFFFFF {
		realLength += 4
	}
	return n, realLength, nil
}

func Decode(src []byte) (data []byte, err error) {
	if src == nil {
		return
	}
	headLen := uint((src[0] & 0x3) + 1)
	if uint(len(src)) < headLen {
		err = errors.New("dataLen<headLen")
		return
	}
	data = src[headLen:]
	return
}

func Encoder(src []byte) (data []byte, err error) {
	if src == nil {
		return
	}
	l := len(src)
	var headerLength int
	if l <= 0x3F {
		headerLength = 1
	} else if l <= 0x3FFF {
		headerLength = 2
	} else if l <= 0x3FFFFF {
		headerLength = 3
	} else if l <= 0x3FFFFFFF {
		headerLength = 4
	} else {
		return nil, errors.New("overflow")
	}
	data = make([]byte, headerLength+l)
	value := uint32(l<<2) | uint32(headerLength-1)
	switch headerLength {
	case 1:
		data[0] = byte(value)
	case 2:
		binary.LittleEndian.PutUint16(data, uint16(value))
	case 3:
		data[0] = byte(value)
		data[1] = byte(value >> 8)
		data[2] = byte(value >> 16)
	case 4:
		binary.LittleEndian.PutUint32(data, value)
	}
	copy(data[headerLength:], src)
	return data, nil
}
