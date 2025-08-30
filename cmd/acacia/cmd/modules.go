package cmd

import (
	"acacia/cmd/acacia/internal/registry" // Import the internal registry package for managing module entries
	"errors"                              // Import errors for creating and handling error messages
	"fmt"                                 // Import fmt for formatted I/O operations
	"strings"                             // Import strings for string manipulation functions

	"github.com/spf13/cobra" // Import Cobra for building powerful modern CLI applications
)

var (
	moduleCompany string // moduleCompany holds the company/dependence flag value for module commands.
	moduleAuthor  string // moduleAuthor holds the author flag value for module commands.
	moduleVersion string // moduleVersion holds the version flag value for module commands.
)

// init function is called before main. It sets up the Cobra commands and flags for module operations.
func init() {
	// Add the 'modulesCmd' as a subcommand of 'registryCmd'.
	registryCmd.AddCommand(modulesCmd)

	// Define flags for the 'modules add' command.
	// These flags allow users to specify additional metadata when adding a module.
	modulesAddCmd.Flags().StringVar(&moduleCompany, "company", "", "company/dependence of the module")
	modulesAddCmd.Flags().StringVar(&moduleAuthor, "author", "", "author of the module")
	modulesAddCmd.Flags().StringVar(&moduleVersion, "version", "", "version of the module")
	// Add the 'modulesAddCmd' as a subcommand of 'modulesCmd'.
	modulesCmd.AddCommand(modulesAddCmd)

	// Define flags for the 'modules remove' command.
	// These flags help in uniquely identifying the module to be removed.
	modulesRemoveCmd.Flags().StringVar(&moduleCompany, "company", "", "company/dependence of the module")
	modulesRemoveCmd.Flags().StringVar(&moduleAuthor, "author", "", "author of the module")
	modulesRemoveCmd.Flags().StringVar(&moduleVersion, "version", "", "version of the module")
	// Add the 'modulesRemoveCmd' as a subcommand of 'modulesCmd'.
	modulesCmd.AddCommand(modulesRemoveCmd)

	// Add the 'modulesListCmd' as a subcommand of 'modulesCmd'.
	modulesCmd.AddCommand(modulesListCmd)
}

// modulesCmd is the Cobra command for managing module registry entries.
// It acts as a parent command for 'add', 'remove', and 'list' module operations.
var modulesCmd = &cobra.Command{
	Use:   "modules",                        // The command name
	Short: "Manage module registry entries", // A brief description
}

// modulesAddCmd is the Cobra command for adding a new module entry to the registry.
// It requires exactly one argument: the module name.
var modulesAddCmd = &cobra.Command{
	Use:   "add <name>",                        // Usage string for the command
	Short: "Add a module name to the registry", // A brief description
	Args:  cobra.ExactArgs(1),                  // Ensures exactly one argument (the module name) is provided
	RunE: func(cmd *cobra.Command, args []string) error { // The function executed when the command is run
		name := strings.TrimSpace(args[0]) // Get and trim the module name from arguments.
		if name == "" {
			return errors.New("module name required") // Return error if name is empty.
		}
		// Create a ModuleEntry struct with the provided name and flag values.
		entry := registry.ModuleEntry{
			Name:    name,
			Company: moduleCompany,
			Author:  moduleAuthor,
			Version: moduleVersion,
		}
		// Load the registry store from the configured file path.
		s, path, err := loadStore()
		if err != nil {
			return err // Return error if loading store fails.
		}
		// Attempt to add the module entry to the store.
		added := s.AddModule(entry)
		if !added {
			// If the module was already present, inform the user and exit successfully.
			fmt.Fprintf(cmd.OutOrStdout(), "module already present: %+v\n", entry)
			return nil
		}
		// Save the updated store back to the file.
		if err := saveStore(s, path); err != nil {
			return err // Return error if saving store fails.
		}
		// Inform the user that the module was successfully added.
		fmt.Fprintf(cmd.OutOrStdout(), "module added: %+v\n", entry)
		return nil
	},
}

// modulesRemoveCmd is the Cobra command for removing a module entry from the registry.
// It requires exactly one argument: the module name.
var modulesRemoveCmd = &cobra.Command{
	Use:   "remove <name>",                          // Usage string for the command
	Short: "Remove a module name from the registry", // A brief description
	Args:  cobra.ExactArgs(1),                       // Ensures exactly one argument (the module name) is provided
	RunE: func(cmd *cobra.Command, args []string) error { // The function executed when the command is run
		name := strings.TrimSpace(args[0]) // Get and trim the module name from arguments.
		if name == "" {
			return errors.New("module name required") // Return error if name is empty.
		}
		// Create a ModuleEntry struct to identify the module to be removed.
		entry := registry.ModuleEntry{
			Name:    name,
			Company: moduleCompany,
			Author:  moduleAuthor,
			Version: moduleVersion,
		}
		// Load the registry store.
		s, path, err := loadStore()
		if err != nil {
			return err // Return error if loading store fails.
		}
		// Attempt to remove the module entry from the store.
		removed := s.RemoveModule(entry)
		if !removed {
			// If the module was not found, return an error.
			return fmt.Errorf("module not found: %+v", entry)
		}
		// Save the updated store back to the file.
		if err := saveStore(s, path); err != nil {
			return err // Return error if saving store fails.
		}
		// Inform the user that the module was successfully removed.
		fmt.Fprintf(cmd.OutOrStdout(), "module removed: %+v\n", entry)
		return nil
	},
}

// modulesListCmd is the Cobra command for listing all registered module entries.
var modulesListCmd = &cobra.Command{
	Use:   "list",                    // Usage string for the command
	Short: "List registered modules", // A brief description
	RunE: func(cmd *cobra.Command, args []string) error { // The function executed when the command is run
		// Load the registry store.
		s, _, err := loadStore()
		if err != nil {
			return err // Return error if loading store fails.
		}
		// Check if there are any registered modules.
		if len(s.Modules) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no modules registered") // Inform if no modules are found.
			return nil
		}
		// Iterate over and print details of each registered module.
		for _, m := range s.Modules {
			fmt.Fprintf(cmd.OutOrStdout(), "Name: %s, Company: %s, Author: %s, Version: %s\n", m.Name, m.Company, m.Author, m.Version)
		}
		return nil
	},
}
