package app

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the complete configuration structure
type Config struct {
	Files    FilesConfig    `yaml:"files"`
	Template TemplateConfig `yaml:"template"`
	Schema   SchemaConfig   `yaml:"schema"`
	Lint     LintConfig     `yaml:"lint"`
	Render   RenderConfig   `yaml:"render"`
	Output   OutputConfig   `yaml:"output"`
}

// FilesConfig contains file-related configuration
type FilesConfig struct {
	Extensions          []string `yaml:"extensions"`
	DefaultTemplatesDir string   `yaml:"default_templates_dir"`
	DefaultOutputDir    string   `yaml:"default_output_dir"`
	DefaultValuesFile   string   `yaml:"default_values_file"`
	Helpers             []string `yaml:"helpers"`
}

// TemplateConfig contains template engine configuration
type TemplateConfig struct {
	LeftDelimiter  string `yaml:"left_delimiter"`
	RightDelimiter string `yaml:"right_delimiter"`
	DefaultMissing string `yaml:"default_missing"`
}

// LintConfig contains linting configuration
type LintConfig struct {
	FailOnWarn        bool     `yaml:"fail_on_warn"`
	FailOnUndefined   bool     `yaml:"fail_on_undefined"`
	StrictMode        bool     `yaml:"strict_mode"`
	OutputFormat      string   `yaml:"output_format"`
	Exclude           []string `yaml:"exclude"`
	DisallowFunctions []string `yaml:"disallow_functions"`
	RequiredVars      []string `yaml:"required_vars"`
	NoUndefCheck      bool     `yaml:"no_undefined_check"`
}

// RenderConfig contains rendering defaults
type RenderConfig struct {
	DryRun         bool   `yaml:"dry_run"`
	InjectGuard    bool   `yaml:"inject_guard"`
	GuardString    string `yaml:"guard_string"`
	PruneEmptyDirs bool   `yaml:"prune_empty_dirs"`
}

// OutputConfig contains output formatting configuration
type OutputConfig struct {
	Color   string `yaml:"color"` // auto, always, never
	Verbose bool   `yaml:"verbose"`
	Quiet   bool   `yaml:"quiet"`
}

// SchemaConfig contains schema validation configuration
type SchemaConfig struct {
	Path     string               `yaml:"path"`     // Path to schema file (default: .templr.schema.yml)
	Mode     string               `yaml:"mode"`     // error|warn|strict (default: warn)
	Generate SchemaGenerateConfig `yaml:"generate"` // Schema generation settings
}

// SchemaGenerateConfig contains schema generation settings
type SchemaGenerateConfig struct {
	Required        string `yaml:"required"`         // all|none|auto (default: auto)
	AdditionalProps bool   `yaml:"additional_props"` // Allow additionalProperties (default: true)
	InferTypes      bool   `yaml:"infer_types"`      // Infer types from values (default: true)
}

// NewDefaultConfig returns a Config with default values
func NewDefaultConfig() *Config {
	return &Config{
		Files: FilesConfig{
			Extensions: []string{"tpl"},
			Helpers:    []string{"_helpers*.tpl"},
		},
		Template: TemplateConfig{
			LeftDelimiter:  "{{",
			RightDelimiter: "}}",
			DefaultMissing: "<no value>",
		},
		Schema: SchemaConfig{
			Path: "",
			Mode: "warn",
			Generate: SchemaGenerateConfig{
				Required:        "auto",
				AdditionalProps: true,
				InferTypes:      true,
			},
		},
		Lint: LintConfig{
			FailOnWarn:        false,
			FailOnUndefined:   false,
			StrictMode:        false,
			OutputFormat:      "text",
			Exclude:           []string{},
			DisallowFunctions: []string{},
			RequiredVars:      []string{},
			NoUndefCheck:      false,
		},
		Render: RenderConfig{
			DryRun:         false,
			InjectGuard:    true,
			GuardString:    "#templr generated",
			PruneEmptyDirs: true,
		},
		Output: OutputConfig{
			Color:   "auto",
			Verbose: false,
			Quiet:   false,
		},
	}
}

