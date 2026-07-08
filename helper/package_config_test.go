package helper

import (
	"testing"
)

func TestNewPackageLoadConfigDisablesVCSStamps(t *testing.T) {
	cfg := NewPackageLoadConfig("/tmp/example", true)

	if cfg == nil {
		t.Fatal("NewPackageLoadConfig() returned nil")
	}

	if !cfg.Tests {
		t.Fatal("NewPackageLoadConfig() should enable test package loading")
	}

	if cfg.Dir != "/tmp/example" {
		t.Fatalf("NewPackageLoadConfig() Dir = %q, want %q", cfg.Dir, "/tmp/example")
	}

	if len(cfg.BuildFlags) != 1 || cfg.BuildFlags[0] != "-buildvcs=false" {
		t.Fatalf("NewPackageLoadConfig() BuildFlags = %v, want [-buildvcs=false]", cfg.BuildFlags)
	}
}
