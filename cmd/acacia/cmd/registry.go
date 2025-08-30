package cmd

import (
	"errors"  // Import errors for creating and handling error messages
	"fmt"     // Import fmt for formatted I/O operations
	"os"      // Import os for operating system functionalities, like file system operations
	"strings" // Import strings for string manipulation functions

	"acacia/cmd/acacia/internal/registry" // Import the internal registry package for managing registry data

	"github.com/spf13/cobra" // Import Cobra for building powerful modern CLI applications
)

var registryFile string // registryFile holds the path to the registries.json file, configurable via a flag.

// init function is called before main. It sets up the Cobra commands and flags for registry operations.
func init() {
	// Add the 'registryCmd' as a subcommand of the 'rootCmd'.
	rootCmd.AddCommand(registryCmd)

	// Allow overriding the default registry file path.
	// The default path is determined by the registry.DefaultPath() function, typically in the user's config directory.
	path, _ := registry.DefaultPath()
	registryCmd.PersistentFlags().StringVar(&registryFile, "file", path, "path to registries.json")

	// Add subcommands for managing registry sources.
	registryCmd.AddCommand(registryAddSourceCmd)
	registryCmd.AddCommand(registryRemoveSourceCmd)
	registryCmd.AddCommand(registryListSourcesCmd)
}

// registryCmd is the Cobra command for managing module and gateway registries.
// It acts as a parent command for operations related to registry sources.
var registryCmd = &cobra.Command{
	Use:   "registry",                             // The command name
	Short: "Manage module and gateway registries", // A brief description
}

// registryAddSourceCmd is the Cobra command for adding a new Git repository URL as a remote registry source.
// It requires exactly one argument: the URL of the Git repository.
var registryAddSourceCmd = &cobra.Command{
	Use:   "add-source <url>",                                     // Usage string for the command
	Short: "Add a Git repository URL as a remote registry source", // A brief description
	Args:  cobra.ExactArgs(1),                                     // Ensures exactly one argument (the URL) is provided
	RunE: func(cmd *cobra.Command, args []string) error { // The function executed when the command is run
		url := strings.TrimSpace(args[0]) // Get and trim the URL from arguments.
		if url == "" {
			return errors.New("registry source URL required") // Return error if URL is empty.
		}
		entry := registry.RegistrySource{URL: url} // Create a RegistrySource struct with the provided URL.
		// Load the registry store from the configured file path.
		s, path, err := loadStore()
		if err != nil {
			return err // Return error if loading store fails.
		}
		// Attempt to add the registry source to the store.
		added := s.AddRegistrySource(entry)
		if !added {
			// If the source was already present, inform the user and exit successfully.
			fmt.Fprintf(cmd.OutOrStdout(), "registry source already present: %s\n", url)
			return nil
		}
		// Save the updated store back to the file.
		if err := saveStore(s, path); err != nil {
			return err // Return error if saving store fails.
		}
		// Inform the user that the registry source was successfully added.
		fmt.Fprintf(cmd.OutOrStdout(), "registry source added: %s\n", url)
		return nil
	},
}

// registryRemoveSourceCmd is the Cobra command for removing a Git repository URL from remote registry sources.
// It requires exactly one argument: the URL of the Git repository to remove.
var registryRemoveSourceCmd = &cobra.Command{
	Use:   "remove-source <url>",                                      // Usage string for the command
	Short: "Remove a Git repository URL from remote registry sources", // A brief description
	Args:  cobra.ExactArgs(1),                                         // Ensures exactly one argument (the URL) is provided
	RunE: func(cmd *cobra.Command, args []string) error { // The function executed when the command is run
		url := strings.TrimSpace(args[0]) // Get and trim the URL from arguments.
		if url == "" {
			return errors.New("registry source URL required") // Return error if URL is empty.
		}
		entry := registry.RegistrySource{URL: url} // Create a RegistrySource struct to identify the source to remove.
		// Load the registry store.
		s, path, err := loadStore()
		if err != nil {
			return err // Return error if loading store fails.
		}
		// Attempt to remove the registry source from the store.
		removed := s.RemoveRegistrySource(entry)
		if !removed {
			// If the source was not found, return an error.
			return fmt.Errorf("registry source not found: %s", url)
		}
		// Save the updated store back to the file.
		if err := saveStore(s, path); err != nil {
			return err // Return error if saving store fails.
		}
		// Inform the user that the registry source was successfully removed.
		fmt.Fprintf(cmd.OutOrStdout(), "registry source removed: %s\n", url)
		return nil
	},
}

// registryListSourcesCmd is the Cobra command for listing all registered remote registry sources.
var registryListSourcesCmd = &cobra.Command{
	Use:   "list-sources",                            // Usage string for the command
	Short: "List registered remote registry sources", // A brief description
	RunE: func(cmd *cobra.Command, args []string) error { // The function executed when the command is run
		// Load the registry store.
		s, _, err := loadStore()
		if err != nil {
			return err // Return error if loading store fails.
		}
		// Check if there are any registered registry sources.
		if len(s.RegistrySources) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no registry sources registered") // Inform if no sources are found.
			return nil
		}
		// Iterate over and print the URL of each registered registry source.
		for _, src := range s.RegistrySources {
			fmt.Fprintln(cmd.OutOrStdout(), src.URL)
		}
		return nil
	},
}

// loadStore loads the registry store from the configured file path.
// It returns the loaded store, the path from which it was loaded, and any error encountered.
func loadStore() (*registry.Store, string, error) {
	path := registryFile // Use the path specified by the --file flag.
	if path == "" {
		var err error
		path, err = registry.DefaultPath() // If no flag is provided, use the default path.
		if err != nil {
			return nil, "", err // Return error if default path cannot be determined.
		}
	}
	s, err := registry.Load(path) // Load the store from the determined path.
	if err != nil {
		return nil, "", err // Return error if loading fails.
	}
	return s, path, nil // Return the loaded store, its path, and nil for no error.
}

// saveStore saves the provided registry store to the specified file path.
// It returns any error encountered during the save operation.
func saveStore(s *registry.Store, path string) error {
	if err := registry.Save(s, path); err != nil {
		return err // Return error if saving fails.
	}
	// Inform the user that the registry was successfully saved.
	fmt.Fprintf(os.Stdout, "saved registry: %s\n", path)
	return nil
}
