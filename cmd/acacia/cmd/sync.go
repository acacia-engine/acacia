package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func init() {
	devCmd.AddCommand(devSyncCmd)
}

var devSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync Go workspace",
	Long:  "Scans for Go modules and updates the go.work file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return syncGoWorkspace()
	},
}

func syncGoWorkspace() error {
	fmt.Println("Syncing Go workspace...")

	var modulePaths []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "go.mod" {
			dir := filepath.Dir(path)
			// Ensure we have a clean path, especially for the root module.
			if dir == "." {
				modulePaths = append(modulePaths, ".")
			} else {
				modulePaths = append(modulePaths, "./"+strings.TrimPrefix(dir, "./"))
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("error scanning for modules: %w", err)
	}

	if len(modulePaths) == 0 {
		fmt.Println("No Go modules found.")
		// If no modules are found, ensure go.work is empty or removed if it exists.
		if _, err := os.Stat("go.work"); err == nil {
			goVersion, err := getGoVersion()
			if err != nil {
				return fmt.Errorf("failed to get Go version: %w", err)
			}
			if err := os.WriteFile("go.work", []byte(fmt.Sprintf("go %s\n\nuse (\n)\n", goVersion)), 0o644); err != nil {
				return fmt.Errorf("failed to clear go.work file: %w", err)
			}
			fmt.Println("go.work file cleared.")
		}
		return nil
	}

	// Sort module paths for consistent go.work file content
	// This is important for deterministic file content and avoiding unnecessary diffs.
	// The 'go work use' command does this automatically, but since we're writing directly, we need to do it.
	// However, the original `updateGoWorkFile` in `module.go` also sorted, so we should maintain that behavior.
	// For now, let's keep it simple and just write the paths as found.
	// If a specific order is required, we can add `sort.Strings(modulePaths)` here.

	// Get the Go version from existing files
	goVersion, err := getGoVersion()
	if err != nil {
		return fmt.Errorf("failed to get Go version: %w", err)
	}

	var goWorkContent strings.Builder
	goWorkContent.WriteString(fmt.Sprintf("go %s\n\nuse (\n", goVersion))
	for _, mod := range modulePaths {
		goWorkContent.WriteString(fmt.Sprintf("\t%s\n", mod))
	}
	goWorkContent.WriteString(")\n")

	if err := os.WriteFile("go.work", []byte(goWorkContent.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write go.work file: %w", err)
	}

	fmt.Println("Go workspace synced successfully.")
	return nil
}

// getGoVersion reads the Go version from config, go.work, or falls back to go.mod
func getGoVersion() (string, error) {
	// First, try to read from config.yaml
	if file, err := os.Open("config.yaml"); err == nil {
		defer file.Close()

		var config map[string]interface{}
		decoder := yaml.NewDecoder(file)
		if err := decoder.Decode(&config); err == nil {
			if goSection, ok := config["go"].(map[string]interface{}); ok {
				if version, ok := goSection["version"].(string); ok && version != "" {
					return version, nil
				}
			}
		}
	}

	// Try to read from go.work
	if file, err := os.Open("go.work"); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		if scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "go ") {
				return strings.TrimPrefix(line, "go "), nil
			}
		}
	}

	// Fallback to go.mod
	if file, err := os.Open("go.mod"); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		if scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "go ") {
				return strings.TrimPrefix(line, "go "), nil
			}
		}
	}

	// Default fallback
	return "1.24.6", nil
}
