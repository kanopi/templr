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

	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"
)

// Build-time variables (overridable via -ldflags)
var (
	Version string // preferred explicit version (e.g., a tag)
)

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
func readAllTplsIntoSet(tpl *template.Template, root string, allowExts map[string]bool) (*template.Template, []string, error) {
	var names []string
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
		_, err = tpl.New(rel).Parse(string(src))
		if err != nil {
			return fmt.Errorf("parse %s: %w", rel, err)
		}
		names = append(names, rel)
		return nil
	})
	return tpl, names, err
}

// shouldRender returns false for "partials" (files whose base name starts with "_").
func shouldRender(rel string) bool {
	base := filepath.Base(rel)
	return !strings.HasPrefix(base, "_")
}

// isEmpty reports true if, after removing *all* whitespace (anywhere), no characters remain.
func isEmpty(b []byte) bool {
	return len(bytes.Fields(b)) == 0
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

// canOverwrite checks guard when target exists.
// If file doesn't exist → allowed. If exists → allowed only if guard is present.
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
	return bytes.Contains(b, []byte(guard)), nil
}

// pruneEmptyDirs removes empty directories under root (bottom-up).
func pruneEmptyDirs(root string) error {
	var dirs []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			dirs = append(dirs, p)
		}
		return nil
	})
	if err != nil {
		return err
	}
	// deepest-first: longer paths first
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, d := range dirs {
		if d == root {
			continue
		}
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		if len(entries) == 0 {
			_ = os.Remove(d)
		}
	}
	return nil
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

	// Create template root with funcs & options
	tpl = template.New("root").Funcs(funcs).Option("missingkey=default")
	if *strict {
		tpl = tpl.Option("missingkey=error")
	}
	tpl = tpl.Delims(*ldelim, *rdelim)

	// ----- WALK MODE -----
	if *walk {
		if *src == "" || *dst == "" {
			fmt.Fprintln(os.Stderr, "-walk requires -src and -dst")
			os.Exit(1)
		}
		absSrc, _ := filepath.Abs(*src)
		absDst, _ := filepath.Abs(*dst)

		// Build values: defaults (values.yaml) → -data → -f → --set
		values = map[string]any{}
		def, derr := loadDefaultValues(absSrc)
		if derr != nil {
			fmt.Fprintln(os.Stderr, "load default values:", derr)
			os.Exit(1)
		}
		values = deepMerge(values, def)
		if *data != "" {
			add, err := loadData(*data)
			if err != nil {
				fmt.Fprintln(os.Stderr, "load data:", err)
				os.Exit(1)
			}
			values = deepMerge(values, add)
		}
		for _, f := range files {
			add, err := loadData(f)
			if err != nil {
				fmt.Fprintln(os.Stderr, "load -f:", f, "error:", err)
				os.Exit(1)
			}
			values = deepMerge(values, add)
		}
		for _, kv := range sets {
			idx := strings.Index(kv, "=")
			if idx <= 0 {
				fmt.Fprintln(os.Stderr, "--set expects key=value, got:", kv)
				os.Exit(1)
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
		tpl, names, err = readAllTplsIntoSet(tpl, absSrc, allowExts)
		if err != nil {
			fmt.Fprintln(os.Stderr, "parse tree:", err)
			os.Exit(1)
		}

		// Compute helper-driven variables (templr.vars)
		if err := computeHelperVars(tpl, values); err != nil {
			fmt.Fprintln(os.Stderr, "helpers vars:", err)
			os.Exit(1)
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
				fmt.Fprintf(os.Stderr, "render error %s: %v\n", name, rerr)
				os.Exit(1)
			}
			if isEmpty(outBytes) {
				if *dryRun {
					fmt.Printf("[dry-run] skip empty %s (no file created)\n", dstPath)
				}
				continue
			}

			// Guard check BEFORE any mkdir/write
			ok, gerr := canOverwrite(dstPath, *guard)
			if gerr != nil && !os.IsNotExist(gerr) {
				fmt.Fprintf(os.Stderr, "guard check %s: %v\n", dstPath, gerr)
				os.Exit(1)
			}
			if !ok {
				if *dryRun {
					fmt.Printf("[dry-run] skip (guard missing) %s\n", dstPath)
				} else {
					fmt.Fprintf(os.Stderr, "skip (guard missing) %s\n", dstPath)
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
				fmt.Printf("[dry-run] would render %s -> %s\n", name, dstPath)
				continue
			}

			// Optionally inject guard comment
			if *inject {
				outBytes = injectGuardForExt(dstPath, outBytes, *guard)
			}
			// Only now create directory and write
			if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
				fmt.Fprintln(os.Stderr, "mkdir:", dstPath, "err:", err)
				os.Exit(1)
			}
			if err := os.WriteFile(dstPath, outBytes, 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "write:", dstPath, "err:", err)
				os.Exit(1)
			}
			fmt.Printf("rendered %s -> %s\n", name, dstPath)
		}

		// Cleanup: remove empty directories under dst
		if err := pruneEmptyDirs(absDst); err != nil {
			fmt.Fprintln(os.Stderr, "prune:", err)
			os.Exit(1)
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
			fmt.Fprintln(os.Stderr, "load default values:", derr)
			os.Exit(1)
		}
		values = deepMerge(values, def)
		if *data != "" {
			add, err := loadData(*data)
			if err != nil {
				fmt.Fprintln(os.Stderr, "load data:", err)
				os.Exit(1)
			}
			values = deepMerge(values, add)
		}
		for _, f := range files {
			add, err := loadData(f)
			if err != nil {
				fmt.Fprintln(os.Stderr, "load -f:", f, "error:", err)
				os.Exit(1)
			}
			values = deepMerge(values, add)
		}
		for _, kv := range sets {
			idx := strings.Index(kv, "=")
			if idx <= 0 {
				fmt.Fprintln(os.Stderr, "--set expects key=value, got:", kv)
				os.Exit(1)
			}
			key := kv[:idx]
			val := parseScalar(kv[idx+1:])
			setByDottedKey(values, key, val)
		}
		values["Files"] = FilesAPI{Root: absDir}

		// Parse all *.tpl in dir using path-based names
		allowExts := buildAllowedExts(extraExts)
		var names []string
		tpl, names, err = readAllTplsIntoSet(tpl, absDir, allowExts)
		if err != nil {
			fmt.Fprintln(os.Stderr, "parse dir templates:", err)
			os.Exit(1)
		}

		// Compute helper-driven variables (templr.vars)
		if err := computeHelperVars(tpl, values); err != nil {
			fmt.Fprintln(os.Stderr, "helpers vars:", err)
			os.Exit(1)
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
			fmt.Fprintln(os.Stderr, "render:", rerr)
			os.Exit(1)
		}
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
				fmt.Fprintf(os.Stderr, "guard check %s: %v\n", *out, gerr)
				os.Exit(1)
			}
			if !ok {
				if *dryRun {
					fmt.Printf("[dry-run] skip (guard missing) %s\n", *out)
				} else {
					fmt.Fprintf(os.Stderr, "skip (guard missing) %s\n", *out)
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
			fmt.Printf("[dry-run] would render entry %s -> %s\n", entryName, target)
			return
		}

		// write (stdout or file)
		var w io.Writer = os.Stdout
		if *out != "" {
			// Optionally inject guard comment
			if *inject {
				outBytes = injectGuardForExt(*out, outBytes, *guard)
			}
			if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
				fmt.Fprintln(os.Stderr, "mkdir out dir:", err)
				os.Exit(1)
			}
			if err := os.WriteFile(*out, outBytes, 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "write out:", err)
				os.Exit(1)
			}
			return
		}
		if _, err := w.Write(outBytes); err != nil {
			fmt.Fprintln(os.Stderr, "write:", err)
			os.Exit(1)
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
		fmt.Fprintln(os.Stderr, "load default values:", derr)
		os.Exit(1)
	}
	values = deepMerge(values, def)
	if *data != "" {
		add, err := loadData(*data)
		if err != nil {
			fmt.Fprintln(os.Stderr, "load data:", err)
			os.Exit(1)
		}
		values = deepMerge(values, add)
	}
	for _, f := range files {
		add, err := loadData(f)
		if err != nil {
			fmt.Fprintln(os.Stderr, "load -f:", f, "error:", err)
			os.Exit(1)
		}
		values = deepMerge(values, add)
	}
	for _, kv := range sets {
		idx := strings.Index(kv, "=")
		if idx <= 0 {
			fmt.Fprintln(os.Stderr, "--set expects key=value, got:", kv)
			os.Exit(1)
		}
		key := kv[:idx]
		val := parseScalar(kv[idx+1:])
		setByDottedKey(values, key, val)
	}
	values["Files"] = FilesAPI{Root: filesRoot}

	var srcBytes []byte
	if *in == "" {
		srcBytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "read stdin:", err)
			os.Exit(1)
		}
	} else {
		srcBytes, err = os.ReadFile(*in)
		if err != nil {
			fmt.Fprintln(os.Stderr, "read template:", err)
			os.Exit(1)
		}
	}

	// Load sidecar helpers in the same directory based on -helpers glob (default: _helpers.tpl)
	if filesRoot != "" && filesRoot != "." && *helpers != "" {
		pattern := filepath.Join(filesRoot, *helpers)
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			for _, hp := range matches {
				if b, e := os.ReadFile(hp); e == nil {
					if _, e2 := tpl.New(filepath.ToSlash(filepath.Base(hp))).Parse(string(b)); e2 != nil {
						fmt.Fprintf(os.Stderr, "parse helper %s: %v\n", hp, e2)
						os.Exit(1)
					}
				}
			}
		}
	}

	tpl, err = tpl.Parse(string(srcBytes))
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse:", err)
		os.Exit(1)
	}

	// Compute helper-driven variables (templr.vars)
	if err := computeHelperVars(tpl, values); err != nil {
		fmt.Fprintln(os.Stderr, "helpers vars:", err)
		os.Exit(1)
	}

	// render to buffer
	outBytes, rerr := renderToBuffer(tpl, "", values)
	if rerr != nil {
		fmt.Fprintln(os.Stderr, "render:", rerr)
		os.Exit(1)
	}
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
			fmt.Fprintf(os.Stderr, "guard check %s: %v\n", *out, gerr)
			os.Exit(1)
		}
		if !ok {
			if *dryRun {
				fmt.Printf("[dry-run] skip (guard missing) %s\n", *out)
				return
			}
			fmt.Fprintf(os.Stderr, "skip (guard missing) %s\n", *out)
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
		fmt.Printf("[dry-run] would render %s -> %s\n", srcLabel, target)
		return
	}

	// write (stdout or file)
	if *out != "" {
		// Optionally inject guard comment
		if *inject {
			outBytes = injectGuardForExt(*out, outBytes, *guard)
		}
		if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "mkdir out dir:", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*out, outBytes, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "write out:", err)
			os.Exit(1)
		}
		return
	}
	if _, err := os.Stdout.Write(outBytes); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
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

// hasGuard checks if content already contains the guard string.
func hasGuard(content []byte, guard string) bool {
	return bytes.Contains(content, []byte(guard))
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
	if len(guard) == 0 || hasGuard(content, guard) {
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
