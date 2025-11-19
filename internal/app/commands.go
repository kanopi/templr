// Package app implements the core templr CLI commands and application logic.
package app

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/kanopi/templr/pkg/templr"
	"gopkg.in/yaml.v3"
)

// SharedOptions contains flags common to all commands
type SharedOptions struct {
	Data           string
	Files          []string
	Sets           []string
	Strict         bool
	DryRun         bool
	Guard          string
	InjectGuard    bool
	DefaultMissing string
	NoColor        bool
	Debug          bool
	Ldelim         string
	Rdelim         string
	ExtraExts      []string
}

// WalkOptions contains options specific to walk mode
type WalkOptions struct {
	Shared SharedOptions
	Src    string
	Dst    string
}

// DirOptions contains options specific to directory mode
type DirOptions struct {
	Shared SharedOptions
	Dir    string
	In     string
	Out    string
}

// RenderOptions contains options specific to single-file render mode
type RenderOptions struct {
	Shared  SharedOptions
	In      string
	Out     string
	Helpers string
}

// SchemaOptions contains options for schema commands
type SchemaOptions struct {
	Shared          SharedOptions
	SchemaPath      string
	Mode            string
	Output          string
	Required        string
	AdditionalProps bool
	Format          string
}

// buildFuncMap creates the template function map with Sprig and custom functions.
// The returned function map includes a closure reference to tpl for the include function.
// This is now a thin wrapper around pkg/templr.BuildFuncMap for backwards compatibility.
func buildFuncMap(tpl **template.Template) template.FuncMap {
	if tpl == nil || *tpl == nil {
		return templr.BuildFuncMap(nil)
	}
	return templr.BuildFuncMap(*tpl)
}

// All template functions have been moved to pkg/templr.BuildFuncMap for code sharing
// between the CLI and web playground.

// buildValues constructs the values map from defaults, data files, and --set overrides
func buildValues(baseDir string, shared SharedOptions) (map[string]any, error) {
	debugSection(shared.Debug, "Value Loading Sequence")
	values := map[string]any{}

	// Load default values.yaml from baseDir if it exists
	debugf(shared.Debug, "Loading default values from %s", baseDir)
	def, err := loadDefaultValues(baseDir)
	if err != nil {
		return nil, fmt.Errorf("load default values: %w", err)
	}
	if len(def) > 0 {
		debugf(shared.Debug, "  → Loaded %d key(s) from default values.yaml", len(def))
		if shared.Debug {
			for k := range def {
				debugf(shared.Debug, "     - %s", k)
			}
		}
	} else {
		debugf(shared.Debug, "  → No default values.yaml found")
	}
	values = deepMerge(values, def)

	// Load --data file if specified
	if shared.Data != "" {
		debugf(shared.Debug, "Loading data from --data=%s", shared.Data)
		add, err := loadData(shared.Data)
		if err != nil {
			return nil, fmt.Errorf("load data: %w", err)
		}
		debugf(shared.Debug, "  → Loaded %d key(s)", len(add))
		if shared.Debug {
			for k := range add {
				debugf(shared.Debug, "     - %s", k)
			}
		}
		values = deepMerge(values, add)
	}

	// Load -f files
	for _, f := range shared.Files {
		debugf(shared.Debug, "Loading data from -f %s", f)
		add, err := loadData(f)
		if err != nil {
			return nil, fmt.Errorf("load -f %s: %w", f, err)
		}
		debugf(shared.Debug, "  → Loaded %d key(s)", len(add))
		if shared.Debug {
			for k := range add {
				debugf(shared.Debug, "     - %s", k)
			}
		}
		values = deepMerge(values, add)
	}

	// Apply --set overrides
	if len(shared.Sets) > 0 {
		debugf(shared.Debug, "Applying %d --set override(s)", len(shared.Sets))
	}
	for _, kv := range shared.Sets {
		idx := strings.Index(kv, "=")
		if idx <= 0 {
			return nil, fmt.Errorf("--set expects key=value, got: %s", kv)
		}
		key := kv[:idx]
		val := parseScalar(kv[idx+1:])
		debugf(shared.Debug, "  → Setting %s = %v", key, val)
		setByDottedKey(values, key, val)
	}

	debugValues(shared.Debug, values, "Final Merged Values")

	return values, nil
}

