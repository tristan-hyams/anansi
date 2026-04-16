package weaver

import (
	"testing"
	"time"
)

func validConfig() *WeaverConfig {
	return &WeaverConfig{
		Workers: 2,
		Rate:    5.0,
		MaxDepth: 3,
		Timeout: 10 * time.Second,
	}
}

func TestValidate_HappyPath(t *testing.T) {
	cfg := validConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidate_WorkersZero(t *testing.T) {
	cfg := validConfig()
	cfg.Workers = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for workers=0")
	}
}

func TestValidate_WorkersNegative(t *testing.T) {
	cfg := validConfig()
	cfg.Workers = -1
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for negative workers")
	}
}

func TestValidate_RateZero(t *testing.T) {
	cfg := validConfig()
	cfg.Rate = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for rate=0")
	}
}

func TestValidate_RateNegative(t *testing.T) {
	cfg := validConfig()
	cfg.Rate = -1
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for negative rate")
	}
}

func TestValidate_MaxDepthNegative(t *testing.T) {
	cfg := validConfig()
	cfg.MaxDepth = -1
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for negative max-depth")
	}
}

func TestValidate_MaxDepthZero_Unlimited(t *testing.T) {
	cfg := validConfig()
	cfg.MaxDepth = 0
	if err := cfg.Validate(); err != nil {
		t.Fatalf("max-depth=0 should be valid (unlimited), got %v", err)
	}
}

func TestValidate_TimeoutZero(t *testing.T) {
	cfg := validConfig()
	cfg.Timeout = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for timeout=0")
	}
}

func TestValidate_TimeoutNegative(t *testing.T) {
	cfg := validConfig()
	cfg.Timeout = -1 * time.Second
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for negative timeout")
	}
}

func TestValidate_DefaultsUserAgent(t *testing.T) {
	cfg := validConfig()
	cfg.UserAgent = ""
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.UserAgent != defaultUserAgent {
		t.Fatalf("expected default user agent %q, got %q", defaultUserAgent, cfg.UserAgent)
	}
}

func TestValidate_PreservesUserAgent(t *testing.T) {
	cfg := validConfig()
	cfg.UserAgent = "CustomBot"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.UserAgent != "CustomBot" {
		t.Fatalf("expected CustomBot, got %q", cfg.UserAgent)
	}
}

func TestValidate_DefaultsProgressInterval(t *testing.T) {
	cfg := validConfig()
	cfg.ProgressInterval = 0
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ProgressInterval != defaultProgressInterval {
		t.Fatalf("expected default progress interval %d, got %d", defaultProgressInterval, cfg.ProgressInterval)
	}
}

func TestValidate_PreservesProgressInterval(t *testing.T) {
	cfg := validConfig()
	cfg.ProgressInterval = 500
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ProgressInterval != 500 {
		t.Fatalf("expected 500, got %d", cfg.ProgressInterval)
	}
}
