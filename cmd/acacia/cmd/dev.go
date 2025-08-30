package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(devCmd)
	devCmd.AddCommand(devBuildCmd)
	devCmd.AddCommand(devTidyCmd)
	devCmd.AddCommand(devTestCmd)
	devCmd.AddCommand(devAllCmd)
}

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Development utilities (build, tidy, test)",
	Long:  "Provides commands to streamline common Go development tasks for Acacia.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default behavior for 'acacia dev' is to run 'acacia dev all'
		return devAllCmd.RunE(cmd, args)
	},
}

var devBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the Acacia CLI executable",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.OutOrStdout(), "Building Acacia CLI...")
		c := exec.Command("go", "build", "./cmd/acacia")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var devTidyCmd = &cobra.Command{
	Use:   "tidy",
	Short: "Run go mod tidy to clean up dependencies",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.OutOrStdout(), "Running go mod tidy...")
		c := exec.Command("go", "mod", "tidy")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var devTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Run all Go tests in the project",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.OutOrStdout(), "Running Go tests...")
		c := exec.Command("go", "test", "./...")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var devAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Run build, tidy, and test in sequence",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := devBuildCmd.RunE(cmd, args); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}
		if err := devTidyCmd.RunE(cmd, args); err != nil {
			return fmt.Errorf("tidy failed: %w", err)
		}
		if err := devTestCmd.RunE(cmd, args); err != nil {
			return fmt.Errorf("tests failed: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "All development tasks completed successfully.")
		return nil
	},
}
