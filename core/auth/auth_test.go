package auth_test

import (
	"context"
	"testing"

	"acacia/core/auth"
)

func TestPrincipalFromContext(t *testing.T) {
	tests := []struct {
		name     string
		ctxFunc  func() context.Context
		wantNil  bool
		wantID   string
		wantType string
	}{
		{
			name:    "no principal in context",
			ctxFunc: func() context.Context { return context.Background() },
			wantNil: true,
		},
		{
			name: "principal in context",
			ctxFunc: func() context.Context {
				p := auth.NewDefaultPrincipal("test", "user", []string{"admin"})
				return auth.ContextWithPrincipal(context.Background(), p)
			},
			wantNil:  false,
			wantID:   "test",
			wantType: "user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := auth.PrincipalFromContext(tt.ctxFunc())
			if tt.wantNil {
				if got != nil {
					t.Errorf("PrincipalFromContext() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Error("PrincipalFromContext() = nil, want principal")
				return
			}
			if got.ID() != tt.wantID || got.Type() != tt.wantType {
				t.Errorf("PrincipalFromContext() got ID=%q type=%q, want ID=%q type=%q", got.ID(), got.Type(), tt.wantID, tt.wantType)
			}
		})
	}
}

func TestDefaultPrincipal(t *testing.T) {
	p := auth.NewDefaultPrincipal("123", "user", []string{"admin", "user"})

	if p.ID() != "123" {
		t.Errorf("ID() = %v, want 123", p.ID())
	}
	if p.Type() != "user" {
		t.Errorf("Type() = %v, want user", p.Type())
	}
	roles := p.Roles()
	if len(roles) != 2 || roles[0] != "admin" || roles[1] != "user" {
		t.Errorf("Roles() = %v, want [admin user]", roles)
	}
}

func TestDefaultAccessController_NoConfigProvider(t *testing.T) {
	var configProvider auth.RBACProvider
	ctrl := auth.NewDefaultAccessController(configProvider)

	ctx := context.Background()
	dummyPrincipal := auth.NewDefaultPrincipal("test", "user", []string{"admin"})

	// Should always return true for allowAll behavior when no provider
	if !ctrl.CanLog(ctx, dummyPrincipal) {
		t.Error("CanLog should return true for allowAll controller")
	}
	if !ctrl.CanAccessMetrics(ctx, dummyPrincipal) {
		t.Error("CanAccessMetrics should return true for allowAll controller")
	}
	if !ctrl.HasPermission(dummyPrincipal, "any") {
		t.Error("HasPermission should return true for allowAll controller")
	}
}

func TestAllowAllAccessController(t *testing.T) {
	ctrl := auth.NewDefaultAccessController(nil) // nil provider creates allowAll

	ctx := context.Background()
	dummyPrincipal := auth.NewDefaultPrincipal("test", "user", []string{})

	if !ctrl.CanLog(ctx, dummyPrincipal) {
		t.Error("CanLog should return true")
	}
	if !ctrl.CanAccessMetrics(ctx, dummyPrincipal) {
		t.Error("CanAccessMetrics should return true")
	}
	if !ctrl.HasPermission(dummyPrincipal, "any") {
		t.Error("HasPermission should return true")
	}
}