// RunWalkMode executes walk mode: recursively render all templates in src to dst
func RunWalkMode(opts WalkOptions) error {
	if opts.Src == "" || opts.Dst == "" {
		return fmt.Errorf("-walk requires -src and -dst")
	}

	absSrc, _ := filepath.Abs(opts.Src)
	absDst, _ := filepath.Abs(opts.Dst)

	// Build values
	values, err := buildValues(absSrc, opts.Shared)
	if err != nil {
		return err
	}

	// Add .Files API
	values["Files"] = FilesAPI{Root: absSrc}

	// Create template with functions
	var tpl *template.Template
	funcs := buildFuncMap(&tpl)
	tpl = template.New("root").Funcs(funcs).Option("missingkey=default")
	if opts.Shared.Strict {
		tpl = tpl.Option("missingkey=error")
	}
	tpl = tpl.Delims(opts.Shared.Ldelim, opts.Shared.Rdelim)

	// Parse ALL templates (so includes/partials are available)
	allowExts := buildAllowedExts(opts.Shared.ExtraExts)
	var names []string
	var sources map[string][]byte
	tpl, names, sources, err = readAllTplsIntoSet(tpl, absSrc, allowExts)
	if err != nil {
		return fmt.Errorf("parse tree: %w", err)
	}

	// Compute helper-driven variables (templr.vars)
	if err := computeHelperVars(tpl, values); err != nil {
		return fmt.Errorf("helpers: %w", err)
	}

	// Render each non-partial template; skip empty; enforce guard on overwrite
	for _, name := range names {
		if !shouldRender(name) {
			continue
		}
		relOut := trimAnyExt(name, allowExts)
		dstPath := filepath.Join(absDst, filepath.FromSlash(relOut))

		// render to buffer first
		outBytes, rerr := renderToBuffer(tpl, name, values)
		if rerr != nil {
			if opts.Shared.Strict {
				strictErrf(rerr, sources, opts.Shared.NoColor)
			}
			return fmt.Errorf("render error %s: %w", name, rerr)
		}
		// apply global default-missing replacement
		outBytes = applyDefaultMissing(outBytes, opts.Shared.DefaultMissing)

		if isEmpty(outBytes) {
			if opts.Shared.DryRun {
				fmt.Printf("[dry-run] skip empty %s (no file created)\n", dstPath)
			}
			continue
		}

		// Guard check BEFORE any mkdir/write
		ok, gerr := canOverwrite(dstPath, opts.Shared.Guard)
		if gerr != nil && !os.IsNotExist(gerr) {
			return fmt.Errorf("guard check %s: %w", dstPath, gerr)
		}
		if !ok {
			if opts.Shared.DryRun {
				fmt.Printf("[dry-run] skip (guard missing) %s\n", dstPath)
			} else {
				warnf("guard", "skip (guard missing) %s", dstPath)
			}
			continue
		}

		if opts.Shared.DryRun {
			simulated := outBytes
			if opts.Shared.InjectGuard {
				simulated = injectGuardForExt(dstPath, simulated, opts.Shared.Guard)
				if !bytes.Equal(simulated, outBytes) {
					fmt.Printf("[dry-run] would inject guard into %s\n", dstPath)
				}
			}
			// Check if file would change
			same, _ := fastEqual(dstPath, simulated)
			if same {
				fmt.Printf("[dry-run] would skip unchanged %s\n", dstPath)
			} else {
				fmt.Printf("[dry-run] would render %s -> %s (changed)\n", name, dstPath)
			}
			continue
		}

		// Optionally inject guard comment
		if opts.Shared.InjectGuard {
			outBytes = injectGuardForExt(dstPath, outBytes, opts.Shared.Guard)
		}
		// Write only if content changed
		changed, err := writeIfChanged(dstPath, outBytes, 0o644)
		if err != nil {
			return fmt.Errorf("write %s: %w", dstPath, err)
		}
		if changed {
			fmt.Printf("rendered %s -> %s\n", name, dstPath)
		}
	}

	// Cleanup: remove empty directories under dst
	if err := templr.PruneEmptyDirs(absDst); err != nil {
		return fmt.Errorf("prune: %w", err)
	}

	return nil
}

