package cmd

import (
	"acacia/cmd/acacia/internal/registry" // Import the internal registry package for managing gateway entries
	"errors"                              // Import errors for creating and handling error messages
	"fmt"                                 // Import fmt for formatted I/O operations
	"strings"                             // Import strings for string manipulation functions

	"github.com/spf13/cobra" // Import Cobra for building powerful modern CLI applications
)

var (
	gatewayCompany string // gatewayCompany holds the company/dependence flag value for gateway commands.
	gatewayAuthor  string // gatewayAuthor holds the author flag value for gateway commands.
	gatewayVersion string // gatewayVersion holds the version flag value for gateway commands.
)

// init function is called before main. It sets up the Cobra commands and flags.
func init() {
	// Add the 'gatewaysCmd' as a subcommand of 'registryCmd'.
	registryCmd.AddCommand(gatewaysCmd)

	// Define flags for the 'gateways add' command.
	// These flags allow users to specify additional metadata when adding a gateway.
	gatewaysAddCmd.Flags().StringVar(&gatewayCompany, "company", "", "company/dependence of the gateway")
	gatewaysAddCmd.Flags().StringVar(&gatewayAuthor, "author", "", "author of the gateway")
	gatewaysAddCmd.Flags().StringVar(&gatewayVersion, "version", "", "version of the gateway")
	// Add the 'gatewaysAddCmd' as a subcommand of 'gatewaysCmd'.
	gatewaysCmd.AddCommand(gatewaysAddCmd)

	// Define flags for the 'gateways remove' command.
	// These flags help in uniquely identifying the gateway to be removed.
	gatewaysRemoveCmd.Flags().StringVar(&gatewayCompany, "company", "", "company/dependence of the gateway")
	gatewaysRemoveCmd.Flags().StringVar(&gatewayAuthor, "author", "", "author of the gateway")
	gatewaysRemoveCmd.Flags().StringVar(&gatewayVersion, "version", "", "version of the gateway")
	// Add the 'gatewaysRemoveCmd' as a subcommand of 'gatewaysCmd'.
	gatewaysCmd.AddCommand(gatewaysRemoveCmd)

	// Add the 'gatewaysListCmd' as a subcommand of 'gatewaysCmd'.
	gatewaysCmd.AddCommand(gatewaysListCmd)
}

// gatewaysCmd is the Cobra command for managing gateway registry entries.
// It acts as a parent command for 'add', 'remove', and 'list' gateway operations.
var gatewaysCmd = &cobra.Command{
	Use:   "gateways",                        // The command name
	Short: "Manage gateway registry entries", // A brief description
}

// gatewaysAddCmd is the Cobra command for adding a new gateway entry to the registry.
// It requires exactly one argument: the gateway name.
var gatewaysAddCmd = &cobra.Command{
	Use:   "add <name>",                         // Usage string for the command
	Short: "Add a gateway name to the registry", // A brief description
	Args:  cobra.ExactArgs(1),                   // Ensures exactly one argument (the gateway name) is provided
	RunE: func(cmd *cobra.Command, args []string) error { // The function executed when the command is run
		name := strings.TrimSpace(args[0]) // Get and trim the gateway name from arguments
		if name == "" {
			return errors.New("gateway name required") // Return error if name is empty
		}
		// Create a GatewayEntry struct with the provided name and flag values.
		entry := registry.GatewayEntry{
			Name:    name,
			Company: gatewayCompany,
			Author:  gatewayAuthor,
			Version: gatewayVersion,
		}
		// Load the registry store from the configured file path.
		s, path, err := loadStore()
		if err != nil {
			return err // Return error if loading store fails
		}
		// Attempt to add the gateway entry to the store.
		added := s.AddGateway(entry)
		if !added {
			// If the gateway was already present, inform the user and exit successfully.
			fmt.Fprintf(cmd.OutOrStdout(), "gateway already present: %+v\n", entry)
			return nil
		}
		// Save the updated store back to the file.
		if err := saveStore(s, path); err != nil {
			return err // Return error if saving store fails
		}
		// Inform the user that the gateway was successfully added.
		fmt.Fprintf(cmd.OutOrStdout(), "gateway added: %+v\n", entry)
		return nil
	},
}

// gatewaysRemoveCmd is the Cobra command for removing a gateway entry from the registry.
// It requires exactly one argument: the gateway name.
var gatewaysRemoveCmd = &cobra.Command{
	Use:   "remove <name>",                           // Usage string for the command
	Short: "Remove a gateway name from the registry", // A brief description
	Args:  cobra.ExactArgs(1),                        // Ensures exactly one argument (the gateway name) is provided
	RunE: func(cmd *cobra.Command, args []string) error { // The function executed when the command is run
		name := strings.TrimSpace(args[0]) // Get and trim the gateway name from arguments
		if name == "" {
			return errors.New("gateway name required") // Return error if name is empty
		}
		// Create a GatewayEntry struct to identify the gateway to be removed.
		entry := registry.GatewayEntry{
			Name:    name,
			Company: gatewayCompany,
			Author:  gatewayAuthor,
			Version: gatewayVersion,
		}
		// Load the registry store.
		s, path, err := loadStore()
		if err != nil {
			return err // Return error if loading store fails
		}
		// Attempt to remove the gateway entry from the store.
		removed := s.RemoveGateway(entry)
		if !removed {
			// If the gateway was not found, return an error.
			return fmt.Errorf("gateway not found: %+v", entry)
		}
		// Save the updated store back to the file.
		if err := saveStore(s, path); err != nil {
			return err // Return error if saving store fails
		}
		// Inform the user that the gateway was successfully removed.
		fmt.Fprintf(cmd.OutOrStdout(), "gateway removed: %+v\n", entry)
		return nil
	},
}

// gatewaysListCmd is the Cobra command for listing all registered gateway entries.
var gatewaysListCmd = &cobra.Command{
	Use:   "list",                     // Usage string for the command
	Short: "List registered gateways", // A brief description
	RunE: func(cmd *cobra.Command, args []string) error { // The function executed when the command is run
		// Load the registry store.
		s, _, err := loadStore()
		if err != nil {
			return err // Return error if loading store fails
		}
		// Check if there are any registered gateways.
		if len(s.Gateways) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no gateways registered") // Inform if no gateways are found
			return nil
		}
		// Iterate over and print details of each registered gateway.
		for _, g := range s.Gateways {
			fmt.Fprintf(cmd.OutOrStdout(), "Name: %s, Company: %s, Author: %s, Version: %s\n", g.Name, g.Company, g.Author, g.Version)
		}
		return nil
	},
}
