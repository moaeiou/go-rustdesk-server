package command

import (
	"fmt"
)

func PrintHelp() {
	fmt.Println("Usage:")
	fmt.Println("  go-rustdesk-server [command]")
	fmt.Println("Commands:")
	fmt.Println("  (none)              Start the RustDesk relay server")
	fmt.Println("  version             Print version and build info")
	fmt.Println("  doctor <tcp|udp> <port>")
	fmt.Println("                      Check one local service")
	fmt.Println("  doctor <host> <tcp|udp> <port>")
	fmt.Println("                      Check one service on a remote host")
	fmt.Println("  help or any unknown command")
	fmt.Println("                      Show this help message")
}