// RunDirMode executes directory mode: parse all templates in dir, execute one entry
//
//nolint:gocyclo,cyclop // orchestration function with inherent complexity
func RunDirMode(opts DirOptions) error {
	if opts.Dir == "" {
		return fmt.Errorf("--dir is required")
	}

	absDir, _ := filepath.Abs(opts.Dir)

	// Build values
	values, err := buildValues(absDir, opts.Shared)
	if err != nil {
		return err
	}

	// Add .Files API
	values["Files"] = FilesAPI{Root: absDir}

	// Create template with functions
	var tpl *template.Template
	funcs := buildFuncMap(&tpl)
	tpl = template.New("root").Funcs(funcs).Option("missingkey=default")
	if opts.Shared.Strict {
		tpl = tpl.Option("missingkey=error")
	}
	tpl = tpl.Delims(opts.Shared.Ldelim, opts.Shared.Rdelim)

	// Parse all *.tpl in dir using path-based names
	allowExts := buildAllowedExts(opts.Shared.ExtraExts)
	var names []string
	var sources map[string][]byte
	tpl, names, sources, err = readAllTplsIntoSet(tpl, absDir, allowExts)
	if err != nil {
		return fmt.Errorf("parse dir templates: %w", err)
	}

	// Compute helper-driven variables (templr.vars)
	if err := computeHelperVars(tpl, values); err != nil {
		return fmt.Errorf("helpers: %w", err)
	}

	// Determine entry template name
	entryName := ""
	if opts.In != "" {
		// If -in is a file path, convert to rel name; otherwise assume it's already a template name.
		if info, err := os.Stat(opts.In); err == nil && !info.IsDir() {
			if rel, er := filepath.Rel(absDir, opts.In); er == nil {
				entryName = filepath.ToSlash(rel)
			} else {
				entryName = filepath.Base(opts.In)
			}
		} else {
			entryName = opts.In
		}
	} else if tpl.Lookup("root") != nil {
		entryName = "root"
	} else if len(names) > 0 {
		entryName = names[0]
	} else {
		return fmt.Errorf("no templates found in --dir")
	}

	// render to buffer
	outBytes, rerr := renderToBuffer(tpl, entryName, values)
	if rerr != nil {
		if opts.Shared.Strict {
			strictErrf(rerr, sources, opts.Shared.NoColor)
		}
		return rerr
	}
	// apply global default-missing replacement
	outBytes = applyDefaultMissing(outBytes, opts.Shared.DefaultMissing)

	if isEmpty(outBytes) {
		target := "stdout"
		if opts.Out != "" {
			target = opts.Out
		}
		if opts.Shared.DryRun {
			fmt.Printf("[dry-run] skip empty render for entry %s -> %s\n", entryName, target)
			return nil
		}
		fmt.Fprintf(os.Stderr, "skipping empty render for entry %s -> %s\n", entryName, target)
		return nil
	}

	// If writing to a file, guard-verify when target exists
	if opts.Out != "" {
		ok, gerr := canOverwrite(opts.Out, opts.Shared.Guard)
		if gerr != nil && !os.IsNotExist(gerr) {
			return fmt.Errorf("guard check %s: %w", opts.Out, gerr)
		}
		if !ok {
			if opts.Shared.DryRun {
				fmt.Printf("[dry-run] skip (guard missing) %s\n", opts.Out)
			} else {
				warnf("guard", "skip (guard missing) %s", opts.Out)
			}
			return nil
		}
	}

	if opts.Shared.DryRun {
		target := "stdout"
		if opts.Out != "" {
			target = opts.Out
		}
		if opts.Out != "" && opts.Shared.InjectGuard {
			simulated := injectGuardForExt(opts.Out, outBytes, opts.Shared.Guard)
			if !bytes.Equal(simulated, outBytes) {
				fmt.Printf("[dry-run] would inject guard into %s\n", opts.Out)
			}
		}
		// Check if file would change
		if opts.Out != "" {
			simToCheck := outBytes
			if opts.Shared.InjectGuard {
				simToCheck = injectGuardForExt(opts.Out, outBytes, opts.Shared.Guard)
			}
			same, _ := fastEqual(opts.Out, simToCheck)
			if same {
				fmt.Printf("[dry-run] would skip unchanged %s\n", opts.Out)
			} else {
				fmt.Printf("[dry-run] would render entry %s -> %s (changed)\n", entryName, target)
			}
		} else {
			fmt.Printf("[dry-run] would render entry %s -> %s\n", entryName, target)
		}
		return nil
	}

	// write (stdout or file)
	if opts.Out != "" {
		// Optionally inject guard comment
		if opts.Shared.InjectGuard {
			outBytes = injectGuardForExt(opts.Out, outBytes, opts.Shared.Guard)
		}
		// Write only if content changed
		changed, err := writeIfChanged(opts.Out, outBytes, 0o644)
		if err != nil {
			return fmt.Errorf("write out: %w", err)
		}
		if changed {
			fmt.Printf("rendered entry %s -> %s\n", entryName, opts.Out)
		}
		return nil
	}

	if _, err := os.Stdout.Write(outBytes); err != nil {
		return err
	}
	return nil
}

