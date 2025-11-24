// Package templr provides a reusable rendering engine for templr.
// It exposes a single-file renderer with Sprig functions, a `safe` helper,
// default-missing replacement, optional strict mode, and a minimal .Files API.
package templr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"gopkg.in/yaml.v3"
)

// FilesAPI abstracts the `.Files` helpers for in-memory or other backends.
// Implementations should return an error when a file is not found.
type FilesAPI interface {
	Get(name string) (string, error)
}

// FilesMap is a simple in-memory `.Files` implementation backed by a map.
type FilesMap map[string]string

// Get returns the file content by name or an error if not found.
func (m FilesMap) Get(name string) (string, error) {
	if s, ok := m[name]; ok {
		return s, nil
	}
	return "", fmt.Errorf("file %q not found", name)
}

// Options configures a single in-memory template render.
// Set Template/Helpers to the text to parse; provide ValuesYAML or ValuesJSON
// for data. Strict toggles missingkey=error. DefaultMissing replaces "<no value>"
// in the final output. Files can provide a `.Files` API. InjectGuard/GuardMarker
// optionally prepend a guard header to the output.
type Options struct {
	Template       string
	Helpers        string
	ValuesYAML     string
	ValuesJSON     string
	Strict         bool
	DefaultMissing string
	Files          FilesAPI
	FuncMap        template.FuncMap
	WarnFunc       func(string) // Function to call for warnings

	InjectGuard bool
	GuardMarker string
}

// Result is the successful render result.
// Output contains the fully rendered template text.
type Result struct{ Output string }

// defaultFuncMapWithOptions creates function map with options (for RenderSingle)
func defaultFuncMapWithOptions(tpl **template.Template, strict bool, defaultMissing string, warnFunc func(string)) template.FuncMap {
	return BuildFuncMapWithOptions(tpl, &FuncMapOptions{
		Strict:         strict,
		DefaultMissing: defaultMissing,
		WarnFunc:       warnFunc,
	})
}

func loadValues(o Options) (map[string]any, error) {
	vals := map[string]any{}
	switch {
	case o.ValuesYAML != "":
		if err := yaml.Unmarshal([]byte(o.ValuesYAML), &vals); err != nil {
			if err2 := json.Unmarshal([]byte(o.ValuesYAML), &vals); err2 != nil {
				return nil, fmt.Errorf("values decode failed: yaml=%v json=%v", err, err2)
			}
		}
	case o.ValuesJSON != "":
		if err := json.Unmarshal([]byte(o.ValuesJSON), &vals); err != nil {
			if err2 := yaml.Unmarshal([]byte(o.ValuesJSON), &vals); err2 != nil {
				return nil, fmt.Errorf("values decode failed: json=%v yaml=%v", err, err2)
			}
		}
	}
	return vals, nil
}

func applyDefaultMissing(b []byte, repl string) []byte {
	if repl == "" || repl == "<no value>" {
		return b
	}
	return bytes.ReplaceAll(b, []byte("<no value>"), []byte(repl))
}

func injectGuard(marker string, content []byte) []byte {
	if marker == "" {
		return content
	}
	var out bytes.Buffer
	out.WriteString(marker)
	if !bytes.HasPrefix(content, []byte("\n")) {
		out.WriteByte('\n')
	}
	out.Write(content)
	return out.Bytes()
}

// RenderSingle renders one in-memory template string using the provided Options.
// It supports Sprig, the `safe` helper, optional `.Files`, strict mode, and
// default-missing replacement. Helpers (if provided) are parsed before Template.
func RenderSingle(opts Options) (Result, error) {
	values, err := loadValues(opts)
	if err != nil {
		return Result{}, err
	}

	if opts.Files != nil {
		values["Files"] = map[string]any{"Get": opts.Files.Get}
	}

	// Create template first
	root := template.New("root").Option("missingkey=default")
	if opts.Strict {
		root = root.Option("missingkey=error")
	}

	// Build funcmap with reference to root template for include function
	funcs := defaultFuncMapWithOptions(&root, opts.Strict, opts.DefaultMissing, opts.WarnFunc)
	for k, v := range opts.FuncMap {
		funcs[k] = v
	}
	root = root.Funcs(funcs)

	if opts.Helpers != "" {
		if _, err := root.Parse(opts.Helpers); err != nil {
			return Result{}, fmt.Errorf("helpers parse: %w", err)
		}
	}
	t, err := root.Parse(opts.Template)
	if err != nil {
		return Result{}, fmt.Errorf("template parse: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, values); err != nil {
		return Result{}, fmt.Errorf("render: %w", err)
	}

	out := applyDefaultMissing(buf.Bytes(), opts.DefaultMissing)
	if opts.InjectGuard && opts.GuardMarker != "" {
		out = injectGuard(opts.GuardMarker, out)
	}
	return Result{Output: string(out)}, nil
}
