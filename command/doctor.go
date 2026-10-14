package command

import (
	"fmt"
	"go-rustdesk-server/protocol"
	"google.golang.org/protobuf/proto"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const dialTimeout = 3 * time.Second

func Doctor() {
	host, network, port, err := doctorTarget(os.Args[2:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		printDoctorUsage()
		os.Exit(2)
	}
	address := net.JoinHostPort(host, strconv.Itoa(port))
	fmt.Printf("[Doctor] Checking %s service at %s\n", strings.ToUpper(network), address)
	if !isReachable(network, address) {
		fmt.Println("[Doctor] FAIL: service is unreachable or did not answer correctly")
		os.Exit(1)
	}
	fmt.Println("[Doctor] OK: service is reachable")
	os.Exit(0)
}

func doctorTarget(arguments []string) (string, string, int, error) {
	host := "localhost"
	if len(arguments) == 3 {
		host = arguments[0]
		arguments = arguments[1:]
	}
	if len(arguments) != 2 {
		return "", "", 0, fmt.Errorf("doctor requires a service type and port")
	}
	network := strings.ToLower(arguments[0])
	if network != "tcp" && network != "udp" {
		return "", "", 0, fmt.Errorf("unsupported service type %q; use tcp or udp", arguments[0])
	}
	port, err := strconv.Atoi(arguments[1])
	if err != nil || port < 1 || port > 65535 {
		return "", "", 0, fmt.Errorf("invalid port %q", arguments[1])
	}
	return host, network, port, nil
}

func printDoctorUsage() {
	fmt.Println("Usage:")
	fmt.Println("  go-rustdesk-server doctor <tcp|udp> <port>")
	fmt.Println("  go-rustdesk-server doctor <host> <tcp|udp> <port>")
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
