package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	modDirFlag          string
	modForceFlag        bool
	modVersionFlag      string
	modDescriptionFlag  string
	modDependenciesFlag []string
)

func init() {
	rootCmd.AddCommand(moduleCmd)
	moduleCmd.AddCommand(moduleCreateCmd)
	moduleCmd.AddCommand(moduleAddCmd)
	moduleCmd.AddCommand(moduleRemoveCmd)

	moduleCreateCmd.Flags().StringVar(&modDirFlag, "dir", "modules", "base directory where the module will be created")
	moduleCreateCmd.Flags().BoolVar(&modForceFlag, "force", false, "overwrite if the target directory already exists")
	moduleCreateCmd.Flags().StringVar(&modVersionFlag, "version", "", "module version to write into registry.json")
	moduleCreateCmd.Flags().StringVar(&modDescriptionFlag, "description", "", "optional module description")
	moduleCreateCmd.Flags().StringSliceVar(&modDependenciesFlag, "dependencies", []string{}, "comma-separated list of module dependencies")

	moduleRemoveCmd.Flags().StringVar(&modDirFlag, "dir", "modules", "base directory where the module is located")
	moduleRemoveCmd.Flags().BoolVar(&modForceFlag, "force", false, "force removal without confirmation")
}

var moduleCmd = &cobra.Command{
	Use:   "module",
	Short: "Module utilities (scaffolding, etc.)",
}

var moduleCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new module folder with the standard structure",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var name string
		if len(args) > 0 {
			name = args[0]
		}

		if name == "" {
			prompt := &survey.Input{Message: "Module name:"}
			survey.AskOne(prompt, &name, survey.WithValidator(survey.Required))
		}

		name = strings.TrimSpace(name)
		if err := validateModuleName(name); err != nil {
			return err
		}

		if modVersionFlag == "" {
			prompt := &survey.Input{Message: "Version:", Default: "0.1.0"}
			survey.AskOne(prompt, &modVersionFlag)
		}

		if modDescriptionFlag == "" {
			prompt := &survey.Input{Message: "Description:"}
			survey.AskOne(prompt, &modDescriptionFlag)
		}

		if len(modDependenciesFlag) == 0 {
			var depsInput string
			prompt := &survey.Input{Message: "Dependencies (comma-separated, e.g., module1,module2):"}
			survey.AskOne(prompt, &depsInput)
			if depsInput != "" {
				modDependenciesFlag = strings.Split(depsInput, ",")
				for i, dep := range modDependenciesFlag {
					modDependenciesFlag[i] = strings.TrimSpace(dep)
				}
			}
		}

		baseDir := strings.TrimSpace(modDirFlag)
		if baseDir == "" {
			baseDir = "modules"
		}
		target := filepath.Join(baseDir, name)

		if st, err := os.Stat(target); err == nil && st.IsDir() {
			if !modForceFlag {
				return fmt.Errorf("module directory already exists: %s (use --force to overwrite)", target)
			}
		} else if err == nil && !st.IsDir() {
			return fmt.Errorf("path exists and is not a directory: %s", target)
		}

		subdirs := []string{
			filepath.Join(target, "application"),
			filepath.Join(target, "domain"),
			filepath.Join(target, "infrastructure"),
			filepath.Join(target, "docs"),
		}

		// Create configs directory for module-specific configuration
		configsDir := filepath.Join(target, "configs")
		subdirs = append(subdirs, configsDir)
		for _, d := range subdirs {
			if err := os.MkdirAll(d, 0o755); err != nil {
				return err
			}
		}

		for _, d := range subdirs[:3] {
			if err := os.WriteFile(filepath.Join(d, ".gitkeep"), []byte(""), 0o644); err != nil {
				return err
			}
		}

		docsReadme := fmt.Sprintf("# %s Module\n\nThis module was scaffolded by the Acacia CLI.\n\nStructure:\n- application/: use cases and services\n- domain/: entities, value objects, domain interfaces\n- infrastructure/: adapters implementing domain interfaces\n- docs/: documentation\n- configs/: default configuration templates\n\n", name)
		if err := os.WriteFile(filepath.Join(target, "docs", "README.md"), []byte(docsReadme), 0o644); err != nil {
			return err
		}

		// Generate default config template
		defaultConfig := fmt.Sprintf(`# Default configuration for %s module
# This file contains default settings that will be merged into the main config.yaml
# when the module is added to the system.

modules:
  %s:
    enabled: true
    # Add module-specific configuration here
    # Example settings (customize as needed):
    # debug: false
    # timeout: "30s"
    # workers: 2
`, name, name)
		if err := os.WriteFile(filepath.Join(target, "configs", "default-config.yaml"), []byte(defaultConfig), 0o644); err != nil {
			return err
		}

		reg := struct {
			Name         string   `json:"name"`
			Version      string   `json:"version"`
			Description  string   `json:"description,omitempty"`
			Dependencies []string `json:"dependencies,omitempty"`
		}{
			Name:         name,
			Version:      modVersionFlag,
			Description:  strings.TrimSpace(modDescriptionFlag),
			Dependencies: modDependenciesFlag,
		}
		b, err := json.MarshalIndent(reg, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(target, "registry.json"), b, 0o644); err != nil {
			return err
		}

		modulePath := fmt.Sprintf("acacia.com/modules/%s", name)
		// Get the Go version from config or existing files
		goVersion, err := getGoVersion()
		if err != nil {
			return fmt.Errorf("failed to get Go version: %w", err)
		}
		goModContent := fmt.Sprintf("module %s\n\ngo %s\n", modulePath, goVersion)
		if err := os.WriteFile(filepath.Join(target, "go.mod"), []byte(goModContent), 0o644); err != nil {
			return err
		}

		if err := syncGoWorkspace(); err != nil {
			return fmt.Errorf("failed to sync go workspace: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "module scaffolded: %s\n", target)
		return nil
	},
}

var moduleAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a module's default configuration to the main config.yaml",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.TrimSpace(args[0])
		if err := validateModuleName(name); err != nil {
			return err
		}

		baseDir := strings.TrimSpace(modDirFlag)
		if baseDir == "" {
			baseDir = "modules"
		}

		// Check if module exists
		modulePath := filepath.Join(baseDir, name)
		if st, err := os.Stat(modulePath); os.IsNotExist(err) {
			return fmt.Errorf("module directory does not exist: %s", modulePath)
		} else if err != nil {
			return fmt.Errorf("error checking module directory: %w", err)
		} else if !st.IsDir() {
			return fmt.Errorf("path exists but is not a directory: %s", modulePath)
		}

		// Check if default config exists
		defaultConfigPath := filepath.Join(modulePath, "configs", "default-config.yaml")
		if _, err := os.Stat(defaultConfigPath); os.IsNotExist(err) {
			return fmt.Errorf("default config not found for module %s: %s", name, defaultConfigPath)
		}

		// Read the default config
		defaultConfigData, err := os.ReadFile(defaultConfigPath)
		if err != nil {
			return fmt.Errorf("failed to read default config: %w", err)
		}

		// Read the main config.yaml
		mainConfigPath := "config.yaml"
		var mainConfigData []byte
		if _, err := os.Stat(mainConfigPath); err == nil {
			mainConfigData, err = os.ReadFile(mainConfigPath)
			if err != nil {
				return fmt.Errorf("failed to read main config: %w", err)
			}
		} else {
			// Create a basic config if it doesn't exist
			mainConfigData = []byte("modules: {}\n")
		}

		// Parse both configs
		var mainConfig map[string]interface{}
		var defaultConfig map[string]interface{}

		if err := yaml.Unmarshal(mainConfigData, &mainConfig); err != nil {
			return fmt.Errorf("failed to parse main config: %w", err)
		}

		if err := yaml.Unmarshal(defaultConfigData, &defaultConfig); err != nil {
			return fmt.Errorf("failed to parse default config: %w", err)
		}

		// Merge the configs (user config takes precedence)
		mergedConfig := mergeConfigs(mainConfig, defaultConfig)

		// Write back the merged config
		mergedData, err := yaml.Marshal(mergedConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal merged config: %w", err)
		}

		if err := os.WriteFile(mainConfigPath, mergedData, 0644); err != nil {
			return fmt.Errorf("failed to write merged config: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully added default configuration for module '%s' to config.yaml\n", name)

		return nil
	},
}

var moduleRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an existing module folder and update go.work",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.TrimSpace(args[0])
		if err := validateModuleName(name); err != nil {
			return err
		}

		baseDir := strings.TrimSpace(modDirFlag)
		if baseDir == "" {
			baseDir = "modules"
		}
		target := filepath.Join(baseDir, name)

		if st, err := os.Stat(target); os.IsNotExist(err) {
			return fmt.Errorf("module directory does not exist: %s", target)
		} else if err != nil {
			return fmt.Errorf("error checking module directory: %w", err)
		} else if !st.IsDir() {
			return fmt.Errorf("path exists but is not a directory: %s", target)
		}

		if !modForceFlag {
			fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to remove module '%s' at '%s'? (y/N): ", name, target)
			var confirmation string
			fmt.Scanln(&confirmation)
			if strings.ToLower(confirmation) != "y" {
				fmt.Fprintf(cmd.OutOrStdout(), "Module removal cancelled.\n")
				return nil
			}
		}

		if err := os.RemoveAll(target); err != nil {
			return fmt.Errorf("failed to remove module directory %s: %w", target, err)
		}

		if err := syncGoWorkspace(); err != nil {
			return fmt.Errorf("failed to sync go workspace after module removal: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Module '%s' removed successfully.\n", name)
		return nil
	},
}

var moduleNameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

func validateModuleName(name string) error {
	if name == "" {
		return errors.New("module name required")
	}
	if strings.ContainsRune(name, os.PathSeparator) {
		return fmt.Errorf("module name must not contain path separators: %q", name)
	}
	if !moduleNameRe.MatchString(name) {
		return fmt.Errorf("invalid module name: %q", name)
	}
	return nil
}

// mergeConfigs recursively merges default config into main config
// User config (main) takes precedence over defaults
func mergeConfigs(main, defaults map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Start with defaults
	for key, defaultValue := range defaults {
		result[key] = defaultValue
	}

	// Override with main config values
	for key, mainValue := range main {
		if mainMap, mainIsMap := mainValue.(map[string]interface{}); mainIsMap {
			if defaultMap, defaultExists := result[key].(map[string]interface{}); defaultExists {
				// Recursively merge nested maps
				result[key] = mergeConfigs(mainMap, defaultMap)
			} else {
				// No default map, use main value
				result[key] = mainValue
			}
		} else {
			// Main value is not a map, use it directly (overrides default)
			result[key] = mainValue
		}
	}

	return result
}
