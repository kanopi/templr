package app

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"gopkg.in/yaml.v3"
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

	var tplName string
	var lineNum int
	var expr string
	var missingKey string

	// Try to parse template name and line number
	if strings.HasPrefix(errMsg, "template: ") {
		rest := errMsg[10:]
		if idx := strings.Index(rest, ":"); idx > 0 {
			tplName = rest[:idx]
			rest = rest[idx+1:]
			if idx2 := strings.Index(rest, ":"); idx2 > 0 {
				if ln, e := strconv.Atoi(rest[:idx2]); e == nil {
					lineNum = ln
				}
			}
		}
	}

	// Extract the expression that failed
	if start := strings.Index(errMsg, "at <"); start >= 0 {
		start += 4
		if end := strings.Index(errMsg[start:], ">"); end >= 0 {
			expr = errMsg[start : start+end]
		}
	}

	// Extract missing key
	if strings.Contains(errMsg, "map has no entry for key") {
		if start := strings.Index(errMsg, `key "`); start >= 0 {
			start += 5
			if end := strings.Index(errMsg[start:], `"`); end >= 0 {
				missingKey = errMsg[start : start+end]
			}
		}
	}

	var buf bytes.Buffer
	buf.WriteString(colorize(colorRed+colorBold, "✗ Strict Mode Error") + "\n")

	if tplName != "" && lineNum > 0 {
		buf.WriteString(colorize(colorCyan, fmt.Sprintf("  %s:%d", tplName, lineNum)) + "\n\n")

		if src, ok := templateSources[tplName]; ok {
			lines := bytes.Split(src, []byte("\n"))
			if lineNum > 0 && lineNum <= len(lines) {
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
						buf.WriteString(colorize(colorGray, lineNumStr) + " | ")
						buf.WriteString(colorize(colorRed, string(lines[i])) + "\n")
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

	if expr != "" {
		buf.WriteString(colorize(colorRed, "  Missing: ") + expr + "\n")
	}
	if missingKey != "" {
		buf.WriteString(colorize(colorRed, "  Key: ") + missingKey + "\n")
	}

	buf.WriteString("\n")
	buf.WriteString(colorize(colorGray, "  Details: "+errMsg) + "\n\n")

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

// Get reads a file and returns its contents as a string.
func (f FilesAPI) Get(path string) (string, error) {
	b, err := os.ReadFile(filepath.Join(f.Root, path))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// GetBytes reads a file and returns its contents as a byte slice.
func (f FilesAPI) GetBytes(path string) ([]byte, error) {
	return os.ReadFile(filepath.Join(f.Root, path))
}

// Glob returns files matching the given glob pattern relative to the root directory.
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

// Exists checks if a file or directory exists at the given path.
func (f FilesAPI) Exists(path string) bool {
	_, err := os.Stat(filepath.Join(f.Root, path))
	return err == nil
}

// FileInfo contains metadata about a file.
type FileInfo struct {
	Name    string
	Size    int64
	Mode    string
	ModTime string
	IsDir   bool
}

// Stat returns metadata about a file.
func (f FilesAPI) Stat(path string) (FileInfo, error) {
	fi, err := os.Stat(filepath.Join(f.Root, path))
	if err != nil {
		return FileInfo{}, err
	}

	return FileInfo{
		Name:    fi.Name(),
		Size:    fi.Size(),
		Mode:    fi.Mode().String(),
		ModTime: fi.ModTime().Format("2006-01-02T15:04:05Z07:00"),
		IsDir:   fi.IsDir(),
	}, nil
}

// Lines reads a file and returns its lines as a slice of strings.
func (f FilesAPI) Lines(path string) ([]string, error) {
	content, err := f.Get(path)
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSuffix(content, "\n"), "\n"), nil
}

// ReadDir returns a list of file and directory names in the given directory.
func (f FilesAPI) ReadDir(path string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(f.Root, path))
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

// GlobDetails returns file metadata for all files matching the given pattern.
func (f FilesAPI) GlobDetails(pat string) ([]FileInfo, error) {
	paths, err := f.Glob(pat)
	if err != nil {
		return nil, err
	}

	infos := make([]FileInfo, 0, len(paths))
	for _, p := range paths {
		if info, err := f.Stat(p); err == nil {
			infos = append(infos, info)
		}
	}
	return infos, nil
}

// AsBase64 reads a file and returns its contents as a base64-encoded string.
func (f FilesAPI) AsBase64(path string) (string, error) {
	b, err := f.GetBytes(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// AsHex reads a file and returns its contents as a hexadecimal string.
func (f FilesAPI) AsHex(path string) (string, error) {
	b, err := f.GetBytes(path)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// AsDataURL reads a file and returns it as a data URL (for embedding in HTML/CSS).
// If mimeType is empty, it will be auto-detected from the file extension.
func (f FilesAPI) AsDataURL(path string, mimeType string) (string, error) {
	b, err := f.GetBytes(path)
	if err != nil {
		return "", err
	}

	if mimeType == "" {
		mimeType = detectMimeType(path)
	}

	encoded := base64.StdEncoding.EncodeToString(b)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded), nil
}

// AsLines reads a file and returns its lines as a slice (alias for Lines).
func (f FilesAPI) AsLines(path string) ([]string, error) {
	return f.Lines(path)
}

// AsJSON reads a JSON file and returns it as a map.
func (f FilesAPI) AsJSON(path string) (map[string]any, error) {
	b, err := f.GetBytes(path)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("parse JSON from %s: %w", path, err)
	}
	return result, nil
}

// AsYAML reads a YAML file and returns it as a map.
func (f FilesAPI) AsYAML(path string) (map[string]any, error) {
	b, err := f.GetBytes(path)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := yaml.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("parse YAML from %s: %w", path, err)
	}
	return result, nil
}

// detectMimeType returns the MIME type based on file extension.
func detectMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".ico":
		return "image/x-icon"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".html", ".htm":
		return "text/html"
	case ".pdf":
		return "application/pdf"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	default:
		return "application/octet-stream"
	}
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
func trimAnyExt(name string, allowExts map[string]bool) string {
	lower := strings.ToLower(name)
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

// readAllTplsIntoSet parses every allowed template file under root into the given template set.
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
		rel = filepath.ToSlash(rel)
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		sources[rel] = src
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

// isEmpty reports true if, after normalizing line endings and stripping BOM, nothing remains.
func isEmpty(b []byte) bool {
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		b = b[3:]
	}
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
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
func applyDefaultMissing(out []byte, replacement string) []byte {
	if replacement == "" || replacement == "<no value>" {
		return out
	}
	return bytes.ReplaceAll(out, []byte("<no value>"), []byte(replacement))
}

// canOverwrite checks guard when target exists.
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
func fastEqual(path string, newBytes []byte) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	if int64(len(newBytes)) != info.Size() {
		return false, nil
	}

	old, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	return bytes.Equal(old, newBytes), nil
}

// writeIfChanged writes newBytes to path only if content differs from existing file.
func writeIfChanged(path string, newBytes []byte, mode os.FileMode) (bool, error) {
	same, err := fastEqual(path, newBytes)
	if err != nil {
		return false, err
	}
	if same {
		return false, nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, err
	}

	f, err := os.CreateTemp(dir, ".templr-*")
	if err != nil {
		return false, err
	}
	tmp := f.Name()
	defer func() { _ = os.Remove(tmp) }()

	if _, err := f.Write(newBytes); err != nil {
		_ = f.Close()
		return false, err
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()
		return false, err
	}

	if err := f.Close(); err != nil {
		return false, err
	}

	if err := os.Chmod(tmp, mode); err != nil {
		return false, err
	}

	if err := os.Rename(tmp, path); err != nil {
		return false, err
	}

	return true, nil
}

// loadDefaultValues attempts to load a default values file from baseDir.
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
			break
		}
	}
	return out, nil
}

