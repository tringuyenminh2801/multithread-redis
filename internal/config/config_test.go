package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg != Default() {
		t.Errorf("Load(nil) = %+v, want %+v", cfg, Default())
	}
}

func TestLoadEnvOverridesDefaults(t *testing.T) {
	t.Setenv("REDIS_POOL_SIZE", "42")
	t.Setenv("REDIS_PORT", ":7000")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.GoroutinePoolSize != 42 || cfg.BindedPort != ":7000" {
		t.Errorf("got pool size %d, port %q; want 42, \":7000\"", cfg.GoroutinePoolSize, cfg.BindedPort)
	}
}

func TestLoadFlagsOverrideEnv(t *testing.T) {
	t.Setenv("REDIS_POOL_SIZE", "42")

	cfg, err := Load([]string{"-pool-size", "7"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.GoroutinePoolSize != 7 {
		t.Errorf("pool size = %d, want 7", cfg.GoroutinePoolSize)
	}
}

func TestLoadRejectsInvalidInt(t *testing.T) {
	t.Setenv("REDIS_BUFFER_SIZE", "lots")

	if _, err := Load(nil); err == nil {
		t.Error("Load accepted a non-numeric REDIS_BUFFER_SIZE")
	}
}
