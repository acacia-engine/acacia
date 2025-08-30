package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Store holds persistent lists of registered modules and gateways.
// It is a simple JSON-backed store intended for CLI management.
// The meaning of these entries is project-defined; here we store names.

type ModuleEntry struct {
	Name      string `json:"name"`
	Company   string `json:"company,omitempty"`
	Author    string `json:"author,omitempty"`
	Version   string `json:"version,omitempty"`
	SourceURL string `json:"sourceURL,omitempty"` // URL to the Git repository of the module
}

type GatewayEntry struct {
	Name      string `json:"name"`
	Company   string `json:"company,omitempty"`
	Author    string `json:"author,omitempty"`
	Version   string `json:"version,omitempty"`
	SourceURL string `json:"sourceURL,omitempty"` // URL to the Git repository of the gateway
}

type RegistrySource struct {
	URL string `json:"url"` // URL to the Git repository acting as a registry
}

type Store struct {
	Modules         []ModuleEntry    `json:"modules"`
	Gateways        []GatewayEntry   `json:"gateways"`
	RegistrySources []RegistrySource `json:"registrySources,omitempty"`
}

// DefaultPath returns the default path to the registry JSON file.
// Priority:
// 1) ACACIA_CONFIG_DIR env var
// 2) os.UserConfigDir()/acacia
func DefaultPath() (string, error) {
	if dir := os.Getenv("ACACIA_CONFIG_DIR"); strings.TrimSpace(dir) != "" {
		return filepath.Join(dir, "registries.json"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "acacia", "registries.json"), nil
}

// Load reads the store from the given path. If the file doesn't exist, returns an empty store.
func Load(path string) (*Store, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Store{}, nil
		}
		return nil, fmt.Errorf("read registry: %w", err)
	}
	var s Store
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("unmarshal registry: %w", err)
	}
	s.dedupeAndSort()
	return &s, nil
}

// Save writes the store to the given path, creating parent directories if needed.
func Save(s *Store, path string) error {
	if s == nil {
		return fmt.Errorf("store is nil")
	}
	s.dedupeAndSort()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write registry: %w", err)
	}
	return nil
}

func (s *Store) dedupeAndSort() {
	if s.Modules == nil {
		s.Modules = []ModuleEntry{}
	}
	if s.Gateways == nil {
		s.Gateways = []GatewayEntry{}
	}
	if s.RegistrySources == nil {
		s.RegistrySources = []RegistrySource{}
	}
	s.Modules = uniqueSortedModules(s.Modules)
	s.Gateways = uniqueSortedGateways(s.Gateways)
	s.RegistrySources = uniqueSortedRegistrySources(s.RegistrySources)
}

func uniqueSortedModules(in []ModuleEntry) []ModuleEntry {
	m := map[string]ModuleEntry{}
	for _, v := range in {
		v.Name = strings.TrimSpace(v.Name)
		if v.Name == "" {
			continue
		}
		key := fmt.Sprintf("%s/%s/%s/%s", v.Company, v.Name, v.Author, v.Version)
		m[key] = v
	}
	out := make([]ModuleEntry, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Company != out[j].Company {
			return out[i].Company < out[j].Company
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		if out[i].Author != out[j].Author {
			return out[i].Author < out[j].Author
		}
		return out[i].Version < out[j].Version
	})
	return out
}

func uniqueSortedGateways(in []GatewayEntry) []GatewayEntry {
	m := map[string]GatewayEntry{}
	for _, v := range in {
		v.Name = strings.TrimSpace(v.Name)
		if v.Name == "" {
			continue
		}
		key := fmt.Sprintf("%s/%s/%s/%s", v.Company, v.Name, v.Author, v.Version)
		m[key] = v
	}
	out := make([]GatewayEntry, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Company != out[j].Company {
			return out[i].Company < out[j].Company
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		if out[i].Author != out[j].Author {
			return out[i].Author < out[j].Author
		}
		return out[i].Version < out[j].Version
	})
	return out
}

func (s *Store) AddModule(entry ModuleEntry) bool {
	entry.Name = strings.TrimSpace(entry.Name)
	if entry.Name == "" {
		return false
	}
	for _, e := range s.Modules {
		if e.Name == entry.Name && e.Company == entry.Company && e.Author == entry.Author && e.Version == entry.Version {
			return false
		}
	}
	s.Modules = append(s.Modules, entry)
	s.Modules = uniqueSortedModules(s.Modules)
	return true
}

func (s *Store) RemoveModule(entry ModuleEntry) bool {
	entry.Name = strings.TrimSpace(entry.Name)
	var out []ModuleEntry
	removed := false
	for _, e := range s.Modules {
		if e.Name == entry.Name && e.Company == entry.Company && e.Author == entry.Author && e.Version == entry.Version {
			removed = true
			continue
		}
		out = append(out, e)
	}
	s.Modules = out
	return removed
}

func (s *Store) AddGateway(entry GatewayEntry) bool {
	entry.Name = strings.TrimSpace(entry.Name)
	if entry.Name == "" {
		return false
	}
	for _, e := range s.Gateways {
		if e.Name == entry.Name && e.Company == entry.Company && e.Author == entry.Author && e.Version == entry.Version {
			return false
		}
	}
	s.Gateways = append(s.Gateways, entry)
	s.Gateways = uniqueSortedGateways(s.Gateways)
	return true
}

func (s *Store) RemoveGateway(entry GatewayEntry) bool {
	entry.Name = strings.TrimSpace(entry.Name)
	var out []GatewayEntry
	removed := false
	for _, e := range s.Gateways {
		if e.Name == entry.Name && e.Company == entry.Company && e.Author == entry.Author && e.Version == entry.Version && e.SourceURL == entry.SourceURL {
			removed = true
			continue
		}
		out = append(out, e)
	}
	s.Gateways = out
	return removed
}

func uniqueSortedRegistrySources(in []RegistrySource) []RegistrySource {
	m := map[string]RegistrySource{}
	for _, v := range in {
		v.URL = strings.TrimSpace(v.URL)
		if v.URL == "" {
			continue
		}
		m[v.URL] = v
	}
	out := make([]RegistrySource, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].URL < out[j].URL
	})
	return out
}

func (s *Store) AddRegistrySource(entry RegistrySource) bool {
	entry.URL = strings.TrimSpace(entry.URL)
	if entry.URL == "" {
		return false
	}
	for _, e := range s.RegistrySources {
		if e.URL == entry.URL {
			return false
		}
	}
	s.RegistrySources = append(s.RegistrySources, entry)
	s.RegistrySources = uniqueSortedRegistrySources(s.RegistrySources)
	return true
}

func (s *Store) RemoveRegistrySource(entry RegistrySource) bool {
	entry.URL = strings.TrimSpace(entry.URL)
	var out []RegistrySource
	removed := false
	for _, e := range s.RegistrySources {
		if e.URL == entry.URL {
			removed = true
			continue
		}
		out = append(out, e)
	}
	s.RegistrySources = out
	return removed
}
