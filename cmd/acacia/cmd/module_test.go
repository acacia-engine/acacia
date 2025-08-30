package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestModuleCreateScaffold(t *testing.T) {
	t.Setenv("ACACIA_CONFIG_DIR", "") // not needed here, but keep env clean
	temp := t.TempDir()
	// Configure flags for this run
	if err := moduleCreateCmd.Flags().Set("dir", temp); err != nil {
		t.Fatalf("set dir flag: %v", err)
	}
	if err := moduleCreateCmd.Flags().Set("version", "1.2.3"); err != nil {
		t.Fatalf("set version flag: %v", err)
	}
	if err := moduleCreateCmd.Flags().Set("description", "Sample module"); err != nil {
		t.Fatalf("set description flag: %v", err)
	}

	name := "mymod"
	if err := moduleCreateCmd.RunE(moduleCreateCmd, []string{name}); err != nil {
		t.Fatalf("run create: %v", err)
	}

	target := filepath.Join(temp, name)
	// Check directories
	for _, d := range []string{"application", "domain", "infrastructure", "docs"} {
		p := filepath.Join(target, d)
		st, err := os.Stat(p)
		if err != nil || !st.IsDir() {
			t.Fatalf("expected directory: %s (err=%v)", p, err)
		}
	}
	// Check registry.json
	b, err := os.ReadFile(filepath.Join(target, "registry.json"))
	if err != nil {
		t.Fatalf("read registry.json: %v", err)
	}
	var reg struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(b, &reg); err != nil {
		t.Fatalf("unmarshal registry: %v", err)
	}
	if reg.Name != name || reg.Version != "1.2.3" || reg.Description != "Sample module" {
		t.Fatalf("unexpected registry content: %+v", reg)
	}
	// Check docs README
	if _, err := os.Stat(filepath.Join(target, "docs", "README.md")); err != nil {
		t.Fatalf("docs README missing: %v", err)
	}
}
