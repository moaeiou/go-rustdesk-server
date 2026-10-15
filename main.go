package main

import (
	"context"
	"flag"
	"fmt"
	"go-rustdesk-server/command"
	"go-rustdesk-server/relay"
	"go-rustdesk-server/rendezvous"
	"go-rustdesk-server/transport"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "run":
			flags := flag.NewFlagSet("run", flag.ContinueOnError)
			flags.SetOutput(os.Stderr)
			configPath := flags.String("c", "", "path to config.toml")
			flags.StringVar(configPath, "config", "", "path to config.toml")
			if err := flags.Parse(os.Args[2:]); err != nil || flags.NArg() != 0 {
				command.PrintHelp()
				return 2
			}
			if *configPath != "" {
				if err := transport.LoadConfigFile(*configPath); err != nil {
					fmt.Fprintln(os.Stderr, err)
					return 1
				}
			}
		case "version":
			command.PrintVersion()
		case "doctor":
			command.Doctor()
		case "help":
			command.PrintHelp()
			return 0
		default:
			command.PrintHelp()
			return 2
		}
	}
	if err := transport.PrepareDataDir(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := transport.Log(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer transport.CloseLog()
	if err := transport.LoadKey(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	errors := make(chan error, 2)
	go func() { errors <- rendezvous.Run(ctx) }()
	go func() { errors <- relay.Run(ctx) }()
	completed := 0
	failed := false
	select {
	case <-ctx.Done():
		transport.Info("Shutting down")
	case err := <-errors:
		completed = 1
		failed = true
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Fprintln(os.Stderr, "critical service stopped unexpectedly")
		}
	}
	stop()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for completed < 2 {
		select {
		case err := <-errors:
			completed++
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		case <-timer.C:
			transport.Warn("Shutdown timed out")
			return 1
		}
	}
	if failed {
		return 1
	}
	return 0
}
