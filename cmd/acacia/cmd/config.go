package cmd

import (
	"acacia/core/config"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configGenerateCmd)

	configGenerateCmd.Flags().Bool("from-modules", false, "Generate complete config from all module defaults")
	configGenerateCmd.Flags().Bool("update-modules", false, "Regenerate default configs for existing modules")
	configGenerateCmd.Flags().Bool("minimal", false, "Create minimal config with essential settings")
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
		fmt.Println("Configuration is valid.")
		return nil
	},
}

var configGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate configuration files",
	Long: `Generate configuration files with various options:

acacia config generate --from-modules    Generate complete config from all module defaults
acacia config generate --update-modules  Regenerate default configs for existing modules
acacia config generate --minimal         Create minimal config with essential settings`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fromModules, _ := cmd.Flags().GetBool("from-modules")
		updateModules, _ := cmd.Flags().GetBool("update-modules")
		minimal, _ := cmd.Flags().GetBool("minimal")

		if fromModules {
			return generateFromModules(cmd)
		} else if updateModules {
			return updateModuleConfigs(cmd)
		} else if minimal {
			return generateMinimalConfig(cmd)
		} else {
			return fmt.Errorf("specify one of: --from-modules, --update-modules, or --minimal")
		}
	},
}

func generateFromModules(cmd *cobra.Command) error {
	fmt.Println("Generating configuration from all module defaults...")

	// This will scan all modules and merge their defaults
	cfg, err := config.GenerateFromModules("modules")
	if err != nil {
		return fmt.Errorf("failed to generate config from modules: %w", err)
	}

	// Save the generated config
	err = config.SaveGeneratedConfig(cfg, "config.yaml")
	if err != nil {
		return fmt.Errorf("failed to save generated config: %w", err)
	}

	fmt.Println("Configuration generated successfully from module defaults.")
	return nil
}

func updateModuleConfigs(cmd *cobra.Command) error {
	fmt.Println("Regenerating default configurations for existing modules...")

	err := config.UpdateModuleDefaults("modules")
	if err != nil {
		return fmt.Errorf("failed to update module defaults: %w", err)
	}

	fmt.Println("Module default configurations updated successfully.")
	return nil
}

func generateMinimalConfig(cmd *cobra.Command) error {
	fmt.Println("Generating minimal configuration...")

	cfg := config.GenerateMinimalConfig()

	err := config.SaveGeneratedConfig(cfg, "config.yaml")
	if err != nil {
		return fmt.Errorf("failed to save minimal config: %w", err)
	}

	fmt.Println("Minimal configuration generated successfully.")
	return nil
}
