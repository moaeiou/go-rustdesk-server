package rendezvous

import (
	"context"
	"go-rustdesk-server/transport"
	"net/http"
	"time"
)

func startHTTPCompatibility(ctx context.Context, errors chan<- error) {
	ports := []int{transport.PortHTTPAPI, transport.PortWebAPI}
	for _, port := range ports {
		go func() { errors <- listenOn(ctx, transport.ListenAddress(port)) }()
	}
}

func listenOn(ctx context.Context, addr string) error {
	handler := func(w http.ResponseWriter, r *http.Request) {
		transport.Info("HTTP %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		w.Header().Set("Content-Length", "0")
		w.Header().Del("Content-Type")
		w.WriteHeader(http.StatusOK)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	transport.Info("Fake HTTP server listening on %s", addr)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