// LoadConfig loads configuration from files with the following precedence:
// 1. Specified config file (--config flag)
// 2. .templr.yaml in current directory
// 3. ~/.config/templr/config.yaml (user config)
// 4. Built-in defaults
func LoadConfig(configPath string) (*Config, error) {
	config := NewDefaultConfig()

	// List of config files to try (in reverse precedence order)
	var configFiles []string

	// Add user config
	if userConfig := getUserConfigPath(); userConfig != "" {
		configFiles = append(configFiles, userConfig)
	}

	// Add project config
	if projectConfig := getProjectConfigPath(); projectConfig != "" {
		configFiles = append(configFiles, projectConfig)
	}

	// Add specified config (highest priority)
	if configPath != "" {
		configFiles = append(configFiles, configPath)
	}

	// Load and merge configs in order
	for _, path := range configFiles {
		if err := loadAndMergeConfig(config, path); err != nil {
			// Only fail if this was an explicitly specified config
			if path == configPath && configPath != "" {
				return nil, fmt.Errorf("load config %s: %w", path, err)
			}
			// Otherwise, just skip missing configs
			continue
		}
	}

	return config, nil
}

// getUserConfigPath returns the user's config file path
func getUserConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".config", "templr", "config.yaml")
}

// getProjectConfigPath returns the project's config file path
func getProjectConfigPath() string {
	// Look for .templr.yaml in current directory
	path := ".templr.yaml"
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

// loadAndMergeConfig loads a config file and merges it into the base config
func loadAndMergeConfig(base *Config, path string) error {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return err
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// Parse YAML
	var loaded Config
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}

	// Merge into base config
	mergeConfigs(base, &loaded)

	return nil
}

// mergeConfigs merges src into dst, with src taking precedence for non-zero values
func mergeConfigs(dst, src *Config) {
	// Merge Files config
	if len(src.Files.Extensions) > 0 {
		dst.Files.Extensions = src.Files.Extensions
	}
	if src.Files.DefaultTemplatesDir != "" {
		dst.Files.DefaultTemplatesDir = src.Files.DefaultTemplatesDir
	}
	if src.Files.DefaultOutputDir != "" {
		dst.Files.DefaultOutputDir = src.Files.DefaultOutputDir
	}
	if src.Files.DefaultValuesFile != "" {
		dst.Files.DefaultValuesFile = src.Files.DefaultValuesFile
	}
	if len(src.Files.Helpers) > 0 {
		dst.Files.Helpers = src.Files.Helpers
	}

	// Merge Template config
	if src.Template.LeftDelimiter != "" {
		dst.Template.LeftDelimiter = src.Template.LeftDelimiter
	}
	if src.Template.RightDelimiter != "" {
		dst.Template.RightDelimiter = src.Template.RightDelimiter
	}
	if src.Template.DefaultMissing != "" {
		dst.Template.DefaultMissing = src.Template.DefaultMissing
	}

	// Merge Schema config
	if src.Schema.Path != "" {
		dst.Schema.Path = src.Schema.Path
	}
	if src.Schema.Mode != "" {
		dst.Schema.Mode = src.Schema.Mode
	}
	if src.Schema.Generate.Required != "" {
		dst.Schema.Generate.Required = src.Schema.Generate.Required
	}
	dst.Schema.Generate.AdditionalProps = src.Schema.Generate.AdditionalProps
	dst.Schema.Generate.InferTypes = src.Schema.Generate.InferTypes

	// Merge Lint config
	// For booleans, we need to check if they were explicitly set in YAML
	// Since we can't easily distinguish false vs unset, we merge all fields
	dst.Lint.FailOnWarn = src.Lint.FailOnWarn
	dst.Lint.FailOnUndefined = src.Lint.FailOnUndefined
	dst.Lint.StrictMode = src.Lint.StrictMode
	dst.Lint.NoUndefCheck = src.Lint.NoUndefCheck

	if src.Lint.OutputFormat != "" {
		dst.Lint.OutputFormat = src.Lint.OutputFormat
	}
	if len(src.Lint.Exclude) > 0 {
		dst.Lint.Exclude = src.Lint.Exclude
	}
	if len(src.Lint.DisallowFunctions) > 0 {
		dst.Lint.DisallowFunctions = src.Lint.DisallowFunctions
	}
	if len(src.Lint.RequiredVars) > 0 {
		dst.Lint.RequiredVars = src.Lint.RequiredVars
	}

	// Merge Render config
	dst.Render.DryRun = src.Render.DryRun
	dst.Render.InjectGuard = src.Render.InjectGuard
	dst.Render.PruneEmptyDirs = src.Render.PruneEmptyDirs

	if src.Render.GuardString != "" {
		dst.Render.GuardString = src.Render.GuardString
	}

	// Merge Output config
	if src.Output.Color != "" {
		dst.Output.Color = src.Output.Color
	}
	dst.Output.Verbose = src.Output.Verbose
	dst.Output.Quiet = src.Output.Quiet
}

