// Package main implements the templr CLI tool for rendering text templates with data and helpers.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"github.com/Masterminds/sprig/v3"
	"github.com/kanopi/templr/pkg/templr"
	"gopkg.in/yaml.v3"
)

// Build-time variables (overridable via -ldflags)
var (
	Version string // preferred explicit version (e.g., a tag)
)

// ANSI color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// Exit codes for CI-friendly behavior.
const (
	ExitOK            = 0
	ExitGeneral       = 1
	ExitTemplateError = 2
	ExitDataError     = 3
	ExitStrictError   = 4
	ExitGuardSkipped  = 5
)

// errf prints a standardized error line and exits with the given code.
// Format: [templr:error:<kind>] message
func errf(code int, kind, format string, a ...any) {
	fmt.Fprintf(os.Stderr, "[templr:error:%s] %s\n", kind, fmt.Sprintf(format, a...))
	os.Exit(code)
}

// warnf prints a standardized warning (does not exit).
// Format: [templr:warn:<kind>] message
func warnf(kind, format string, a ...any) {
	fmt.Fprintf(os.Stderr, "[templr:warn:%s] %s\n", kind, fmt.Sprintf(format, a...))
}

// strictErrf prints an enhanced strict mode error with context and exits with ExitStrictError.
func strictErrf(err error, sources map[string][]byte, noColor bool) {
	fmt.Fprint(os.Stderr, formatStrictError(err, sources, noColor))
	os.Exit(ExitStrictError)
}

// formatStrictError enhances strict mode errors with colors, context lines, and helpful hints.
// It parses Go template errors to extract line numbers and missing keys, then formats them
// in a developer-friendly way with syntax highlighting and contextual information.
func formatStrictError(err error, templateSources map[string][]byte, noColor bool) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()

	// Helper to optionally colorize text
	colorize := func(color, text string) string {
		if noColor {
			return text
		}
		return color + text + colorReset
	}

	// Parse template error format: "template: name:line:col: executing ... at <expr>: message"
	// Example: template: example.tpl:5:12: executing "example.tpl" at <.missing.key>: map has no entry for key "missing"
	var tplName string
	var lineNum int
	var expr string
	var missingKey string

	// Try to parse template name and line number
	if strings.HasPrefix(errMsg, "template: ") {
		rest := errMsg[10:] // skip "template: "
		// Find first colon (marks end of template name)
		if idx := strings.Index(rest, ":"); idx > 0 {
			tplName = rest[:idx]
			rest = rest[idx+1:]
			// Try to parse line number
			if idx2 := strings.Index(rest, ":"); idx2 > 0 {
				if ln, e := strconv.Atoi(rest[:idx2]); e == nil {
					lineNum = ln
				}
			}
		}
	}

	// Extract the expression that failed (between < and >)
	if start := strings.Index(errMsg, "at <"); start >= 0 {
		start += 4
		if end := strings.Index(errMsg[start:], ">"); end >= 0 {
			expr = errMsg[start : start+end]
		}
	}

	// Extract missing key from error message
	if strings.Contains(errMsg, "map has no entry for key") {
		if start := strings.Index(errMsg, `key "`); start >= 0 {
			start += 5
			if end := strings.Index(errMsg[start:], `"`); end >= 0 {
				missingKey = errMsg[start : start+end]
			}
		}
	}

	var buf bytes.Buffer

	// Error header
	buf.WriteString(colorize(colorRed+colorBold, "✗ Strict Mode Error") + "\n")

	if tplName != "" && lineNum > 0 {
		buf.WriteString(colorize(colorCyan, fmt.Sprintf("  %s:%d", tplName, lineNum)) + "\n\n")

		// Try to show context if we have the source
		if src, ok := templateSources[tplName]; ok {
			lines := bytes.Split(src, []byte("\n"))
			if lineNum > 0 && lineNum <= len(lines) {
				// Show context: 1 line before and 1 line after
				start := lineNum - 2
				if start < 0 {
					start = 0
				}
				end := lineNum + 1
				if end > len(lines) {
					end = len(lines)
				}

				for i := start; i < end; i++ {
					lineNumStr := fmt.Sprintf("%4d", i+1)
					if i+1 == lineNum {
						// Highlight the error line
						buf.WriteString(colorize(colorGray, lineNumStr) + " | ")
						buf.WriteString(colorize(colorRed, string(lines[i])) + "\n")
						// Add pointer to the error location
						buf.WriteString(colorize(colorGray, "     | "))
						buf.WriteString(colorize(colorRed, "^ Error occurred here") + "\n")
					} else {
						buf.WriteString(colorize(colorGray, lineNumStr) + " | ")
						buf.WriteString(string(lines[i]) + "\n")
					}
				}
				buf.WriteString("\n")
			}
		}
	}

	// Show the missing key/expression
	if expr != "" {
		buf.WriteString(colorize(colorRed, "  Missing: ") + expr + "\n")
	}
	if missingKey != "" {
		buf.WriteString(colorize(colorRed, "  Key: ") + missingKey + "\n")
	}

	buf.WriteString("\n")

	// Show the original error message (dimmed)
	buf.WriteString(colorize(colorGray, "  Details: "+errMsg) + "\n\n")

	// Helpful hint
	buf.WriteString(colorize(colorYellow, "  💡 Tip: "))
	if missingKey != "" {
		buf.WriteString(fmt.Sprintf("Define '%s' in your values file, or run without --strict to use defaults.\n", missingKey))
	} else if expr != "" {
		buf.WriteString(fmt.Sprintf("Define '%s' in your values file, or run without --strict to use defaults.\n", expr))
	} else {
		buf.WriteString("Check your values file to ensure all required keys are defined, or run without --strict.\n")
	}

	return buf.String()
}

