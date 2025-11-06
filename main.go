// Package main implements the templr CLI tool for rendering text templates with data and helpers.
package main

import (
	"fmt"
	"os"

	"github.com/kanopi/templr/internal/app"
	"github.com/spf13/cobra"
)

// Build-time variables (overridable via -ldflags)
var (
	Version string // preferred explicit version (e.g., a tag)
)

// Shared flag variables
var (
	flagConfig         string
	flagData           string
	flagFiles          []string
	flagSets           []string
	flagStrict         bool
	flagDryRun         bool
	flagGuard          string
	flagInjectGuard    bool
	flagDefaultMissing string
	flagNoColor        bool
	flagLdelim         string
	flagRdelim         string
	flagExtraExts      []string
)

// Command-specific flag variables
var (
	// render command
	flagRenderIn      string
	flagRenderOut     string
	flagRenderHelpers string

	// dir command
	flagDirPath string
	flagDirIn   string
	flagDirOut  string

	// walk command
	flagWalkSrc string
	flagWalkDst string

	// lint command
	flagLintIn           string
	flagLintDir          string
	flagLintSrc          string
	flagLintFailOnWarn   bool
	flagLintFormat       string
	flagLintNoUndefCheck bool

	// schema command
	flagSchemaPath            string
	flagSchemaMode            string
	flagSchemaOutput          string
	flagSchemaRequired        string
	flagSchemaAdditionalProps bool
)

var rootCmd = &cobra.Command{
	Use:   "templr",
	Short: "A Go-based templating CLI inspired by Helm",
	Long: `templr is a powerful Go-based templating CLI that renders templates from
single files or entire directories. It provides Helm-like features including
template helpers, strict mode, guards, and flexible data input.

SUBCOMMANDS:
  render    Render a single template file (default if no subcommand given)
  dir       Render templates from a directory
  walk      Recursively render template directory trees
  lint      Validate template syntax and detect issues
  version   Print version information

EXAMPLES:
  # Render a single template file
  templr render -in template.tpl -data values.yaml -out output.txt

  # Render from stdin to stdout
  echo '{{ .name }}' | templr render --set name=World

  # Render templates from a directory
  templr dir --dir templates/ -in main.tpl -data values.yaml -out output.txt

  # Walk and render entire directory tree
  templr walk --src templates/ --dst output/

  # Validate template syntax
  templr lint --src templates/ -data values.yaml

  # Backward compatible: old syntax still works
  templr -in template.tpl -data values.yaml -out output.txt

For detailed help on a specific command:
  templr help <command>`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "Render a single template file",
	Long: `Render a single template file to an output file or stdout.

Examples:
  # Render from file to file
  templr render -in template.tpl -data values.yaml -out output.txt

  # Render from stdin to stdout
  echo 'Hello {{ .name }}' | templr render -data values.yaml

  # Render with --set overrides
  templr render -in template.tpl --set name=World -out output.txt`,
	RunE: func(_ *cobra.Command, _ []string) error {
		opts := app.RenderOptions{
			Shared: app.SharedOptions{
				Data:           flagData,
				Files:          flagFiles,
				Sets:           flagSets,
				Strict:         flagStrict,
				DryRun:         flagDryRun,
				Guard:          flagGuard,
				InjectGuard:    flagInjectGuard,
				DefaultMissing: flagDefaultMissing,
				NoColor:        flagNoColor,
				Ldelim:         flagLdelim,
				Rdelim:         flagRdelim,
				ExtraExts:      flagExtraExts,
			},
			In:      flagRenderIn,
			Out:     flagRenderOut,
			Helpers: flagRenderHelpers,
		}
		return app.RunRenderMode(opts)
	},
}

var dirCmd = &cobra.Command{
	Use:   "dir",
	Short: "Render templates from a directory",
	Long: `Parse all templates in a directory together and execute an entry template.

This mode allows you to use template includes and partials across multiple files.

Examples:
  # Render using an entry template
  templr dir --dir templates/ -in main.tpl -data values.yaml -out output.txt

  # Render with auto-detected entry (looks for "root" template)
  templr dir --dir templates/ -data values.yaml -out output.txt`,
	RunE: func(_ *cobra.Command, _ []string) error {
		opts := app.DirOptions{
			Shared: app.SharedOptions{
				Data:           flagData,
				Files:          flagFiles,
				Sets:           flagSets,
				Strict:         flagStrict,
				DryRun:         flagDryRun,
				Guard:          flagGuard,
				InjectGuard:    flagInjectGuard,
				DefaultMissing: flagDefaultMissing,
				NoColor:        flagNoColor,
				Ldelim:         flagLdelim,
				Rdelim:         flagRdelim,
				ExtraExts:      flagExtraExts,
			},
			Dir: flagDirPath,
			In:  flagDirIn,
			Out: flagDirOut,
		}
		return app.RunDirMode(opts)
	},
}

