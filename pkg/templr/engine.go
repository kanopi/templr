package templr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"
)

type FilesAPI interface {
	Get(name string) (string, error)
}

type FilesMap map[string]string

func (m FilesMap) Get(name string) (string, error) {
	if s, ok := m[name]; ok {
		return s, nil
	}
	return "", fmt.Errorf("file %q not found", name)
}

type Options struct {
	Template       string
	Helpers        string
	ValuesYAML     string
	ValuesJSON     string
	Strict         bool
	DefaultMissing string
	Files          FilesAPI
	FuncMap        template.FuncMap

	InjectGuard   bool
	GuardMarker   string
}

type Result struct { Output string }

func defaultFuncMap() template.FuncMap {
	fm := sprig.FuncMap()
	fm["safe"] = func(v any, def string) string {
		if v == nil { return def }
		if s, ok := v.(string); ok {
			if len(bytes.TrimSpace([]byte(s))) == 0 { return def }
			return s
		}
		return fmt.Sprint(v)
	}
	return fm
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
	if repl == "" || repl == "<no value>" { return b }
	return bytes.ReplaceAll(b, []byte("<no value>"), []byte(repl))
}

func injectGuard(marker string, content []byte) []byte {
	if marker == "" { return content }
	var out bytes.Buffer
	out.WriteString(marker)
	if !bytes.HasPrefix(content, []byte("\n")) { out.WriteByte('\n') }
	out.Write(content)
	return out.Bytes()
}

func RenderSingle(opts Options) (Result, error) {
	values, err := loadValues(opts)
	if err != nil { return Result{}, err }

	funcs := defaultFuncMap()
	for k, v := range opts.FuncMap { funcs[k] = v }

	if opts.Files != nil {
		values["Files"] = map[string]any{ "Get": opts.Files.Get }
	}

	root := template.New("root").Funcs(funcs).Option("missingkey=default")
	if opts.Strict { root = root.Option("missingkey=error") }

	if opts.Helpers != "" {
		if _, err := root.Parse(opts.Helpers); err != nil { return Result{}, fmt.Errorf("helpers parse: %w", err) }
	}
	t, err := root.Parse(opts.Template)
	if err != nil { return Result{}, fmt.Errorf("template parse: %w", err) }

	var buf bytes.Buffer
	if err := t.Execute(&buf, values); err != nil { return Result{}, fmt.Errorf("render: %w", err) }

	out := applyDefaultMissing(buf.Bytes(), opts.DefaultMissing)
	if opts.InjectGuard && opts.GuardMarker != "" { out = injectGuard(opts.GuardMarker, out) }
	return Result{Output: string(out)}, nil
}