// getVersion returns a human-friendly version string.
// Priority:
//  1. Version (ldflags)
//  2. "dev"
func getVersion() string {
	if Version != "" {
		return Version
	}
	return "dev"
}

type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}

// FilesAPI provides a Helm-like .Files facade anchored at a directory.
type FilesAPI struct {
	Root string
}

func (f FilesAPI) Get(path string) (string, error) {
	b, err := os.ReadFile(filepath.Join(f.Root, path))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (f FilesAPI) GetBytes(path string) ([]byte, error) {
	return os.ReadFile(filepath.Join(f.Root, path))
}

func (f FilesAPI) Glob(pat string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(f.Root, pat))
	if err != nil {
		return nil, err
	}
	rel := make([]string, 0, len(matches))
	for _, m := range matches {
		if r, err := filepath.Rel(f.Root, m); err == nil {
			rel = append(rel, filepath.ToSlash(r))
		} else {
			rel = append(rel, m)
		}
	}
	return rel, nil
}

func loadData(path string) (map[string]any, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	var m map[string]any
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.NewDecoder(f).Decode(&m); err != nil {
			return nil, fmt.Errorf("yaml decode: %w", err)
		}
	case ".json":
		if err := json.NewDecoder(f).Decode(&m); err != nil {
			return nil, fmt.Errorf("json decode: %w", err)
		}
	default:
		// Try YAML then JSON
		if err := yaml.NewDecoder(f).Decode(&m); err != nil {
			if _, e := f.Seek(0, 0); e != nil {
				return nil, e
			}
			if err2 := json.NewDecoder(f).Decode(&m); err2 != nil {
				return nil, fmt.Errorf("could not parse as YAML or JSON: %v / %v", err, err2)
			}
		}
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

// deepMerge merges src into dst (maps only), recursively.
func deepMerge(dst, src map[string]any) map[string]any {
	if dst == nil {
		dst = map[string]any{}
	}
	for k, v := range src {
		if dm, ok := dst[k].(map[string]any); ok {
			if sm, ok := v.(map[string]any); ok {
				dst[k] = deepMerge(dm, sm)
				continue
			}
		}
		dst[k] = v
	}
	return dst
}

// parseScalar tries to convert a string to bool, int, float, or JSON/YAML; falls back to string.
func parseScalar(s string) any {
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err == nil {
		return v
	}
	if err := yaml.Unmarshal([]byte(s), &v); err == nil {
		return v
	}
	return s
}

// setByDottedKey assigns val into m using a dotted path (e.g., "a.b.c") creating maps along the way.
func setByDottedKey(m map[string]any, dotted string, val any) {
	parts := strings.Split(dotted, ".")
	cur := m
	for i, p := range parts {
		if i == len(parts)-1 {
			cur[p] = val
			return
		}
		next, ok := cur[p]
		if !ok {
			nm := map[string]any{}
			cur[p] = nm
			cur = nm
			continue
		}
		nmm, ok := next.(map[string]any)
		if !ok {
			nmm = map[string]any{}
			cur[p] = nmm
		}
		cur = nmm
	}
}

// buildAllowedExts returns a set of allowed template extensions, always including ".tpl".
// The inputs should be bare extensions without leading dots (e.g., "md", "txt").
func buildAllowedExts(extra []string) map[string]bool {
	m := map[string]bool{".tpl": true}
	for _, e := range extra {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		m[e] = true
	}
	return m
}

// trimAnyExt removes the first matching extension from name based on allowExts.
// If none match, returns name unchanged.
func trimAnyExt(name string, allowExts map[string]bool) string {
	lower := strings.ToLower(name)
	// Prefer longer extensions first (e.g., .tpl.txt before .txt) if ever present
	var exts []string
	for e := range allowExts {
		exts = append(exts, e)
	}
	sort.Slice(exts, func(i, j int) bool { return len(exts[i]) > len(exts[j]) })
	for _, e := range exts {
		if strings.HasSuffix(lower, e) {
			return name[:len(name)-len(e)]
		}
	}
	return name
}

// readAllTplsIntoSet parses every allowed template file under root into the given template set,
// naming each template by its forward-slash relative path (to avoid base name collisions).
// Also returns a map of template sources for error reporting.
func readAllTplsIntoSet(tpl *template.Template, root string, allowExts map[string]bool) (*template.Template, []string, map[string][]byte, error) {
	var names []string
	sources := make(map[string][]byte)
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if !allowExts[ext] {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel) // normalize
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		sources[rel] = src // Store source for error reporting
		_, err = tpl.New(rel).Parse(string(src))
		if err != nil {
			return fmt.Errorf("parse %s: %w", rel, err)
		}
		names = append(names, rel)
		return nil
	})
	return tpl, names, sources, err
}

