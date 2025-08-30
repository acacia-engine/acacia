package cmd

import (
	"acacia/cmd/acacia/internal/pluginloader" // Import the pluginloader package

	"acacia/core/auth"    // Import the auth package for AccessController
	"acacia/core/config"  // Import the configuration package
	"acacia/core/kernel"  // Import the kernel package for the Kernel interface
	"acacia/core/logger"  // Import the logging package
	"acacia/core/metrics" // Import the metrics package

	"context"   // Import context for managing request-scoped values, cancellation signals, and deadlines
	"log"       // Import log for simple logging
	"os"        // For operating system functionalities, like signal handling
	"os/signal" // For listening to OS signals
	"syscall"   // For specific system calls, like SIGINT and SIGTERM
	"time"      // Import time for duration and timeout functionalities

	"github.com/spf13/cobra" // Import Cobra for building powerful modern CLI applications
	"go.uber.org/zap"        // For structured logging
)

// init function is called before main. It sets up the Cobra command for serving the application.
func init() {
	// Add the 'serveCmd' as a subcommand of the 'rootCmd'.
	rootCmd.AddCommand(serveCmd)
}

// serveCmd is the Cobra command for running the Acacia server.
var serveCmd = &cobra.Command{
	Use:   "serve",                 // The command name
	Short: "Run the Acacia server", // A brief description
	RunE: func(cmd *cobra.Command, args []string) error { // The function executed when the command is run
		// Create context for logging (with component name)
		ctx := context.WithValue(cmd.Context(), "componentName", "serve")

		// Load application configuration from its source (e.g., file, environment variables).
		cfg, err := config.LoadConfig()
		if err != nil {
			logger.Fatal(ctx, "Failed to load configuration", zap.Error(err))
		}
		logger.Info(ctx, "Configuration loaded successfully", zap.Any("config", cfg))

		// Create an RBAC provider from the loaded configuration.
		rbacProvider := auth.NewConfigRBACProvider(cfg.Auth.Roles)

		// Create a DefaultAccessController instance with the provider.
		var accessController auth.AccessController = auth.NewDefaultAccessController(rbacProvider)

		// Inject the access controller into logger and metrics packages
		logger.SetAccessController(accessController)
		metrics.SetAccessController(accessController)

		// Create a new kernel instance with the access controller
		k := kernel.New(cfg, accessController)

		// Load plugins (modules and gateways) from the build/plugins directory
		pluginDir := "build/plugins" // Assuming plugins are built into this directory
		if err := pluginloader.LoadPlugins(k, pluginDir, cfg, logger.Logger); err != nil {
			logger.Fatal(ctx, "Failed to load plugins", zap.Error(err))
		}
		logger.Info(ctx, "Plugins loaded successfully", zap.String("pluginDir", pluginDir))

		// Start the kernel. This typically involves starting internal services,
		// listening for connections, and initializing modules/gateways.
		if err := k.Start(ctx); err != nil {
			return err // Return error if kernel fails to start.
		}
		log.Printf("Acacia server started.") // Log that the server has started.

		// Set up a channel to listen for OS interrupt signals (SIGINT, SIGTERM).
		// These signals are used to trigger a graceful shutdown.
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh // Block until an OS signal is received.

		// Initiate graceful shutdown.
		// Create a context with a timeout for the shutdown process to prevent indefinite blocking.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		defer cancel() // Ensure the cancel function is called to release resources.

		// Stop the kernel gracefully. This allows ongoing operations to complete
		// within the shutdown context's timeout.
		if err := k.Stop(shutdownCtx); err != nil {
			log.Printf("error during shutdown: %v", err) // Log any errors encountered during shutdown.
		}

		log.Println("Acacia stopped gracefully") // Log successful graceful shutdown.
		return nil
	},
}

// serve is no longer needed as its logic is now directly in serveCmd.RunE
// func serve(cmd *cobra.Command, args []string, krn kernel.Kernel) error {
// 	// ... (logic moved to serveCmd.RunE)
// }
