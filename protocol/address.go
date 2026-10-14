package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"lukechampine.com/uint128"
	"math"
	"net"
	"strconv"
	"strings"
	"time"
)

func EncodeAddr(addr string) []byte {
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return nil
	}
	port, _ := strconv.ParseUint(portText, 10, 32)
	ipAddr := net.ParseIP(host)
	if ipAddr == nil {
		return nil
	}

	if ipv4 := ipAddr.To4(); ipv4 != nil && !strings.Contains(host, ":") {
		timestamp := uint128.From64(uint64(uint32(time.Now().UnixMicro())))
		var rawIP uint32
		_ = binary.Read(bytes.NewBuffer(ipv4), binary.LittleEndian, &rawIP)
		encoded := uint128.From64(uint64(rawIP)).Add(timestamp).Lsh(49).
			Or(timestamp.Lsh(17)).
			Or(timestamp.And(uint128.From64(0xFFFF)).Add(uint128.From64(port)))

		result := make([]byte, 16)
		encoded.PutBytes(result)
		for len(result) > 0 && result[len(result)-1] == 0 {
			result = result[:len(result)-1]
		}
		return result
	}

	ipv6 := ipAddr.To16()
	if ipv6 == nil {
		return nil
	}
	result := make([]byte, 18)
	copy(result, ipv6)
	binary.LittleEndian.PutUint16(result[16:], uint16(port))
	return result
}

func DecodeAddr(data []byte) string {
	if len(data) == 18 {
		ip := net.IP(append([]byte(nil), data[:16]...))
		port := binary.LittleEndian.Uint16(data[16:])
		return net.JoinHostPort(ip.String(), strconv.Itoa(int(port)))
	}

	padded := make([]byte, 16)
	copy(padded, data)
	number := uint128.FromBytes(padded)
	timestamp := number.Rsh(17).And(uint128.From64(math.MaxUint32))
	ip := number.Rsh(49).Sub(timestamp)
	port := uint16(number.And(uint128.From64(0xFFFFFF)).Sub(timestamp.And(uint128.From64(0xFFFF))).Lo)

	ipBytes := make([]byte, 16)
	ip.PutBytes(ipBytes)
	return fmt.Sprintf("%d.%d.%d.%d:%d", ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3], port)
}
