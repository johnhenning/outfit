package main

import (
	"context"
	"fmt"
	"github.com/johnhenning/outfit/internal/cli"
	"os"
	"os/signal"
)

var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := cli.New(version).ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "outfit:", err)
		os.Exit(1)
	}
}
