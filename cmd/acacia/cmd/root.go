package cmd

import (
	"acacia/core/config" // Import the config package to access application configuration structures
	// Import the kernel package for the Kernel interface
	"context" // Import context for managing request-scoped values, cancellation signals, and deadlines
	"fmt"     // Import fmt for formatted I/O operations (e.g., printing to console)
	"os"      // Import os for operating system functionalities, such as exiting the program

	"github.com/spf13/cobra" // Import Cobra for building powerful modern CLI applications
)

var (
	version   = "0.1.0"      // version holds the current version of the Acacia CLI application.
	appConfig *config.Config // appConfig is a package-level variable to store the loaded application configuration,
	// making it accessible across different Cobra commands.
)

// rootCmd is the base command for the Acacia CLI.
// It defines the main entry point for all Acacia commands and their default behavior.
var rootCmd = &cobra.Command{
	Use:     "acacia",                                                        // The name of the command that users will type
	Short:   "Acacia Engine CLI",                                             // A brief, one-line description of the command
	Long:    "Acacia Engine CLI for managing and running the Acacia server.", // A longer description providing more detail
	Version: version,                                                         // Sets the version string for the command
	// RunE defines the function to execute when the rootCmd is called without any subcommands.
	// It now displays the help message by default, instead of running the 'serve' command.
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help() // Display help message
	},
}

// Execute is the main function to execute the root command of the Acacia CLI.
// It now only takes a context for graceful shutdown. Configuration and kernel
// initialization will happen within the 'serve' command.
func Execute(ctx context.Context) {
	// Set the context for the root command. This context can be used by subcommands
	// to handle cancellation signals (e.g., from OS interrupts).
	rootCmd.SetContext(ctx)

	// Execute the root command. If an error occurs during execution, it is printed
	// to standard error, and the application exits with a non-zero status code.
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err) // Print the error message to standard error
		os.Exit(1)                   // Exit the application with an error code
	}
}
