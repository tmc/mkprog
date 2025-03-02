package review

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the configuration for the code review tool
type Config struct {
	// SeverityThreshold is the minimum severity level to report (info, warning, error)
	SeverityThreshold string `yaml:"severity_threshold"`

	// Ignore is a list of patterns to ignore
	Ignore []string `yaml:"ignore"`

	// Include is a list of patterns to include
	Include []string `yaml:"include"`

	// Exclude is a list of patterns to exclude
	Exclude []string `yaml:"exclude"`

	// Filters is a list of rule categories to include
	Filters []string `yaml:"filters"`

	// Rules contains rule-specific configuration
	Rules map[string]interface{} `yaml:"rules"`

	// Verbose enables verbose logging
	Verbose bool `yaml:"verbose"`

	// MaxErrors is the maximum number of errors to report (0 = no limit)
	MaxErrors int `yaml:"max_errors"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		SeverityThreshold: "warning",
		Ignore: []string{
			"**/vendor/**",
			"**/generated/**",
			"**/*_test.go", // By default, don't review test files
		},
		Rules: map[string]interface{}{
			"max_function_length": 100,
			"require_comments":    true,
			"enforce_error_handling": true,
			"unused_imports":      true,
			"check_nil_errors":    true,
		},
		Verbose:   false,
		MaxErrors: 0,
	}
}

// LoadConfig loads a configuration from a file
func LoadConfig(path string) (*Config, error) {
	config := DefaultConfig()

	// Check if config file exists
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return config, nil
	}

	// Read the config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	// Parse the YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}

	return config, nil
}