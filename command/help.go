package command

import (
	"fmt"
)

func PrintHelp() {
	fmt.Println("Usage:")
	fmt.Println("  go-rustdesk-server [command]")
	fmt.Println("Commands:")
	fmt.Println("  (none)              Start using ./config.toml (compatibility)")
	fmt.Println("  run                 Start the RustDesk server")
	fmt.Println("    -c, --config PATH Load configuration from PATH")
	fmt.Println("  version             Print version and build info")
	fmt.Println("  doctor <host:port> <tcp|udp>")
	fmt.Println("                      Check service")
	fmt.Println("  help/any unknown command")
	fmt.Println("                      Show this help message")
}