var walkCmd = &cobra.Command{
	Use:   "walk",
	Short: "Recursively render template directory trees",
	Long: `Recursively walk through a source directory and render all templates,
mirroring the directory structure in the destination.

Template file extensions (.tpl by default, plus any specified with --ext)
are stripped from output filenames.

Examples:
  # Walk and render all templates
  templr walk --src templates/ --dst output/

  # Walk with additional file extensions
  templr walk --src templates/ --dst output/ --ext md --ext txt

  # Dry-run to preview changes
  templr walk --src templates/ --dst output/ --dry-run`,
	RunE: func(_ *cobra.Command, _ []string) error {
		opts := app.WalkOptions{
			Shared: app.SharedOptions{
				Data:           flagData,
				Files:          flagFiles,
				Sets:           flagSets,
				Strict:         flagStrict,
				DryRun:         flagDryRun,
				Guard:          flagGuard,
				InjectGuard:    flagInjectGuard,
				DefaultMissing: flagDefaultMissing,
				NoColor:        flagNoColor,
				Ldelim:         flagLdelim,
				Rdelim:         flagRdelim,
				ExtraExts:      flagExtraExts,
			},
			Src: flagWalkSrc,
			Dst: flagWalkDst,
		}
		return app.RunWalkMode(opts)
	},
}

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Validate template syntax and detect issues",
	Long: `Validate template syntax without rendering. Optionally detect undefined variables.

This command helps catch template errors early in CI/CD pipelines by:
  - Checking template syntax correctness (parse errors)
  - Detecting undefined variable references (with --data)
  - Reporting issues with file paths and line numbers

Examples:
  # Lint a single template file
  templr lint -i template.tpl -d values.yaml

  # Lint all templates in a directory
  templr lint --dir templates/ -d values.yaml

  # Lint entire directory tree
  templr lint --src templates/ -d values.yaml

  # Fail CI on warnings (not just errors)
  templr lint --src templates/ -d values.yaml --fail-on-warn

  # Skip undefined variable checking (syntax only)
  templr lint --src templates/ --no-undefined-check`,
	RunE: func(_ *cobra.Command, _ []string) error {
		// Load configuration
		config, err := app.LoadConfig(flagConfig)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		opts := app.LintOptions{
			Shared: app.SharedOptions{
				Data:           flagData,
				Files:          flagFiles,
				Sets:           flagSets,
				Strict:         flagStrict,
				DryRun:         flagDryRun,
				Guard:          flagGuard,
				InjectGuard:    flagInjectGuard,
				DefaultMissing: flagDefaultMissing,
				NoColor:        flagNoColor,
				Ldelim:         flagLdelim,
				Rdelim:         flagRdelim,
				ExtraExts:      flagExtraExts,
			},
			In:           flagLintIn,
			Dir:          flagLintDir,
			Src:          flagLintSrc,
			FailOnWarn:   flagLintFailOnWarn,
			Format:       flagLintFormat,
			NoUndefCheck: flagLintNoUndefCheck,
		}

		// Apply config to options (CLI flags take precedence)
		app.ApplyConfigToLintOptions(&opts, config)

		return app.RunLintMode(opts)
	},
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Schema validation and generation commands",
	Long: `Validate data against schemas or generate schemas from data.

Subcommands:
  validate  Validate data files against a schema
  generate  Generate a schema from data files`,
}

var schemaValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate data against a schema",
	Long: `Validate merged data files against a YAML schema.

The schema file can be specified via:
  1. --schema flag (highest priority)
  2. schema.path in .templr.yaml config
  3. Auto-discovery: .templr.schema.yml or .templr/schema.yml

Examples:
  # Validate using auto-discovered schema
  templr schema validate

  # Validate with explicit schema
  templr schema validate -schema my-schema.yml

  # Validate with specific data files
  templr schema validate -data values.yaml -schema schema.yml

  # Fail on errors (vs warnings)
  templr schema validate --schema-mode error`,
	RunE: func(_ *cobra.Command, _ []string) error {
		// Load config
		config, err := app.LoadConfig(flagConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[templr:error] load config: %v\n", err)
			os.Exit(app.ExitGeneral)
		}

		opts := app.SchemaOptions{
			Shared: app.SharedOptions{
				Data:           flagData,
				Files:          flagFiles,
				Sets:           flagSets,
				Strict:         flagStrict,
				DryRun:         flagDryRun,
				Guard:          flagGuard,
				InjectGuard:    flagInjectGuard,
				DefaultMissing: flagDefaultMissing,
				NoColor:        flagNoColor,
				Ldelim:         flagLdelim,
				Rdelim:         flagRdelim,
				ExtraExts:      flagExtraExts,
			},
			SchemaPath: flagSchemaPath,
			Mode:       flagSchemaMode,
		}

		if err := app.RunSchemaValidate(opts, config); err != nil {
			fmt.Fprintf(os.Stderr, "[templr:error] %v\n", err)
			os.Exit(app.ExitSchemaError)
		}
		return nil
	},
}

var schemaGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a schema from data files",
	Long: `Generate a YAML schema by analyzing your data files.

The generated schema will infer types and structure from your values.

Examples:
  # Generate schema to stdout
  templr schema generate -data values.yaml

  # Generate schema to file
  templr schema generate -data values.yaml -o schema.yml

  # Mark all fields as required
  templr schema generate -data values.yaml --required all -o schema.yml

  # Disallow additional properties
  templr schema generate -data values.yaml --additional-props=false -o schema.yml`,
	RunE: func(_ *cobra.Command, _ []string) error {
		// Load config
		config, err := app.LoadConfig(flagConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[templr:error] load config: %v\n", err)
			os.Exit(app.ExitGeneral)
		}

		opts := app.SchemaOptions{
			Shared: app.SharedOptions{
				Data:           flagData,
				Files:          flagFiles,
				Sets:           flagSets,
				Strict:         flagStrict,
				DryRun:         flagDryRun,
				Guard:          flagGuard,
				InjectGuard:    flagInjectGuard,
				DefaultMissing: flagDefaultMissing,
				NoColor:        flagNoColor,
				Ldelim:         flagLdelim,
				Rdelim:         flagRdelim,
				ExtraExts:      flagExtraExts,
			},
			Output:          flagSchemaOutput,
			Required:        flagSchemaRequired,
			AdditionalProps: flagSchemaAdditionalProps,
		}

		if err := app.RunSchemaGenerate(opts, config); err != nil {
			fmt.Fprintf(os.Stderr, "[templr:error] %v\n", err)
			os.Exit(app.ExitGeneral)
		}
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println(app.GetVersion())
	},
}

