package rendezvous

import (
	"context"
	"fmt"
	"go-rustdesk-server/transport"
)

func Run(ctx context.Context) error {
	if err := loadPeers(); err != nil {
		return fmt.Errorf("load peers: %w", err)
	}
	errors := make(chan error, 6)
	startHTTPCompatibility(ctx, errors)
	go func() {
		errors <- transport.NewMonitor(false, "udp", transport.ListenAddress(transport.PortSignal), handlerMsg).Run(ctx)
	}()
	for _, addr := range []int{transport.PortNAT, transport.PortSignal, transport.PortWS} {
		addr := addr
		go func() {
			errors <- transport.NewMonitor(false, "tcp", transport.ListenAddress(addr), handlerMsg).Run(ctx)
		}()
	}
	select {
	case <-ctx.Done():
		return nil
	case err := <-errors:
		if err != nil {
			return fmt.Errorf("rendezvous service: %w", err)
		}
		return nil
	}
}
