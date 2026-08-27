package hookkit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverConfigWalksUpAndSupportsStringSelections(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "src", "pkg")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(`{
  "rules": ["test/first", {"id":"test/second", "options":{"note":"notes"}}]
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	config, found, err := DiscoverConfig(nested)
	if err != nil {
		t.Fatal(err)
	}
	if found != configPath {
		t.Fatalf("found %q, want %q", found, configPath)
	}
	if len(config.Rules) != 2 {
		t.Fatalf("got %d rules, want 2", len(config.Rules))
	}
	if config.Rules[0].ID != "test/first" {
		t.Fatalf("got first rule %q", config.Rules[0].ID)
	}
	if got := config.Rules[1].Options["note"]; got != "notes" {
		t.Fatalf("got option %v, want notes", got)
	}
}

func TestDiscoverConfigReturnsEmptySelectionWhenMissing(t *testing.T) {
	config, found, err := DiscoverConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if found != "" {
		t.Fatalf("found unexpected config %q", found)
	}
	if len(config.Rules) != 0 {
		t.Fatalf("got %d rules, want 0", len(config.Rules))
	}
}
