package transport

import (
	"net"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
)

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func InSameSubnet(ip1Str, ip2Str, maskStr string) bool {
	ip1 := net.ParseIP(ip1Str)
	ip2 := net.ParseIP(ip2Str)
	mask := net.IPMask(net.ParseIP(maskStr).To4())
	if ip1 == nil || ip2 == nil {
		return false
	}
	return ip1.Mask(mask).Equal(ip2.Mask(mask))
}

func InSubnet(ip string) bool {
	if !(strings.HasPrefix(ip, "192.168.") || strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "172.")) {
		return false
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	for _, iface := range interfaces {
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				localIP := ipNet.IP.String()
				if strings.HasPrefix(localIP, "192.168.") || strings.HasPrefix(localIP, "10.") || strings.HasPrefix(localIP, "172.") {
					if InSameSubnet(localIP, ip, net.IP(ipNet.Mask).String()) {
						return true
					}
				}
			}
		}
	}
	return false
}

func GetHostPort(addr string) (string, uint64) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0
	}
	p, _ := strconv.ParseUint(portStr, 10, 32)
	return host, p
}

// RelayServerAddress returns the configured relay endpoint, or derives the
// interface address that can reach the other peer. Client-provided relay
// addresses are deliberately ignored because a local client may report
// localhost, which is unusable by its remote peer.
var relayRotation atomic.Uint64

func RelayServerAddress(writer *Writer, otherPeerIP string) string {
	if len(Config.RelayServers) > 0 {
		return normalizeRelayAddress(Config.RelayServers[(relayRotation.Add(1)-1)%uint64(len(Config.RelayServers))])
	}
	if writer != nil {
		if host, _, err := net.SplitHostPort(writer.GetLocalAddrStr()); err == nil {
			if ip := net.ParseIP(strings.Trim(host, "[]")); usableRelayIP(ip) {
				return net.JoinHostPort(ip.String(), strconv.Itoa(PortRelay))
			}
		}
	}
	if ip := routedLocalIP(otherPeerIP); usableRelayIP(ip) {
		return net.JoinHostPort(ip.String(), strconv.Itoa(PortRelay))
	}
	return ""
}

func normalizeRelayAddress(value string) string {
	if _, _, err := net.SplitHostPort(value); err == nil {
		return value
	}
	if ip := net.ParseIP(strings.Trim(value, "[]")); ip != nil {
		return net.JoinHostPort(ip.String(), strconv.Itoa(PortRelay))
	}
	return net.JoinHostPort(value, strconv.Itoa(PortRelay))
}

func usableRelayIP(ip net.IP) bool {
	return ip != nil && !ip.IsUnspecified() && !ip.IsLoopback()
}

func routedLocalIP(remoteIP string) net.IP {
	ip := net.ParseIP(strings.Trim(remoteIP, "[]"))
	if ip == nil || ip.IsUnspecified() {
		return nil
	}
	connection, err := net.DialUDP(udp, nil, &net.UDPAddr{IP: ip, Port: 9})
	if err != nil {
		return nil
	}
	defer connection.Close()
	local, _ := connection.LocalAddr().(*net.UDPAddr)
	if local == nil {
		return nil
	}
	return local.IP
}
