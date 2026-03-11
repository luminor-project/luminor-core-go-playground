package config

import "testing"

func TestLoad_DefaultsAreValid(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load with defaults failed: %v", err)
	}
	if cfg.AgentWorkloadMode != "fake" {
		t.Errorf("expected default AgentWorkloadMode 'fake', got %q", cfg.AgentWorkloadMode)
	}
	if cfg.AppEnv != "development" {
		t.Errorf("expected default AppEnv 'development', got %q", cfg.AppEnv)
	}
}

func TestLoad_InvalidAgentWorkloadMode(t *testing.T) {
	t.Setenv("AGENT_WORKLOAD_MODE", "invalid")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid AGENT_WORKLOAD_MODE")
	}
}

func TestLoad_AgentWorkloadModeLive(t *testing.T) {
	t.Setenv("AGENT_WORKLOAD_MODE", "live")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load with mode 'live' failed: %v", err)
	}
	if cfg.AgentWorkloadMode != "live" {
		t.Errorf("expected AgentWorkloadMode 'live', got %q", cfg.AgentWorkloadMode)
	}
}

func TestIsProduction(t *testing.T) {
	cfg := Config{AppEnv: "production"}
	if !cfg.IsProduction() {
		t.Error("expected IsProduction() true")
	}
	if cfg.IsDevelopment() {
		t.Error("expected IsDevelopment() false")
	}
}

func TestIsDevelopment(t *testing.T) {
	cfg := Config{AppEnv: "development"}
	if !cfg.IsDevelopment() {
		t.Error("expected IsDevelopment() true")
	}
	if cfg.IsProduction() {
		t.Error("expected IsProduction() false")
	}
}