// RunRenderMode executes single-file render mode
//
//nolint:gocyclo,cyclop // orchestration function with inherent complexity
func RunRenderMode(opts RenderOptions) error {
	debugSection(opts.Shared.Debug, "Template Rendering Flow")

	// Determine Files.Root (dir of -in if present)
	filesRoot := "."
	if opts.In != "" {
		if info, err := os.Stat(opts.In); err == nil && !info.IsDir() {
			if abs, e := filepath.Abs(opts.In); e == nil {
				filesRoot = filepath.Dir(abs)
			}
		}
	}
	debugf(opts.Shared.Debug, "Files.Root directory: %s", filesRoot)

	// Build values
	values, err := buildValues(filesRoot, opts.Shared)
	if err != nil {
		return err
	}

	// Add .Files API
	values["Files"] = FilesAPI{Root: filesRoot}
	debugf(opts.Shared.Debug, "Added .Files API with root: %s", filesRoot)

	// Create template with functions
	debugf(opts.Shared.Debug, "Creating template with delimiters: %s ... %s", opts.Shared.Ldelim, opts.Shared.Rdelim)
	if opts.Shared.Strict {
		debugf(opts.Shared.Debug, "Strict mode enabled (missingkey=error)")
	}
	var tpl *template.Template
	funcs := buildFuncMap(&tpl)
	tpl = template.New("root").Funcs(funcs).Option("missingkey=default")
	if opts.Shared.Strict {
		tpl = tpl.Option("missingkey=error")
	}
	tpl = tpl.Delims(opts.Shared.Ldelim, opts.Shared.Rdelim)

	// Read template source
	var srcBytes []byte
	sources := make(map[string][]byte)
	tplName := "stdin"
	if opts.In == "" {
		debugf(opts.Shared.Debug, "Reading template from stdin")
		srcBytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
	} else {
		debugf(opts.Shared.Debug, "Reading template from file: %s", opts.In)
		srcBytes, err = os.ReadFile(opts.In)
		if err != nil {
			return fmt.Errorf("read template: %w", err)
		}
		tplName = filepath.Base(opts.In)
	}
	debugf(opts.Shared.Debug, "Main template: %s (%d bytes)", tplName, len(srcBytes))
	sources[tplName] = srcBytes
	sources["root"] = srcBytes // Also map to "root" since that's what template.Parse uses

	// Load sidecar helpers in the same directory based on -helpers glob (default: _helpers.tpl)
	if filesRoot != "" && filesRoot != "." && opts.Helpers != "" {
		pattern := filepath.Join(filesRoot, opts.Helpers)
		debugf(opts.Shared.Debug, "Looking for helper templates: %s", pattern)
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			debugf(opts.Shared.Debug, "Found %d helper template(s)", len(matches))
			for _, hp := range matches {
				if b, e := os.ReadFile(hp); e == nil {
					helperName := filepath.ToSlash(filepath.Base(hp))
					debugf(opts.Shared.Debug, "  → Loading helper: %s (%d bytes)", helperName, len(b))
					sources[helperName] = b
					if _, e2 := tpl.New(helperName).Parse(string(b)); e2 != nil {
						return fmt.Errorf("parse helper %s: %w", hp, e2)
					}
				}
			}
		} else {
			debugf(opts.Shared.Debug, "  → No helper templates found")
		}
	}

	debugf(opts.Shared.Debug, "Parsing main template")
	tpl, err = tpl.Parse(string(srcBytes))
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	// Compute helper-driven variables (templr.vars)
	debugf(opts.Shared.Debug, "Checking for templr.vars template")
	if err := computeHelperVars(tpl, values); err != nil {
		return fmt.Errorf("helpers: %w", err)
	}
	if tpl.Lookup("templr.vars") != nil {
		debugf(opts.Shared.Debug, "  → templr.vars executed, values updated")
		if opts.Shared.Debug {
			debugValues(opts.Shared.Debug, values, "Values After templr.vars")
		}
	} else {
		debugf(opts.Shared.Debug, "  → No templr.vars template found")
	}

	// render to buffer
	debugf(opts.Shared.Debug, "Rendering template")
	outBytes, rerr := renderToBuffer(tpl, "", values)
	if rerr != nil {
		if opts.Shared.Strict {
			strictErrf(rerr, sources, opts.Shared.NoColor)
		}
		return rerr
	}
	debugf(opts.Shared.Debug, "Render complete (%d bytes)", len(outBytes))

	// apply global default-missing replacement
	outBytes = applyDefaultMissing(outBytes, opts.Shared.DefaultMissing)

	if isEmpty(outBytes) {
		target := "stdout"
		if opts.Out != "" {
			target = opts.Out
		}
		if opts.Shared.DryRun {
			srcLabel := "stdin"
			if opts.In != "" {
				srcLabel = opts.In
			}
			fmt.Printf("[dry-run] skip empty render %s -> %s\n", srcLabel, target)
			return nil
		}
		fmt.Fprintf(os.Stderr, "skipping empty render -> %s\n", target)
		return nil
	}

	// If writing to a file, guard-verify when target exists
	if opts.Out != "" {
		ok, gerr := canOverwrite(opts.Out, opts.Shared.Guard)
		if gerr != nil && !os.IsNotExist(gerr) {
			return fmt.Errorf("guard check %s: %w", opts.Out, gerr)
		}
		if !ok {
			if opts.Shared.DryRun {
				fmt.Printf("[dry-run] skip (guard missing) %s\n", opts.Out)
				return nil
			}
			warnf("guard", "skip (guard missing) %s", opts.Out)
			return nil
		}
	}

	if opts.Shared.DryRun {
		target := "stdout"
		if opts.Out != "" {
			target = opts.Out
		}
		srcLabel := "stdin"
		if opts.In != "" {
			srcLabel = opts.In
		}
		if opts.Out != "" && opts.Shared.InjectGuard {
			simulated := injectGuardForExt(opts.Out, outBytes, opts.Shared.Guard)
			if !bytes.Equal(simulated, outBytes) {
				fmt.Printf("[dry-run] would inject guard into %s\n", opts.Out)
			}
		}
		// Check if file would change
		if opts.Out != "" {
			simToCheck := outBytes
			if opts.Shared.InjectGuard {
				simToCheck = injectGuardForExt(opts.Out, outBytes, opts.Shared.Guard)
			}
			same, _ := fastEqual(opts.Out, simToCheck)
			if same {
				fmt.Printf("[dry-run] would skip unchanged %s\n", opts.Out)
			} else {
				fmt.Printf("[dry-run] would render %s -> %s (changed)\n", srcLabel, target)
			}
		} else {
			fmt.Printf("[dry-run] would render %s -> %s\n", srcLabel, target)
		}
		return nil
	}

	// write (stdout or file)
	if opts.Out != "" {
		// Optionally inject guard comment
		if opts.Shared.InjectGuard {
			outBytes = injectGuardForExt(opts.Out, outBytes, opts.Shared.Guard)
		}
		// Write only if content changed
		changed, err := writeIfChanged(opts.Out, outBytes, 0o644)
		if err != nil {
			return fmt.Errorf("write out: %w", err)
		}
		if changed {
			srcLabel := "stdin"
			if opts.In != "" {
				srcLabel = opts.In
			}
			fmt.Printf("rendered %s -> %s\n", srcLabel, opts.Out)
		}
		return nil
	}

	if _, err := os.Stdout.Write(outBytes); err != nil {
		return err
	}
	return nil
}

