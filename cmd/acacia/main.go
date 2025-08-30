package main

import (
	"acacia/cmd/acacia/cmd" // Import the CLI commands package

	"acacia/core/logger" // Import the logging package

	"context"   // For context management, especially for graceful shutdown
	"os"        // For operating system functionalities, like signal handling
	"os/signal" // For listening to OS signals
	"syscall"   // For specific system calls, like SIGINT and SIGTERM

	"go.uber.org/zap" // For structured logging
)

// main is the entry point of the Acacia application.
func main() {
	// Create context for logging (with component name)
	ctx := context.WithValue(context.Background(), "componentName", "main")

	// Ensure that the logger's buffered logs are flushed before the application exits.
	defer func() {
		if err := logger.Logger.Sync(); err != nil {
			// If syncing fails, it's typically during application shutdown.
			// Avoid panicking here; instead, log the error to standard error
			// or handle it gracefully to ensure a clean exit.
			// fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
		}
	}()

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Info(ctx, "Received signal, initiating graceful shutdown", zap.String("signal", sig.String()))
		cancel()
	}()

	// Execute the root command of the Cobra CLI, passing the cancellable context.
	// Configuration, kernel, and plugin loading will now happen within the 'serve' command.
	cmd.Execute(ctx)
}