// normalize strips UTF-8 BOM and converts CRLF to LF for consistent processing.
func normalize(content []byte) []byte {
	if len(content) >= 3 && content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF {
		content = content[3:]
	}
	return bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
}

// hasGuardFlexible checks if content contains the guard marker.
func hasGuardFlexible(path string, content []byte, marker string) bool {
	b := normalize(content)
	base := strings.ToLower(filepath.Base(path))
	ext := strings.ToLower(filepath.Ext(path))

	if ext == ".json" {
		return bytes.Contains(b, []byte(marker))
	}

	candidates := []string{marker}

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
		candidates = append(candidates, "// "+marker, "//"+marker)
	}

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
	return content[0] == '#' && content[1] == '!'
}

// injectGuardForExt injects guard into content using a style determined by file path.
func injectGuardForExt(path string, content []byte, guard string) []byte {
	if len(guard) == 0 || hasGuardFlexible(path, content, guard) {
		return content
	}

	base := strings.ToLower(filepath.Base(path))
	ext := strings.ToLower(filepath.Ext(path))

	if ext == ".json" {
		return content
	}

	if base == "dockerfile" {
		return []byte("# " + guard + "\n" + string(content))
	}

	addLineTop := func(prefix string) []byte {
		return []byte(prefix + guard + "\n" + string(content))
	}
	addBlockTop := func(open, closeToken string) []byte {
		return []byte(open + " " + guard + " " + closeToken + "\n" + string(content))
	}
	addAfterShebang := func(prefix string) []byte {
		idx := bytes.IndexByte(content, '\n')
		if idx == -1 {
			return append(append(content, []byte("\n"+prefix+guard+"\n")...), []byte{}...)
		}
		she := content[:idx+1]
		rest := content[idx+1:]
		return append(append(she, []byte(prefix+guard+"\n")...), rest...)
	}

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

	if ext == ".php" || ext == ".phtml" {
		trimmed := bytes.TrimLeft(content, "\ufeff")
		if bytes.HasPrefix(trimmed, []byte("<?php")) {
			idx := bytes.IndexByte(trimmed, '\n')
			if idx == -1 {
				return append(trimmed, []byte("\n// "+guard+"\n")...)
			}
			head := trimmed[:idx+1]
			rest := trimmed[idx+1:]
			var buf bytes.Buffer
			buf.Write(head)
			buf.WriteString("// " + guard + "\n")
			buf.Write(rest)
			if !bytes.HasPrefix(content, []byte("<?php")) {
				return append([]byte("\ufeff"), buf.Bytes()...)
			}
			return buf.Bytes()
		}
		return []byte("<?php // " + guard + " ?>\n" + string(content))
	}

	markupExts := map[string]bool{".html": true, ".htm": true, ".xml": true, ".md": true}
	if markupExts[ext] {
		return addBlockTop("<!--", "-->")
	}

	if ext == ".css" || ext == ".scss" {
		return addBlockTop("/*", "*/")
	}

	slashSlashExts := map[string]bool{
		".js": true, ".ts": true, ".mjs": true, ".cjs": true,
		".go": true, ".java": true, ".kt": true, ".kts": true,
		".c": true, ".h": true, ".cpp": true, ".hpp": true, ".cc": true, ".hh": true,
		".rs": true, ".swift": true,
	}
	if slashSlashExts[ext] {
		return addLineTop("// ")
	}

	return addLineTop("# ")
}

// computeHelperVars executes an optional helper template named "templr.vars".
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
