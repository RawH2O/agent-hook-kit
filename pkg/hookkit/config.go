package hookkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	ConfigFileName       = ".agent-hook-kit.json"
	NestedConfigFileName = ".agent-hook-kit/config.json"
)

// Config is deliberately project-local. A project chooses rules by ID, while
// rule implementation remains completely unaware of the project path/name.
type Config struct {
	Rules   []RuleSelection `json:"rules"`
	OnError string          `json:"on_error,omitempty"`
}

// RuleSelection accepts either "rule-id" or an object with options.
type RuleSelection struct {
	ID      string         `json:"id"`
	Options map[string]any `json:"options,omitempty"`
}

func (s *RuleSelection) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return fmt.Errorf("empty rule selection")
	}
	if data[0] == '"' {
		if err := json.Unmarshal(data, &s.ID); err != nil {
			return fmt.Errorf("decode rule ID: %w", err)
		}
		return nil
	}
	type selection RuleSelection
	var decoded selection
	if err := json.Unmarshal(data, &decoded); err != nil {
		return fmt.Errorf("decode rule selection: %w", err)
	}
	*s = RuleSelection(decoded)
	return nil
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	for index, selection := range config.Rules {
		if selection.ID == "" {
			return Config{}, fmt.Errorf("config %s: rules[%d] has an empty ID", path, index)
		}
	}
	return config, nil
}

// DiscoverConfig walks upward from start. No config is a valid configuration
// with zero selected rules; this prevents every compiled rule from running in
// every project by accident.
func DiscoverConfig(start string) (Config, string, error) {
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return Config{}, "", fmt.Errorf("get working directory: %w", err)
		}
	}
	start, err := filepath.Abs(start)
	if err != nil {
		return Config{}, "", fmt.Errorf("resolve working directory %s: %w", start, err)
	}
	if info, statErr := os.Stat(start); statErr == nil && !info.IsDir() {
		start = filepath.Dir(start)
	}

	for dir := start; ; dir = filepath.Dir(dir) {
		candidates := []string{
			filepath.Join(dir, ConfigFileName),
			filepath.Join(dir, NestedConfigFileName),
		}
		for _, candidate := range candidates {
			if _, statErr := os.Stat(candidate); statErr == nil {
				config, loadErr := LoadConfig(candidate)
				return config, candidate, loadErr
			} else if !os.IsNotExist(statErr) {
				return Config{}, "", fmt.Errorf("inspect config %s: %w", candidate, statErr)
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return Config{}, "", nil
}
