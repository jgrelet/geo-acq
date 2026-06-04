package config

import (
	"path/filepath"
	"testing"
)

func TestBackupPathDefaultsToMissionNameBesideConfig(t *testing.T) {
	cfg := Config{
		Mission: Mission{Name: "Test Mission"},
		Backup:  Backup{Raw: true, Processed: true},
	}

	rawWant := filepath.Join("examples", "Test-Mission-raw.sqlite")
	if got := cfg.RawBackupPath(filepath.Join("examples", "listener.toml")); got != rawWant {
		t.Fatalf("RawBackupPath() = %q, want %q", got, rawWant)
	}

	processedWant := filepath.Join("examples", "Test-Mission-data.sqlite")
	if got := cfg.ProcessedBackupPath(filepath.Join("examples", "listener.toml")); got != processedWant {
		t.Fatalf("ProcessedBackupPath() = %q, want %q", got, processedWant)
	}
}

func TestBackupPathUsesCustomRelativePathsBesideConfig(t *testing.T) {
	cfg := Config{
		Mission: Mission{Name: "Test Mission"},
		Backup: Backup{
			Raw:           true,
			Processed:     true,
			RawPath:       filepath.Join("db", "navigation.sqlite"),
			ProcessedPath: filepath.Join("db", "measurements.sqlite"),
		},
	}

	rawWant := filepath.Join("examples", "db", "navigation.sqlite")
	if got := cfg.RawBackupPath(filepath.Join("examples", "listener.toml")); got != rawWant {
		t.Fatalf("RawBackupPath() = %q, want %q", got, rawWant)
	}

	processedWant := filepath.Join("examples", "db", "measurements.sqlite")
	if got := cfg.ProcessedBackupPath(filepath.Join("examples", "listener.toml")); got != processedWant {
		t.Fatalf("ProcessedBackupPath() = %q, want %q", got, processedWant)
	}
}

func TestBackupPathUsesCustomRelativeDirectory(t *testing.T) {
	root := t.TempDir()
	cfg := Config{
		Mission: Mission{Name: "Test Mission"},
		Backup: Backup{
			Raw:           true,
			Processed:     true,
			RawPath:       "db",
			ProcessedPath: "db/",
		},
	}

	rawWant := filepath.Join(root, "db", "Test-Mission-raw.sqlite")
	if got := cfg.RawBackupPath(filepath.Join(root, "listener.toml")); got != rawWant {
		t.Fatalf("RawBackupPath() = %q, want %q", got, rawWant)
	}

	processedWant := filepath.Join(root, "db", "Test-Mission-data.sqlite")
	if got := cfg.ProcessedBackupPath(filepath.Join(root, "listener.toml")); got != processedWant {
		t.Fatalf("ProcessedBackupPath() = %q, want %q", got, processedWant)
	}
}

func TestBackupPathUsesCustomAbsolutePathAsIs(t *testing.T) {
	rawPath := filepath.Join(t.TempDir(), "navigation.sqlite")
	cfg := Config{
		Mission: Mission{Name: "Test Mission"},
		Backup:  Backup{Raw: true, RawPath: rawPath},
	}

	if got := cfg.RawBackupPath(filepath.Join("examples", "listener.toml")); got != filepath.Clean(rawPath) {
		t.Fatalf("RawBackupPath() = %q, want %q", got, rawPath)
	}
}