// shouldRender returns false for "partials" (files whose base name starts with "_").
func shouldRender(rel string) bool {
	base := filepath.Base(rel)
	return !strings.HasPrefix(base, "_")
}

// isEmpty reports true if, after normalizing line endings,
// stripping BOM, and removing all Unicode whitespace, nothing remains.
// This handles edge cases like CRLF, UTF-8 BOM, non-breaking spaces,
// zero-width spaces, and other Unicode whitespace characters.
func isEmpty(b []byte) bool {
	// Strip UTF-8 BOM if present
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		b = b[3:]
	}

	// Normalize CRLF -> LF
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))

	// Remove all Unicode whitespace
	// (including spaces, tabs, newlines, NBSP, ZWSP categories, etc.)
	for _, r := range string(b) {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// renderToBuffer executes a template into an in-memory buffer.
func renderToBuffer(tpl *template.Template, name string, values map[string]any) ([]byte, error) {
	var buf bytes.Buffer
	if name == "" {
		if err := tpl.Execute(&buf, values); err != nil {
			return nil, err
		}
	} else {
		if err := tpl.ExecuteTemplate(&buf, name, values); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// applyDefaultMissing replaces the engine's "<no value>" placeholder with a configured string.
// Used with template.Option("missingkey=default").
func applyDefaultMissing(out []byte, replacement string) []byte {
	if replacement == "" || replacement == "<no value>" {
		return out
	}
	return bytes.ReplaceAll(out, []byte("<no value>"), []byte(replacement))
}

// canOverwrite checks guard when target exists.
// If file doesn't exist → allowed. If exists → allowed only if guard is present.
// Uses flexible guard detection that accounts for different comment styles per language.
func canOverwrite(path, guard string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	if info.IsDir() {
		return false, fmt.Errorf("output path is a directory: %s", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	return hasGuardFlexible(path, b, guard), nil
}

// fastEqual reports true if existing file at path has the same bytes as newBytes.
// It avoids loading both into memory when sizes differ (fast path optimization).
func fastEqual(path string, newBytes []byte) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	// Quick size check (fast path - no file read needed)
	if int64(len(newBytes)) != info.Size() {
		return false, nil
	}

	// Same size: read and compare bytes
	old, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	return bytes.Equal(old, newBytes), nil
}

// writeIfChanged writes newBytes to path only if content differs from existing file.
// Uses atomic write (tmp file + rename) to prevent partial writes on crash/interrupt.
// Returns (changed bool, error).
func writeIfChanged(path string, newBytes []byte, mode os.FileMode) (bool, error) {
	// Check if content is the same
	same, err := fastEqual(path, newBytes)
	if err != nil {
		return false, err
	}
	if same {
		return false, nil // No change needed
	}

	// Content differs - perform atomic write
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, err
	}

	// Create temp file in same directory (ensures same filesystem for atomic rename)
	f, err := os.CreateTemp(dir, ".templr-*")
	if err != nil {
		return false, err
	}
	tmp := f.Name()
	defer func() { _ = os.Remove(tmp) }() // Cleanup on failure

	// Write to temp file
	if _, err := f.Write(newBytes); err != nil {
		_ = f.Close()
		return false, err
	}

	// Sync to disk (durability)
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return false, err
	}

	if err := f.Close(); err != nil {
		return false, err
	}

	// Set permissions
	if err := os.Chmod(tmp, mode); err != nil {
		return false, err
	}

	// Atomic rename (only visible point of update)
	if err := os.Rename(tmp, path); err != nil {
		return false, err
	}

	return true, nil
}

//nolint:gocyclo,cyclop // main orchestrates CLI flow; complexity is acceptable here.
func main() { //nolint:gocyclo,cyclop
	// Modes:
	//  - Single file: [-in FILE] [-out FILE] (stdin/stdout if omitted)
	//  - Multi-file (execute one entry): -dir DIR [-in ENTRY or define "root"] [-out FILE]
	//  - Walk tree (render all .tpl -> strip .tpl): -walk -src DIR -dst DIR
	in := flag.String("in", "", "Template file OR entry template name (if -dir is used). Omit for stdin (single-file).")
	out := flag.String("out", "", "Output file path (omit for stdout)")
	data := flag.String("data", "", "Path to base JSON or YAML data file")
	var files stringSlice
	flag.Var(&files, "f", "Additional values files (YAML/JSON). Repeatable.")
	var sets stringSlice
	flag.Var(&sets, "set", "key=value overrides. Repeatable. Supports dotted keys.")

	dir := flag.String("dir", "", "Directory containing *.tpl templates to parse together (multi-file mode)")
	walk := flag.Bool("walk", false, "Render all *.tpl under -src into -dst, mirroring paths and stripping .tpl")
	src := flag.String("src", "", "Templates root for -walk mode")
	dst := flag.String("dst", "", "Output root for -walk mode")

	ldelim := flag.String("ldelim", "{{", "Left delimiter")
	rdelim := flag.String("rdelim", "}}", "Right delimiter")
	strict := flag.Bool("strict", false, "Fail on missing keys")
	dryRun := flag.Bool("dry-run", false, "Preview which files would be rendered (no writes)")
	guard := flag.String("guard", "#templr generated", "Guard string required in existing files to allow overwrite")
	inject := flag.Bool("inject-guard", true, "Automatically insert the guard as a comment into written files (when supported)")
	helpers := flag.String("helpers", "_helpers*.tpl", "Glob pattern of helper templates to load (single-file mode). Set empty to skip.")
	var extraExts stringSlice
	flag.Var(&extraExts, "ext", "Additional template file extensions to treat as templates (e.g., md, txt). Repeatable; do not include the leading dot.")

	showVersion := flag.Bool("version", false, "Print version and exit")
	defaultMissing := flag.String("default-missing", "<no value>", "String to render when a variable/key is missing (works with missingkey=default)")
	noColor := flag.Bool("no-color", false, "Disable colored output (useful for CI/non-ANSI terminals)")

	flag.Parse()

	if *showVersion {
		fmt.Println(getVersion())
		return
	}

	// Values will be constructed per mode below.
	var values map[string]any
	var err error

	// Build function map
	var tpl *template.Template
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
		if tpl == nil {
			return "", fmt.Errorf("template not initialized")
		}
		if err := tpl.ExecuteTemplate(&b, name, data); err != nil {
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

	// Create template root with funcs & options
	tpl = template.New("root").Funcs(funcs).Option("missingkey=default")
	if *strict {
		tpl = tpl.Option("missingkey=error")
	}
	tpl = tpl.Delims(*ldelim, *rdelim)

	// ----- WALK MODE -----
	if *walk {
		if *src == "" || *dst == "" {
			errf(ExitGeneral, "args", "-walk requires -src and -dst")
		}
		absSrc, _ := filepath.Abs(*src)
		absDst, _ := filepath.Abs(*dst)

		// Build values: defaults (values.yaml) → -data → -f → --set
		values = map[string]any{}
		def, derr := loadDefaultValues(absSrc)
		if derr != nil {
			errf(ExitDataError, "data", "load default values: %v", derr)
		}
		values = deepMerge(values, def)
		if *data != "" {
			add, err := loadData(*data)
			if err != nil {
				errf(ExitDataError, "data", "load data: %v", err)
			}
			values = deepMerge(values, add)
		}
		for _, f := range files {
			add, err := loadData(f)
			if err != nil {
				errf(ExitDataError, "data", "load -f %s: %v", f, err)
			}
			values = deepMerge(values, add)
		}
		for _, kv := range sets {
			idx := strings.Index(kv, "=")
			if idx <= 0 {
				errf(ExitGeneral, "args", "--set expects key=value, got: %s", kv)
			}
			key := kv[:idx]
			val := parseScalar(kv[idx+1:])
			setByDottedKey(values, key, val)
		}
		// .Files resolves relative to src
		values["Files"] = FilesAPI{Root: absSrc}

		// Parse ALL templates (so includes/partials are available)
		allowExts := buildAllowedExts(extraExts)
		var names []string
		var sources map[string][]byte
		tpl, names, sources, err = readAllTplsIntoSet(tpl, absSrc, allowExts)
		if err != nil {
			errf(ExitTemplateError, "parse", "parse tree: %v", err)
		}

		// Compute helper-driven variables (templr.vars)
		if err := computeHelperVars(tpl, values); err != nil {
			errf(ExitTemplateError, "helpers", "%v", err)
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
				if *strict {
					strictErrf(rerr, sources, *noColor)
				}
				errf(ExitTemplateError, "render", "render error %s: %v", name, rerr)
			}
			// apply global default-missing replacement
			outBytes = applyDefaultMissing(outBytes, *defaultMissing)

			if isEmpty(outBytes) {
				if *dryRun {
					fmt.Printf("[dry-run] skip empty %s (no file created)\n", dstPath)
				}
				continue
			}

			// Guard check BEFORE any mkdir/write
			ok, gerr := canOverwrite(dstPath, *guard)
			if gerr != nil && !os.IsNotExist(gerr) {
				errf(ExitGeneral, "guard", "guard check %s: %v", dstPath, gerr)
			}
			if !ok {
				if *dryRun {
					fmt.Printf("[dry-run] skip (guard missing) %s\n", dstPath)
				} else {
					warnf("guard", "skip (guard missing) %s", dstPath)
				}
				continue
			}

			if *dryRun {
				simulated := outBytes
				if *inject {
					simulated = injectGuardForExt(dstPath, simulated, *guard)
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
			if *inject {
				outBytes = injectGuardForExt(dstPath, outBytes, *guard)
			}
			// Write only if content changed
			changed, err := writeIfChanged(dstPath, outBytes, 0o644)
			if err != nil {
				errf(ExitGeneral, "write", "write %s: %v", dstPath, err)
			}
			if changed {
				fmt.Printf("rendered %s -> %s\n", name, dstPath)
			}
		}

		// Cleanup: remove empty directories under dst
		if err := templr.PruneEmptyDirs(absDst); err != nil {
			errf(ExitGeneral, "io", "prune: %v", err)
		}
		return
	}

	// ----- MULTI-FILE DIR MODE -----
	if *dir != "" {
		absDir, _ := filepath.Abs(*dir)
		// Build values: defaults (values.yaml) → -data → -f → --set
		values = map[string]any{}
		def, derr := loadDefaultValues(absDir)
		if derr != nil {
			errf(ExitDataError, "data", "load default values: %v", derr)
		}
		values = deepMerge(values, def)
		if *data != "" {
			add, err := loadData(*data)
			if err != nil {
				errf(ExitDataError, "data", "load data: %v", err)
			}
			values = deepMerge(values, add)
		}
		for _, f := range files {
			add, err := loadData(f)
			if err != nil {
				errf(ExitDataError, "data", "load -f %s: %v", f, err)
			}
			values = deepMerge(values, add)
		}
		for _, kv := range sets {
			idx := strings.Index(kv, "=")
			if idx <= 0 {
				errf(ExitGeneral, "args", "--set expects key=value, got: %s", kv)
			}
			key := kv[:idx]
			val := parseScalar(kv[idx+1:])
			setByDottedKey(values, key, val)
		}
		values["Files"] = FilesAPI{Root: absDir}

		// Parse all *.tpl in dir using path-based names
		allowExts := buildAllowedExts(extraExts)
		var names []string
		var sources map[string][]byte
		tpl, names, sources, err = readAllTplsIntoSet(tpl, absDir, allowExts)
		if err != nil {
			errf(ExitTemplateError, "parse", "parse dir templates: %v", err)
		}

		// Compute helper-driven variables (templr.vars)
		if err := computeHelperVars(tpl, values); err != nil {
			errf(ExitTemplateError, "helpers", "%v", err)
		}

		entryName := ""
		if *in != "" {
			// If -in is a file path, convert to rel name; otherwise assume it's already a template name.
			if info, err := os.Stat(*in); err == nil && !info.IsDir() {
				if rel, er := filepath.Rel(absDir, *in); er == nil {
					entryName = filepath.ToSlash(rel)
				} else {
					entryName = filepath.Base(*in)
				}
			} else {
				entryName = *in
			}
		} else if tpl.Lookup("root") != nil {
			entryName = "root"
		} else if len(names) > 0 {
			entryName = names[0]
		} else {
			fmt.Fprintln(os.Stderr, "no templates found in -dir")
			os.Exit(1)
		}

		// render to buffer
		outBytes, rerr := renderToBuffer(tpl, entryName, values)
		if rerr != nil {
			if *strict {
				strictErrf(rerr, sources, *noColor)
			}
			errf(ExitTemplateError, "render", "%v", rerr)
		}
		// apply global default-missing replacement
		outBytes = applyDefaultMissing(outBytes, *defaultMissing)

		if isEmpty(outBytes) {
			target := "stdout"
			if *out != "" {
				target = *out
			}
			if *dryRun {
				fmt.Printf("[dry-run] skip empty render for entry %s -> %s\n", entryName, target)
				return
			}
			fmt.Fprintf(os.Stderr, "skipping empty render for entry %s -> %s\n", entryName, target)
			return
		}

		// If writing to a file, guard-verify when target exists
		if *out != "" {
			ok, gerr := canOverwrite(*out, *guard)
			if gerr != nil && !os.IsNotExist(gerr) {
				errf(ExitGeneral, "guard", "guard check %s: %v", *out, gerr)
			}
			if !ok {
				if *dryRun {
					fmt.Printf("[dry-run] skip (guard missing) %s\n", *out)
				} else {
					warnf("guard", "skip (guard missing) %s", *out)
				}
				return
			}
		}

		if *dryRun {
			target := "stdout"
			if *out != "" {
				target = *out
			}
			if *out != "" && *inject {
				simulated := injectGuardForExt(*out, outBytes, *guard)
				if !bytes.Equal(simulated, outBytes) {
					fmt.Printf("[dry-run] would inject guard into %s\n", *out)
				}
			}
			// Check if file would change
			if *out != "" {
				simToCheck := outBytes
				if *inject {
					simToCheck = injectGuardForExt(*out, outBytes, *guard)
				}
				same, _ := fastEqual(*out, simToCheck)
				if same {
					fmt.Printf("[dry-run] would skip unchanged %s\n", *out)
				} else {
					fmt.Printf("[dry-run] would render entry %s -> %s (changed)\n", entryName, target)
				}
			} else {
				fmt.Printf("[dry-run] would render entry %s -> %s\n", entryName, target)
			}
			return
		}

		// write (stdout or file)
		var w io.Writer = os.Stdout
		if *out != "" {
			// Optionally inject guard comment
			if *inject {
				outBytes = injectGuardForExt(*out, outBytes, *guard)
			}
			// Write only if content changed
			changed, err := writeIfChanged(*out, outBytes, 0o644)
			if err != nil {
				errf(ExitGeneral, "write", "write out: %v", err)
			}
			if changed {
				fmt.Printf("rendered entry %s -> %s\n", entryName, *out)
			}
			return
		}
		if _, err := w.Write(outBytes); err != nil {
			errf(ExitGeneral, "write", "%v", err)
		}
		return
	}

	// ----- SINGLE-FILE MODE -----
	// Determine Files.Root (dir of -in if present)
	filesRoot := "."
	if *in != "" {
		if info, err := os.Stat(*in); err == nil && !info.IsDir() {
			if abs, e := filepath.Abs(*in); e == nil {
				filesRoot = filepath.Dir(abs)
			}
		}
	}
	// Build values: defaults (values.yaml) → -data → -f → --set
	values = map[string]any{}
	def, derr := loadDefaultValues(filesRoot)
	if derr != nil {
		errf(ExitDataError, "data", "load default values: %v", derr)
	}
	values = deepMerge(values, def)
	if *data != "" {
		add, err := loadData(*data)
		if err != nil {
			errf(ExitDataError, "data", "load data: %v", err)
		}
		values = deepMerge(values, add)
	}
	for _, f := range files {
		add, err := loadData(f)
		if err != nil {
			errf(ExitDataError, "data", "load -f %s: %v", f, err)
		}
		values = deepMerge(values, add)
	}
	for _, kv := range sets {
		idx := strings.Index(kv, "=")
		if idx <= 0 {
			errf(ExitGeneral, "args", "--set expects key=value, got: %s", kv)
		}
		key := kv[:idx]
		val := parseScalar(kv[idx+1:])
		setByDottedKey(values, key, val)
	}
	values["Files"] = FilesAPI{Root: filesRoot}

	var srcBytes []byte
	sources := make(map[string][]byte)
	tplName := "stdin"
	if *in == "" {
		srcBytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			errf(ExitGeneral, "read", "read stdin: %v", err)
		}
	} else {
		srcBytes, err = os.ReadFile(*in)
		if err != nil {
			errf(ExitGeneral, "read", "read template: %v", err)
		}
		tplName = filepath.Base(*in)
	}
	sources[tplName] = srcBytes
	sources["root"] = srcBytes // Also map to "root" since that's what template.Parse uses

	// Load sidecar helpers in the same directory based on -helpers glob (default: _helpers.tpl)
	if filesRoot != "" && filesRoot != "." && *helpers != "" {
		pattern := filepath.Join(filesRoot, *helpers)
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			for _, hp := range matches {
				if b, e := os.ReadFile(hp); e == nil {
					helperName := filepath.ToSlash(filepath.Base(hp))
					sources[helperName] = b
					if _, e2 := tpl.New(helperName).Parse(string(b)); e2 != nil {
						errf(ExitTemplateError, "parse", "parse helper %s: %v", hp, e2)
					}
				}
			}
		}
	}

	tpl, err = tpl.Parse(string(srcBytes))
	if err != nil {
		errf(ExitTemplateError, "parse", "%v", err)
	}

	// Compute helper-driven variables (templr.vars)
	if err := computeHelperVars(tpl, values); err != nil {
		errf(ExitTemplateError, "helpers", "%v", err)
	}

	// render to buffer
	outBytes, rerr := renderToBuffer(tpl, "", values)
	if rerr != nil {
		if *strict {
			strictErrf(rerr, sources, *noColor)
		}
		errf(ExitTemplateError, "render", "%v", rerr)
	}
	// apply global default-missing replacement
	outBytes = applyDefaultMissing(outBytes, *defaultMissing)

	if isEmpty(outBytes) {
		target := "stdout"
		if *out != "" {
			target = *out
		}
		if *dryRun {
			srcLabel := "stdin"
			if *in != "" {
				srcLabel = *in
			}
			fmt.Printf("[dry-run] skip empty render %s -> %s\n", srcLabel, target)
			return
		}
		fmt.Fprintf(os.Stderr, "skipping empty render -> %s\n", target)
		return
	}

	// If writing to a file, guard-verify when target exists
	if *out != "" {
		ok, gerr := canOverwrite(*out, *guard)
		if gerr != nil && !os.IsNotExist(gerr) {
			errf(ExitGeneral, "guard", "guard check %s: %v", *out, gerr)
		}
		if !ok {
			if *dryRun {
				fmt.Printf("[dry-run] skip (guard missing) %s\n", *out)
				return
			}
			warnf("guard", "skip (guard missing) %s", *out)
			return
		}
	}

	if *dryRun {
		target := "stdout"
		if *out != "" {
			target = *out
		}
		srcLabel := "stdin"
		if *in != "" {
			srcLabel = *in
		}
		if *out != "" && *inject {
			simulated := injectGuardForExt(*out, outBytes, *guard)
			if !bytes.Equal(simulated, outBytes) {
				fmt.Printf("[dry-run] would inject guard into %s\n", *out)
			}
		}
		// Check if file would change
		if *out != "" {
			simToCheck := outBytes
			if *inject {
				simToCheck = injectGuardForExt(*out, outBytes, *guard)
			}
			same, _ := fastEqual(*out, simToCheck)
			if same {
				fmt.Printf("[dry-run] would skip unchanged %s\n", *out)
			} else {
				fmt.Printf("[dry-run] would render %s -> %s (changed)\n", srcLabel, target)
			}
		} else {
			fmt.Printf("[dry-run] would render %s -> %s\n", srcLabel, target)
		}
		return
	}

	// write (stdout or file)
	if *out != "" {
		// Optionally inject guard comment
		if *inject {
			outBytes = injectGuardForExt(*out, outBytes, *guard)
		}
		// Write only if content changed
		changed, err := writeIfChanged(*out, outBytes, 0o644)
		if err != nil {
			errf(ExitGeneral, "write", "write out: %v", err)
		}
		if changed {
			srcLabel := "stdin"
			if *in != "" {
				srcLabel = *in
			}
			fmt.Printf("rendered %s -> %s\n", srcLabel, *out)
		}
		return
	}
	if _, err := os.Stdout.Write(outBytes); err != nil {
		errf(ExitGeneral, "write", "%v", err)
	}
}

// loadDefaultValues attempts to load a default values file from baseDir.
// It prefers values.yaml, then values.yml. Missing files are not errors.
func loadDefaultValues(baseDir string) (map[string]any, error) {
	candidates := []string{"values.yaml", "values.yml"}
	out := map[string]any{}
	for _, name := range candidates {
		p := filepath.Join(baseDir, name)
		if _, err := os.Stat(p); err == nil {
			m, err := loadData(p)
			if err != nil {
				return nil, fmt.Errorf("load default %s: %w", p, err)
			}
			out = deepMerge(out, m)
			// stop at first found like Helm
			break
		}
	}
	return out, nil
}

// normalize strips UTF-8 BOM and converts CRLF to LF for consistent processing.
func normalize(content []byte) []byte {
	// Strip UTF-8 BOM if present
	if len(content) >= 3 && content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF {
		content = content[3:]
	}
	// Normalize CRLF -> LF
	return bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
}

// hasGuardFlexible checks if content contains the guard marker in a format
// appropriate for the file type. This mirrors the injection logic in injectGuardForExt.
func hasGuardFlexible(path string, content []byte, marker string) bool {
	// Normalize content (strip BOM, normalize line endings)
	b := normalize(content)

	base := strings.ToLower(filepath.Base(path))
	ext := strings.ToLower(filepath.Ext(path))

	// JSON: no comment support by default
	if ext == ".json" {
		// Fallback: check for raw marker (rare case)
		return bytes.Contains(b, []byte(marker))
	}

	// Build candidate patterns based on file type
	candidates := []string{marker} // raw marker as fallback

	switch {
	case base == "dockerfile":
		candidates = append(candidates, "# "+marker, "#"+marker)

	case ext == ".php" || ext == ".phtml":
		candidates = append(candidates,
			"// "+marker, "//"+marker,
			"<?php // "+marker+" ?>")

	case ext == ".css" || ext == ".scss":
		candidates = append(candidates, "/* "+marker+" */")

	case ext == ".html" || ext == ".htm" || ext == ".xml" || ext == ".md":
		candidates = append(candidates, "<!-- "+marker+" -->")

	case ext == ".sh" || ext == ".bash" || ext == ".zsh" || ext == ".env" ||
		ext == ".yml" || ext == ".yaml" || ext == ".toml" ||
		ext == ".ini" || ext == ".conf" ||
		ext == ".py" || ext == ".rb":
		candidates = append(candidates, "# "+marker, "#"+marker)

	default:
		// C-style languages: .js, .ts, .go, .java, .kt, .c, .cpp, .rs, etc.
		candidates = append(candidates, "// "+marker, "//"+marker)
	}

	// Check if any candidate is present
	for _, cand := range candidates {
		if bytes.Contains(b, []byte(cand)) {
			return true
		}
	}

	return false
}

// isShebang reports if content starts with a #! shebang.
func isShebang(content []byte) bool {
	if len(content) < 2 {
		return false
	}
	if content[0] != '#' || content[1] != '!' {
		return false
	}
	// only accept at very start (no BOM/whitespace)
	return true
}

// injectGuardForExt injects guard into content using a style determined by file path and content.
// Returns possibly-modified content. If injection is unsafe (e.g., JSON), returns original content.
func injectGuardForExt(path string, content []byte, guard string) []byte {
	if len(guard) == 0 || hasGuardFlexible(path, content, guard) {
		return content
	}

	base := strings.ToLower(filepath.Base(path))
	ext := strings.ToLower(filepath.Ext(path))

	// JSON: skip (no comments)
	if ext == ".json" {
		return content
	}

	// Dockerfile: no ext but known filename
	if base == "dockerfile" {
		return []byte("# " + guard + "\n" + string(content))
	}

	// Helpers
	addLineTop := func(prefix string) []byte {
		return []byte(prefix + guard + "\n" + string(content))
	}
	addBlockTop := func(open, closeToken string) []byte {
		return []byte(open + " " + guard + " " + closeToken + "\n" + string(content))
	}
	addAfterShebang := func(prefix string) []byte {
		// split first line
		idx := bytes.IndexByte(content, '\n')
		if idx == -1 {
			// single line shebang file; append guard after newline
			return append(append(content, []byte("\n"+prefix+guard+"\n")...), []byte{}...)
		}
		she := content[:idx+1]
		rest := content[idx+1:]
		return append(append(she, []byte(prefix+guard+"\n")...), rest...)
	}

	// Shell / Python / Ruby / Env / YAML-ish
	hashCommentExts := map[string]bool{
		".sh": true, ".bash": true, ".zsh": true, ".env": true,
		".yml": true, ".yaml": true, ".toml": true, ".ini": true, ".conf": true,
		".py": true, ".rb": true,
	}
	if hashCommentExts[ext] {
		if isShebang(content) {
			return addAfterShebang("# ")
		}
		return addLineTop("# ")
	}

	// PHP
	if ext == ".php" || ext == ".phtml" {
		// If starts with <?php, insert right after
		trimmed := bytes.TrimLeft(content, "\ufeff") // skip BOM if present
		if bytes.HasPrefix(trimmed, []byte("<?php")) {
			// find the first newline after the opening tag line
			idx := bytes.IndexByte(trimmed, '\n')
			if idx == -1 {
				// one-liner starting with <?php ... ; append guard on new line
				return append(trimmed, []byte("\n// "+guard+"\n")...)
			}
			head := trimmed[:idx+1]
			rest := trimmed[idx+1:]
			var buf bytes.Buffer
			buf.Write(head)
			buf.WriteString("// " + guard + "\n")
			buf.Write(rest)
			// If we trimmed a BOM, re-prepend it
			if !bytes.HasPrefix(content, []byte("<?php")) {
				return append([]byte("\ufeff"), buf.Bytes()...)
			}
			return buf.Bytes()
		}
		// Otherwise, safest: prepend a tiny php block with a comment.
		return []byte("<?php // " + guard + " ?>\n" + string(content))
	}

	// HTML / XML / Markdown
	markupExts := map[string]bool{".html": true, ".htm": true, ".xml": true, ".md": true}
	if markupExts[ext] {
		return addBlockTop("<!--", "-->")
	}

	// CSS / SCSS
	if ext == ".css" || ext == ".scss" {
		return addBlockTop("/*", "*/")
	}

	// JS / TS / Go / C / C++ / H / Java / Kotlin and similar line-comment langs
	slashSlashExts := map[string]bool{
		".js": true, ".ts": true, ".mjs": true, ".cjs": true,
		".go": true, ".java": true, ".kt": true, ".kts": true,
		".c": true, ".h": true, ".cpp": true, ".hpp": true, ".cc": true, ".hh": true,
		".rs": true, ".swift": true,
	}
	if slashSlashExts[ext] {
		return addLineTop("// ")
	}

	// Default: conservative line comment
	return addLineTop("# ")
}

// computeHelperVars executes an optional helper template named "templr.vars".
// If present, it should render YAML or JSON that will be deep-merged into the values map.
//
// Usage:
// In any parsed template file (commonly in _helpers.tpl), define:
// {{- define "templr.vars" -}}
// {{- $env := mustMerge (default (dict) .images.env) (default (dict) .mariadb.env) -}}
// {{ toYaml (dict "env" $env "nameSlug" (replace (lower .name) " " "-")) }}
// {{- end -}}
// The YAML/JSON produced will be deep-merged into the root values before rendering other templates.
func computeHelperVars(tpl *template.Template, values map[string]any) error {
	if tpl == nil {
		return nil
	}
	if tpl.Lookup("templr.vars") == nil {
		return nil
	}
	out, err := renderToBuffer(tpl, "templr.vars", values)
	if err != nil {
		return fmt.Errorf("templr.vars execute: %w", err)
	}
	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		return nil
	}
	m := map[string]any{}
	if err := yaml.Unmarshal(out, &m); err != nil {
		var jm any
		if jerr := json.Unmarshal(out, &jm); jerr != nil {
			return fmt.Errorf("templr.vars parse as YAML/JSON failed: %v / %v", err, jerr)
		}
		mm, ok := jm.(map[string]any)
		if !ok {
			return fmt.Errorf("templr.vars JSON did not produce an object")
		}
		deepMerge(values, mm)
		return nil
	}
	deepMerge(values, m)
	return nil
}
