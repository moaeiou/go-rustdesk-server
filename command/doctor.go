package command

import (
	"fmt"
	"go-rustdesk-server/protocol"
	"google.golang.org/protobuf/proto"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const dialTimeout = 3 * time.Second

type serviceCheck struct {
	network     string
	port        int
	description string
}

var standardServiceChecks = []serviceCheck{
	{network: "tcp", port: 21114, description: "HTTP compatibility"},
	{network: "tcp", port: 21115, description: "NAT test"},
	{network: "tcp", port: 21116, description: "ID and rendezvous"},
	{network: "udp", port: 21116, description: "Registration and heartbeat"},
	{network: "tcp", port: 21117, description: "Relay"},
	{network: "tcp", port: 21118, description: "Web client"},
	{network: "tcp", port: 21119, description: "HTTP compatibility"},
}

func Doctor() {
	arguments := os.Args[2:]
	if len(arguments) == 1 {
		if _, _, err := net.SplitHostPort(arguments[0]); err == nil {
			PrintHelp()
			os.Exit(2)
		}
		doctorStandardServices(arguments[0])
	}
	if len(arguments) != 2 {
		PrintHelp()
		os.Exit(2)
	}
	host, network, port, err := doctorTarget(arguments[0], arguments[1])
	if err != nil {
		PrintHelp()
		os.Exit(2)
	}
	doctorService(host, serviceCheck{network: network, port: port, description: "Custom service"})
}

func doctorStandardServices(host string) {
	addresses, err := resolveDoctorHost(host)
	if err != nil {
		fmt.Printf("[Doctor] DNS FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()
	fmt.Printf("[Doctor] Checking standard RustDesk services on every resolved address\n\n")
	results := checkAllAddresses(addresses, standardServiceChecks)
	allReachable := true
	for addressIndex, ip := range addresses {
		fmt.Printf("  %s %s\n", ipFamily(ip), ip)
		for checkIndex, check := range standardServiceChecks {
			status := "OK"
			if !results[addressIndex][checkIndex] {
				status = "FAIL"
				allReachable = false
			}
			endpoint := net.JoinHostPort(ip.String(), strconv.Itoa(check.port))
			fmt.Printf("    %-4s %-3s %-24s %s\n", status, strings.ToUpper(check.network), endpoint, check.description)
		}
		fmt.Println()
	}
	if !allReachable {
		fmt.Println("[Doctor] Some resolved addresses or services failed")
		os.Exit(1)
	}
	fmt.Println("[Doctor] All services are reachable over every resolved address")
	os.Exit(0)
}

func doctorService(host string, check serviceCheck) {
	addresses, err := resolveDoctorHost(host)
	if err != nil {
		fmt.Printf("[Doctor] DNS FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()
	results := checkAllAddresses(addresses, []serviceCheck{check})
	allReachable := true
	for index, ip := range addresses {
		status := "OK"
		if !results[index][0] {
			status = "FAIL"
			allReachable = false
		}
		endpoint := net.JoinHostPort(ip.String(), strconv.Itoa(check.port))
		fmt.Printf("  %-4s %-4s %-3s %s\n", status, ipFamily(ip), strings.ToUpper(check.network), endpoint)
	}
	if !allReachable {
		fmt.Println("[Doctor] Service failed on one or more resolved addresses")
		os.Exit(1)
	}
	fmt.Println("[Doctor] Service is reachable over every resolved address")
	os.Exit(0)
}

func resolveDoctorHost(host string) ([]net.IP, error) {
	host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	fmt.Printf("[Doctor] Resolving %s\n", host)
	if parsed := net.ParseIP(host); parsed != nil {
		fmt.Printf("  %s %s (literal)\n", ipFamily(parsed), parsed)
		return []net.IP{parsed}, nil
	}
	if canonical, err := net.LookupCNAME(host); err == nil {
		canonical = strings.TrimSuffix(canonical, ".")
		if !strings.EqualFold(canonical, strings.TrimSuffix(host, ".")) {
			fmt.Printf("  CNAME %s\n", canonical)
		}
	}
	resolved, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	addresses := uniqueSortedIPs(resolved)
	if len(addresses) == 0 {
		return nil, fmt.Errorf("%s has no IPv4 or IPv6 address", host)
	}
	for _, ip := range addresses {
		fmt.Printf("  %s %s\n", ipFamily(ip), ip)
	}
	return addresses, nil
}

func uniqueSortedIPs(resolved []net.IP) []net.IP {
	unique := make(map[string]net.IP, len(resolved))
	for _, ip := range resolved {
		unique[ip.String()] = ip
	}
	addresses := make([]net.IP, 0, len(unique))
	for _, ip := range unique {
		addresses = append(addresses, ip)
	}
	sort.Slice(addresses, func(i, j int) bool {
		iV4, jV4 := addresses[i].To4() != nil, addresses[j].To4() != nil
		if iV4 != jV4 {
			return iV4
		}
		return addresses[i].String() < addresses[j].String()
	})
	return addresses
}

func checkAllAddresses(addresses []net.IP, checks []serviceCheck) [][]bool {
	results := make([][]bool, len(addresses))
	completed := make(chan struct{}, len(addresses)*len(checks))
	for addressIndex, ip := range addresses {
		results[addressIndex] = make([]bool, len(checks))
		for checkIndex, check := range checks {
			go func() {
				endpoint := net.JoinHostPort(ip.String(), strconv.Itoa(check.port))
				results[addressIndex][checkIndex] = isReachable(check.network, endpoint)
				completed <- struct{}{}
			}()
		}
	}
	for count := 0; count < len(addresses)*len(checks); count++ {
		<-completed
	}
	return results
}

func ipFamily(ip net.IP) string {
	if ip.To4() != nil {
		return "IPv4"
	}
	return "IPv6"
}

func doctorTarget(endpoint, serviceType string) (string, string, int, error) {
	host, portText, err := net.SplitHostPort(endpoint)
	if err != nil || host == "" {
		return "", "", 0, fmt.Errorf("invalid endpoint %q", endpoint)
	}
	network := strings.ToLower(serviceType)
	if network != "tcp" && network != "udp" {
		return "", "", 0, fmt.Errorf("unsupported service type %q", serviceType)
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return "", "", 0, fmt.Errorf("invalid port %q", portText)
	}
	return host, network, port, nil
}

func isReachable(network, address string) bool {
	if network == "udp" {
		return probeRustDeskUDP(address)
	}
	connection, err := net.DialTimeout(network, address, dialTimeout)
	if err != nil {
		return false
	}
	_ = connection.Close()
	return true
}

func probeRustDeskUDP(address string) bool {
	remote, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return false
	}
	connection, err := net.DialUDP("udp", nil, remote)
	if err != nil {
		return false
	}
	defer connection.Close()
	request := &protocol.RendezvousMessage{Union: &protocol.RendezvousMessage_RegisterPeer{
		RegisterPeer: &protocol.RegisterPeer{Id: "doctor-check"},
	}}
	payload, err := proto.Marshal(request)
	if err != nil {
		return false
	}
	if err := connection.SetDeadline(time.Now().Add(dialTimeout)); err != nil {
		return false
	}
	if _, err := connection.Write(payload); err != nil {
		return false
	}
	responseData := make([]byte, 65535)
	length, err := connection.Read(responseData)
	if err != nil {
		return false
	}
	response := &protocol.RendezvousMessage{}
	return proto.Unmarshal(responseData[:length], response) == nil && response.GetRegisterPeerResponse() != nil
}
