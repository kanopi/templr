// Package app implements the core templr CLI commands and application logic.
package app

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
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

// buildFuncMap creates the template function map with Sprig and custom functions.
// The returned function map includes a closure reference to tpl for the include function.
func buildFuncMap(tpl **template.Template) template.FuncMap {
	funcs := sprig.TxtFuncMap()

	// Ensure YAML helpers exist (some environments/vendors strip these from Sprig)
	if _, ok := funcs["toYaml"]; !ok {
		funcs["toYaml"] = func(v any) (string, error) {
			b, err := yaml.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(b), nil
		}
	}
	if _, ok := funcs["fromYaml"]; !ok {
		funcs["fromYaml"] = func(s string) (map[string]any, error) {
			var m map[string]any
			if err := yaml.Unmarshal([]byte(s), &m); err != nil {
				return nil, err
			}
			return m, nil
		}
	}
	if _, ok := funcs["mustToYaml"]; !ok {
		funcs["mustToYaml"] = func(v any) string {
			b, err := yaml.Marshal(v)
			if err != nil {
				panic(err)
			}
			return string(b)
		}
	}
	if _, ok := funcs["mustFromYaml"]; !ok {
		funcs["mustFromYaml"] = func(s string) map[string]any {
			var m map[string]any
			if err := yaml.Unmarshal([]byte(s), &m); err != nil {
				panic(err)
			}
			return m
		}
	}

	// Helm-like helpers
	funcs["include"] = func(name string, data any) (string, error) {
		var b bytes.Buffer
		if *tpl == nil {
			return "", fmt.Errorf("template not initialized")
		}
		if err := (*tpl).ExecuteTemplate(&b, name, data); err != nil {
			return "", err
		}
		return b.String(), nil
	}
	funcs["required"] = func(msg string, v any) (any, error) {
		switch x := v.(type) {
		case nil:
			return nil, errors.New(msg)
		case string:
			if strings.TrimSpace(x) == "" {
				return nil, errors.New(msg)
			}
		case []any:
			if len(x) == 0 {
				return nil, errors.New(msg)
			}
		case map[string]any:
			if len(x) == 0 {
				return nil, errors.New(msg)
			}
		}
		return v, nil
	}
	funcs["fail"] = func(msg string) (string, error) { return "", errors.New(msg) }

	// set: mutate a map with key=value and return it (useful for introducing new vars)
	funcs["set"] = func(m map[string]any, key string, val any) (map[string]any, error) {
		if m == nil {
			return nil, fmt.Errorf("set: target map is nil")
		}
		m[key] = val
		return m, nil
	}
	// setd: dotted-key set (e.g., "a.b.c")
	funcs["setd"] = func(m map[string]any, dotted string, val any) (map[string]any, error) {
		if m == nil {
			return nil, fmt.Errorf("setd: target map is nil")
		}
		setByDottedKey(m, dotted, val)
		return m, nil
	}
	// mergeDeep: deep-merge two maps (right wins); returns new merged map
	funcs["mergeDeep"] = func(a, b map[string]any) map[string]any {
		out := map[string]any{}
		for k, v := range a {
			out[k] = v
		}
		return deepMerge(out, b)
	}
	// safe: render value or fallback when missing/empty
	funcs["safe"] = func(v any, def string) string {
		if v == nil {
			return def
		}
		switch vv := v.(type) {
		case string:
			if strings.TrimSpace(vv) == "" {
				return def
			}
			return vv
		default:
			return fmt.Sprint(v)
		}
	}

	return funcs
}

// buildValues constructs the values map from defaults, data files, and --set overrides
func buildValues(baseDir string, shared SharedOptions) (map[string]any, error) {
	values := map[string]any{}

	// Load default values.yaml from baseDir if it exists
	def, err := loadDefaultValues(baseDir)
	if err != nil {
		return nil, fmt.Errorf("load default values: %w", err)
	}
	values = deepMerge(values, def)

	// Load --data file if specified
	if shared.Data != "" {
		add, err := loadData(shared.Data)
		if err != nil {
			return nil, fmt.Errorf("load data: %w", err)
		}
		values = deepMerge(values, add)
	}

	// Load -f files
	for _, f := range shared.Files {
		add, err := loadData(f)
		if err != nil {
			return nil, fmt.Errorf("load -f %s: %w", f, err)
		}
		values = deepMerge(values, add)
	}

	// Apply --set overrides
	for _, kv := range shared.Sets {
		idx := strings.Index(kv, "=")
		if idx <= 0 {
			return nil, fmt.Errorf("--set expects key=value, got: %s", kv)
		}
		key := kv[:idx]
		val := parseScalar(kv[idx+1:])
		setByDottedKey(values, key, val)
	}

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
	// Determine Files.Root (dir of -in if present)
	filesRoot := "."
	if opts.In != "" {
		if info, err := os.Stat(opts.In); err == nil && !info.IsDir() {
			if abs, e := filepath.Abs(opts.In); e == nil {
				filesRoot = filepath.Dir(abs)
			}
		}
	}

	// Build values
	values, err := buildValues(filesRoot, opts.Shared)
	if err != nil {
		return err
	}

	// Add .Files API
	values["Files"] = FilesAPI{Root: filesRoot}

	// Create template with functions
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
		srcBytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
	} else {
		srcBytes, err = os.ReadFile(opts.In)
		if err != nil {
			return fmt.Errorf("read template: %w", err)
		}
		tplName = filepath.Base(opts.In)
	}
	sources[tplName] = srcBytes
	sources["root"] = srcBytes // Also map to "root" since that's what template.Parse uses

	// Load sidecar helpers in the same directory based on -helpers glob (default: _helpers.tpl)
	if filesRoot != "" && filesRoot != "." && opts.Helpers != "" {
		pattern := filepath.Join(filesRoot, opts.Helpers)
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			for _, hp := range matches {
				if b, e := os.ReadFile(hp); e == nil {
					helperName := filepath.ToSlash(filepath.Base(hp))
					sources[helperName] = b
					if _, e2 := tpl.New(helperName).Parse(string(b)); e2 != nil {
						return fmt.Errorf("parse helper %s: %w", hp, e2)
					}
				}
			}
		}
	}

	tpl, err = tpl.Parse(string(srcBytes))
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	// Compute helper-driven variables (templr.vars)
	if err := computeHelperVars(tpl, values); err != nil {
		return fmt.Errorf("helpers: %w", err)
	}

	// render to buffer
	outBytes, rerr := renderToBuffer(tpl, "", values)
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