// RunSchemaValidate validates data against a schema
func RunSchemaValidate(opts SchemaOptions, config *Config) error {
	// Load and merge data
	vals, err := buildValues(".", opts.Shared)
	if err != nil {
		return err
	}

	// Determine schema path
	schemaPath := opts.SchemaPath
	if schemaPath == "" {
		// Try auto-discovery
		schemaPath = FindSchemaFile(config.Schema.Path)
		if schemaPath == "" {
			return fmt.Errorf("no schema file found (checked: %s, .templr.schema.yml, .templr/schema.yml)", config.Schema.Path)
		}
	}

	// Determine mode
	mode := opts.Mode
	if mode == "" {
		mode = config.Schema.Mode
	}
	if mode == "" {
		mode = "warn"
	}

	// Validate
	result, err := ValidateWithSchema(vals, schemaPath, mode)
	if err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	// Format and print errors
	if !result.Passed {
		output := FormatSchemaErrors(result, mode)
		fmt.Fprint(os.Stderr, output)

		if mode != "warn" {
			return fmt.Errorf("validation failed")
		}

		fmt.Printf("✓ Validation complete (%d warning%s)\n", len(result.Errors), pluralize(len(result.Errors)))
		return nil
	}

	fmt.Println("✓ Validation passed")
	return nil
}