func init() {
	// Add persistent (global) flags to root command
	rootCmd.PersistentFlags().StringVar(&flagConfig, "config", "", "Path to config file (default: .templr.yaml or ~/.config/templr/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&flagData, "data", "d", "", "Path to base JSON or YAML data file")
	rootCmd.PersistentFlags().StringArrayVarP(&flagFiles, "f", "f", nil, "Additional values files (YAML/JSON). Repeatable.")
	rootCmd.PersistentFlags().StringArrayVar(&flagSets, "set", nil, "key=value overrides. Repeatable. Supports dotted keys.")
	rootCmd.PersistentFlags().BoolVar(&flagStrict, "strict", false, "Fail on missing keys")
	rootCmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "Preview which files would be rendered (no writes)")
	rootCmd.PersistentFlags().StringVar(&flagGuard, "guard", "#templr generated", "Guard string required in existing files to allow overwrite")
	rootCmd.PersistentFlags().BoolVar(&flagInjectGuard, "inject-guard", true, "Automatically insert the guard as a comment into written files")
	rootCmd.PersistentFlags().StringVar(&flagDefaultMissing, "default-missing", "<no value>", "String to render when a variable/key is missing")
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "Disable colored output (useful for CI/non-ANSI terminals)")
	rootCmd.PersistentFlags().StringVar(&flagLdelim, "ldelim", "{{", "Left delimiter")
	rootCmd.PersistentFlags().StringVar(&flagRdelim, "rdelim", "}}", "Right delimiter")
	rootCmd.PersistentFlags().StringArrayVar(&flagExtraExts, "ext", nil, "Additional template file extensions (e.g., md, txt). Repeatable.")

	// Render command flags
	renderCmd.Flags().StringVarP(&flagRenderIn, "in", "i", "", "Template file (omit for stdin)")
	renderCmd.Flags().StringVarP(&flagRenderOut, "out", "o", "", "Output file (omit for stdout)")
	renderCmd.Flags().StringVar(&flagRenderHelpers, "helpers", "_helpers*.tpl", "Glob pattern of helper templates to load. Set empty to skip.")

	// Dir command flags
	dirCmd.Flags().StringVar(&flagDirPath, "dir", "", "Directory containing templates (required)")
	dirCmd.Flags().StringVarP(&flagDirIn, "in", "i", "", "Entry template name (default: 'root' or first template)")
	dirCmd.Flags().StringVarP(&flagDirOut, "out", "o", "", "Output file (omit for stdout)")
	_ = dirCmd.MarkFlagRequired("dir")

	// Walk command flags
	walkCmd.Flags().StringVar(&flagWalkSrc, "src", "", "Source template directory (required)")
	walkCmd.Flags().StringVar(&flagWalkDst, "dst", "", "Destination output directory (required)")
	_ = walkCmd.MarkFlagRequired("src")
	_ = walkCmd.MarkFlagRequired("dst")

	// Lint command flags
	lintCmd.Flags().StringVarP(&flagLintIn, "in", "i", "", "Single template file to lint")
	lintCmd.Flags().StringVar(&flagLintDir, "dir", "", "Directory of templates to lint")
	lintCmd.Flags().StringVar(&flagLintSrc, "src", "", "Source directory tree to walk and lint")
	lintCmd.Flags().BoolVar(&flagLintFailOnWarn, "fail-on-warn", false, "Exit with code 1 on warnings (default: errors only)")
	lintCmd.Flags().StringVar(&flagLintFormat, "format", "text", "Output format: text, json, github-actions")
	lintCmd.Flags().BoolVar(&flagLintNoUndefCheck, "no-undefined-check", false, "Skip undefined variable detection")

	// Schema validate command flags
	schemaValidateCmd.Flags().StringVar(&flagSchemaPath, "schema", "", "Path to schema file (default: auto-discover)")
	schemaValidateCmd.Flags().StringVar(&flagSchemaMode, "schema-mode", "", "Validation mode: warn|error|strict (default from config or warn)")

	// Schema generate command flags
	schemaGenerateCmd.Flags().StringVarP(&flagSchemaOutput, "output", "o", "", "Output schema file (default: stdout)")
	schemaGenerateCmd.Flags().StringVar(&flagSchemaRequired, "required", "", "Mark fields as required: all|none|auto (default from config or auto)")
	schemaGenerateCmd.Flags().BoolVar(&flagSchemaAdditionalProps, "additional-props", true, "Allow additional properties in schema")

	// Add schema subcommands
	schemaCmd.AddCommand(schemaValidateCmd, schemaGenerateCmd)

	// Add subcommands
	rootCmd.AddCommand(renderCmd, dirCmd, walkCmd, lintCmd, schemaCmd, versionCmd)
}

func main() {
	// Set version in app package for build-time injection
	app.Version = Version

	// Check for legacy flag syntax (backward compatibility)
	if len(os.Args) > 1 {
		firstArg := os.Args[1]

		// Handle version flags specially
		if firstArg == "-version" || firstArg == "--version" {
			fmt.Println(app.GetVersion())
			return
		}

		// Handle help flags - use new mode
		if firstArg == "-h" || firstArg == "--help" {
			_ = rootCmd.Help()
			return
		}

		// Known subcommands - if first arg is one of these, use new mode
		knownSubcommands := map[string]bool{
			"render":     true,
			"dir":        true,
			"walk":       true,
			"lint":       true,
			"schema":     true,
			"version":    true,
			"help":       true,
			"completion": true,
		}

		// If first arg is NOT a known subcommand, use legacy mode
		if !knownSubcommands[firstArg] {
			// This handles cases like:
			// - templr -in file.tpl
			// - templr --walk --src ... --dst ...
			// - templr --dir templates/
			app.RunLegacyMode()
			return
		}
	}

	// Execute cobra command (will show help if no args)
	if err := rootCmd.Execute(); err != nil {
		// Map errors to appropriate exit codes
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		// Try to determine error type from message
		errMsg := err.Error()
		if app.Contains(errMsg, "parse") || app.Contains(errMsg, "template") {
			os.Exit(app.ExitTemplateError)
		} else if app.Contains(errMsg, "data") || app.Contains(errMsg, "load") {
			os.Exit(app.ExitDataError)
		} else if app.Contains(errMsg, "guard") {
			os.Exit(app.ExitGuardSkipped)
		}

		os.Exit(app.ExitGeneral)
	}
}