// ApplyConfigToSharedOptions applies config values to SharedOptions
// CLI flags take precedence over config file values
func ApplyConfigToSharedOptions(opts *SharedOptions, config *Config) {
	// Apply template delimiters if not set via CLI
	if opts.Ldelim == "{{" && config.Template.LeftDelimiter != "" {
		opts.Ldelim = config.Template.LeftDelimiter
	}
	if opts.Rdelim == "}}" && config.Template.RightDelimiter != "" {
		opts.Rdelim = config.Template.RightDelimiter
	}

	// Apply default missing value if not set via CLI
	if opts.DefaultMissing == "<no value>" && config.Template.DefaultMissing != "" {
		opts.DefaultMissing = config.Template.DefaultMissing
	}

	// Apply extra extensions from config if not set via CLI
	if len(opts.ExtraExts) == 0 && len(config.Files.Extensions) > 0 {
		// Only add non-default extensions (skip .tpl as it's always included)
		for _, ext := range config.Files.Extensions {
			if ext != "tpl" {
				opts.ExtraExts = append(opts.ExtraExts, ext)
			}
		}
	}

	// Apply guard string from config if not set via CLI
	if opts.Guard == "#templr generated" && config.Render.GuardString != "" {
		opts.Guard = config.Render.GuardString
	}

	// Apply inject guard from config (CLI doesn't have explicit flag for this)
	opts.InjectGuard = config.Render.InjectGuard

	// Apply strict mode from config if not set via CLI
	if !opts.Strict && config.Lint.StrictMode {
		opts.Strict = config.Lint.StrictMode
	}

	// Apply dry-run from config if not set via CLI
	if !opts.DryRun && config.Render.DryRun {
		opts.DryRun = config.Render.DryRun
	}

	// Apply no-color from config if not set via CLI
	if !opts.NoColor && config.Output.Color == "never" {
		opts.NoColor = true
	}
}

// ApplyConfigToLintOptions applies config values to LintOptions
func ApplyConfigToLintOptions(opts *LintOptions, config *Config) {
	// Apply shared options first
	ApplyConfigToSharedOptions(&opts.Shared, config)

	// Apply lint-specific options
	if !opts.FailOnWarn && config.Lint.FailOnWarn {
		opts.FailOnWarn = config.Lint.FailOnWarn
	}

	if opts.Format == "text" && config.Lint.OutputFormat != "" {
		opts.Format = config.Lint.OutputFormat
	}

	if !opts.NoUndefCheck && config.Lint.NoUndefCheck {
		opts.NoUndefCheck = config.Lint.NoUndefCheck
	}

	// Store config reference for use in linting
	opts.Config = config
}
