package kernel_test

import (
	"acacia/core/auth"
	"acacia/core/config"
	"acacia/core/kernel"
	"context"
	"strings"
	"testing"
)

func TestKernel_Security_RequiresPrincipal(t *testing.T) {
	krn := kernel.New(&config.Config{}, nil)

	// Test AddModule without principal should fail
	err := krn.AddModule(context.Background(), &recModule{name: "test", rec: &recorder{}})
	if err == nil {
		t.Fatal("AddModule should fail without principal")
	}
	if !strings.Contains(err.Error(), "security violation") {
		t.Fatalf("Expected security violation error, got: %v", err)
	}

	// Test AddModule with principal should succeed
	ctx := auth.ContextWithPrincipal(context.Background(), &testPrincipal{
		id: "test-user", pType: "system", roles: []string{"kernel.module.*"},
	})
	err = krn.AddModule(ctx, &recModule{name: "test", rec: &recorder{}})
	if err != nil {
		t.Fatalf("AddModule should succeed with proper principal: %v", err)
	}
}

// Mock access controller that denies module permissions
type denyModuleAccessController struct{}

func (d *denyModuleAccessController) CanLog(ctx context.Context, p auth.Principal) bool { return true }
func (d *denyModuleAccessController) CanAccessMetrics(ctx context.Context, p auth.Principal) bool {
	return true
}
func (d *denyModuleAccessController) CanPublishEvent(ctx context.Context, p auth.Principal, eventType string) bool {
	return true
}
func (d *denyModuleAccessController) CanSubscribeEvent(ctx context.Context, p auth.Principal, eventType string) bool {
	return true
}
func (d *denyModuleAccessController) CanAccessConfig(ctx context.Context, p auth.Principal, configKey string) bool {
	return true
}
func (d *denyModuleAccessController) CanReloadModule(ctx context.Context, p auth.Principal, moduleToReload string) bool {
	return true
}
func (d *denyModuleAccessController) HasPermission(p auth.Principal, perm auth.Permission) bool {
	// Deny all kernel module permissions
	if strings.HasPrefix(string(perm), "kernel.module.") {
		return false
	}
	return true
}

func TestKernel_Security_InsufficientPermissions(t *testing.T) {
	// Use a custom access controller that denies module permissions
	krn := kernel.New(&config.Config{}, &denyModuleAccessController{})

	// Test with principal that doesn't have required permissions
	ctx := auth.ContextWithPrincipal(context.Background(), &testPrincipal{
		id: "test-user", pType: "user", roles: []string{"read-only"},
	})
	err := krn.AddModule(ctx, &recModule{name: "test", rec: &recorder{}})
	if err == nil {
		t.Fatal("AddModule should fail with insufficient permissions")
	}
	if !strings.Contains(err.Error(), "access denied") {
		t.Fatalf("Expected access denied error, got: %v", err)
	}
}

func TestKernel_Security_EnableDisablePermissions(t *testing.T) {
	krn := kernel.New(&config.Config{}, nil)

	// First add a module with proper permissions
	ctx := auth.ContextWithPrincipal(context.Background(), &testPrincipal{
		id: "admin", pType: "system", roles: []string{"kernel.module.*"},
	})
	testModule := &recModule{name: "test-module", rec: &recorder{}}
	err := krn.AddModule(ctx, testModule)
	if err != nil {
		t.Fatalf("AddModule should succeed: %v", err)
	}

	// Test EnableModule without permissions should fail
	err = krn.EnableModule(context.Background(), "test-module")
	if err == nil {
		t.Fatal("EnableModule should fail without principal")
	}

	// Test EnableModule with permissions should succeed
	err = krn.EnableModule(ctx, "test-module")
	if err != nil {
		t.Fatalf("EnableModule should succeed with permissions: %v", err)
	}

	// Test DisableModule without permissions should fail
	err = krn.DisableModule(context.Background(), "test-module")
	if err == nil {
		t.Fatal("DisableModule should fail without principal")
	}

	// Test DisableModule with permissions should succeed
	err = krn.DisableModule(ctx, "test-module")
	if err != nil {
		t.Fatalf("DisableModule should succeed with permissions: %v", err)
	}
}
