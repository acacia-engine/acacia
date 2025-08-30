package registry

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStoreAddRemoveModulesDedupSort(t *testing.T) {
	s := &Store{}
	if added := s.AddModule(ModuleEntry{Name: "b"}); !added {
		t.Fatal("expected add b")
	}
	if added := s.AddModule(ModuleEntry{Name: "a"}); !added {
		t.Fatal("expected add a")
	}
	if added := s.AddModule(ModuleEntry{Name: " a "}); added {
		t.Fatal("expected duplicate a not added")
	}
	if added := s.AddModule(ModuleEntry{Name: ""}); added {
		t.Fatal("expected empty not added")
	}
	if added := s.AddModule(ModuleEntry{Name: "a", Company: "c1", Version: "v1"}); !added {
		t.Fatal("expected add a/c1/v1")
	}
	if added := s.AddModule(ModuleEntry{Name: "a", Company: "c1", Version: "v1"}); added {
		t.Fatal("expected duplicate a/c1/v1 not added")
	}

	expect := []ModuleEntry{
		{Name: "a"},
		{Name: "b"},
		{Name: "a", Company: "c1", Version: "v1"},
	}
	if !reflect.DeepEqual(s.Modules, expect) {
		t.Fatalf("modules mismatch: got %+v want %+v", s.Modules, expect)
	}

	if removed := s.RemoveModule(ModuleEntry{Name: "a"}); !removed {
		t.Fatal("expected remove a")
	}
	if removed := s.RemoveModule(ModuleEntry{Name: "x"}); removed {
		t.Fatal("expected remove x to be false")
	}

	expect = []ModuleEntry{
		{Name: "b"},
		{Name: "a", Company: "c1", Version: "v1"},
	}
	if !reflect.DeepEqual(s.Modules, expect) {
		t.Fatalf("modules after remove mismatch: got %+v want %+v", s.Modules, expect)
	}
}

func TestStoreAddRemoveGatewaysDedupSort(t *testing.T) {
	s := &Store{}
	if added := s.AddGateway(GatewayEntry{Name: "g2"}); !added {
		t.Fatal("expected add g2")
	}
	if added := s.AddGateway(GatewayEntry{Name: "g1"}); !added {
		t.Fatal("expected add g1")
	}
	if added := s.AddGateway(GatewayEntry{Name: " g1 "}); added {
		t.Fatal("expected duplicate g1 not added")
	}
	if added := s.AddGateway(GatewayEntry{Name: ""}); added {
		t.Fatal("expected empty not added")
	}
	if added := s.AddGateway(GatewayEntry{Name: "g1", Company: "c1", Version: "v1"}); !added {
		t.Fatal("expected add g1/c1/v1")
	}
	if added := s.AddGateway(GatewayEntry{Name: "g1", Company: "c1", Version: "v1"}); added {
		t.Fatal("expected duplicate g1/c1/v1 not added")
	}

	expect := []GatewayEntry{
		{Name: "g1"},
		{Name: "g2"},
		{Name: "g1", Company: "c1", Version: "v1"},
	}
	if !reflect.DeepEqual(s.Gateways, expect) {
		t.Fatalf("gateways mismatch: got %+v want %+v", s.Gateways, expect)
	}

	if removed := s.RemoveGateway(GatewayEntry{Name: "g1"}); !removed {
		t.Fatal("expected remove g1")
	}
	if removed := s.RemoveGateway(GatewayEntry{Name: "x"}); removed {
		t.Fatal("expected remove x to be false")
	}

	expect = []GatewayEntry{
		{Name: "g2"},
		{Name: "g1", Company: "c1", Version: "v1"},
	}
	if !reflect.DeepEqual(s.Gateways, expect) {
		t.Fatalf("gateways after remove mismatch: got %+v want %+v", s.Gateways, expect)
	}
}

func TestLoadSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registries.json")

	s := &Store{
		Modules: []ModuleEntry{
			{Name: "m2", Company: "c2", Author: "a2", Version: "v2"},
			{Name: "m1", Company: "c1", Author: "a1", Version: "v1"},
			{Name: "m1", Company: "c1", Author: "a1", Version: "v1"}, // Duplicate
			{Name: ""}, // Empty name
		},
		Gateways: []GatewayEntry{
			{Name: "g2", Company: "c2", Author: "a2", Version: "v2"},
			{Name: "g1", Company: "c1", Author: "a1", Version: "v1"},
			{Name: "g1", Company: "c1", Author: "a1", Version: "v1"}, // Duplicate
			{Name: " "}, // Empty name after trim
		},
	}
	if err := Save(s, path); err != nil {
		t.Fatalf("save error: %v", err)
	}
	// file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat error: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	// dedup and sorted
	wantModules := []ModuleEntry{
		{Name: "m1", Company: "c1", Author: "a1", Version: "v1"},
		{Name: "m2", Company: "c2", Author: "a2", Version: "v2"},
	}
	if !reflect.DeepEqual(loaded.Modules, wantModules) {
		t.Fatalf("modules after load mismatch: got %+v want %+v", loaded.Modules, wantModules)
	}
	wantGateways := []GatewayEntry{
		{Name: "g1", Company: "c1", Author: "a1", Version: "v1"},
		{Name: "g2", Company: "c2", Author: "a2", Version: "v2"},
	}
	if !reflect.DeepEqual(loaded.Gateways, wantGateways) {
		t.Fatalf("gateways after load mismatch: got %+v want %+v", loaded.Gateways, wantGateways)
	}

	// Test RegistrySources
	s.RegistrySources = []RegistrySource{
		{URL: "http://example.com/repo2"},
		{URL: "http://example.com/repo1"},
		{URL: "http://example.com/repo1"}, // Duplicate
		{URL: " "},                        // Empty URL after trim
	}
	if err := Save(s, path); err != nil {
		t.Fatalf("save error for sources: %v", err)
	}

	loaded, err = Load(path)
	if err != nil {
		t.Fatalf("load error for sources: %v", err)
	}
	wantSources := []RegistrySource{
		{URL: "http://example.com/repo1"},
		{URL: "http://example.com/repo2"},
	}
	if !reflect.DeepEqual(loaded.RegistrySources, wantSources) {
		t.Fatalf("registry sources after load mismatch: got %+v want %+v", loaded.RegistrySources, wantSources)
	}
}

func TestStoreAddRemoveRegistrySource(t *testing.T) {
	s := &Store{}
	if added := s.AddRegistrySource(RegistrySource{URL: "http://example.com/repo2"}); !added {
		t.Fatal("expected add repo2")
	}
	if added := s.AddRegistrySource(RegistrySource{URL: "http://example.com/repo1"}); !added {
		t.Fatal("expected add repo1")
	}
	if added := s.AddRegistrySource(RegistrySource{URL: " http://example.com/repo1 "}); added {
		t.Fatal("expected duplicate repo1 not added")
	}
	if added := s.AddRegistrySource(RegistrySource{URL: ""}); added {
		t.Fatal("expected empty not added")
	}

	expect := []RegistrySource{
		{URL: "http://example.com/repo1"},
		{URL: "http://example.com/repo2"},
	}
	if !reflect.DeepEqual(s.RegistrySources, expect) {
		t.Fatalf("registry sources mismatch: got %+v want %+v", s.RegistrySources, expect)
	}

	if removed := s.RemoveRegistrySource(RegistrySource{URL: "http://example.com/repo1"}); !removed {
		t.Fatal("expected remove repo1")
	}
	if removed := s.RemoveRegistrySource(RegistrySource{URL: "http://example.com/repoX"}); removed {
		t.Fatal("expected remove repoX to be false")
	}

	expect = []RegistrySource{
		{URL: "http://example.com/repo2"},
	}
	if !reflect.DeepEqual(s.RegistrySources, expect) {
		t.Fatalf("registry sources after remove mismatch: got %+v want %+v", s.RegistrySources, expect)
	}
}