// RunSchemaGenerate generates a schema from data
func RunSchemaGenerate(opts SchemaOptions, config *Config) error {
	// Load and merge data
	vals, err := buildValues(".", opts.Shared)
	if err != nil {
		return err
	}

	// Build generation config
	genConfig := config.Schema.Generate
	if opts.Required != "" {
		genConfig.Required = opts.Required
	}
	genConfig.AdditionalProps = opts.AdditionalProps

	// Generate schema
	schema, err := GenerateSchema(vals, genConfig)
	if err != nil {
		return fmt.Errorf("generate schema: %w", err)
	}

	// Marshal to YAML
	schemaBytes, err := yaml.Marshal(schema)
	if err != nil {
		return fmt.Errorf("marshal schema: %w", err)
	}

	// Write output
	if opts.Output != "" {
		if err := os.WriteFile(opts.Output, schemaBytes, 0o644); err != nil {
			return fmt.Errorf("write schema file: %w", err)
		}
		fmt.Printf("Generated schema -> %s\n", opts.Output)
	} else {
		// Print to stdout
		fmt.Print(string(schemaBytes))
	}

	return nil
}

// pluralize returns "s" if count is not 1
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// Debug logging helpers
func debugf(debug bool, format string, args ...any) {
	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

func debugSection(debug bool, title string) {
	if debug {
		fmt.Fprint(os.Stderr, "\n"+strings.Repeat("=", 60)+"\n")
		fmt.Fprintf(os.Stderr, "[DEBUG] %s\n", title)
		fmt.Fprint(os.Stderr, strings.Repeat("=", 60)+"\n")
	}
}

func debugValues(debug bool, values map[string]any, title string) {
	if !debug {
		return
	}

	debugSection(debug, title)

	// Convert to YAML for pretty printing
	yamlBytes, err := yaml.Marshal(values)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[DEBUG] Error marshaling values: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stderr, "%s\n", string(yamlBytes))
}
