//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/kanopi/templr/pkg/templr"
)

type in struct {
	Template       string            `json:"template"`
	Values         string            `json:"values"`
	Helpers        string            `json:"helpers"`
	DefaultMissing string            `json:"defaultMissing"`
	Strict         bool              `json:"strict"`
	Files          map[string]string `json:"files"`
	InjectGuard    bool              `json:"injectGuard"`
	GuardMarker    string            `json:"guardMarker"`
}

type out struct {
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

func render(this js.Value, args []js.Value) any {
	if len(args) != 1 {
		return toJS(out{Error: "templrRender expects one JSON argument"})
	}
	var req in
	if err := json.Unmarshal([]byte(args[0].String()), &req); err != nil {
		return toJS(out{Error: "bad JSON: " + err.Error()})
	}

	opts := templr.Options{
		Template:       req.Template,
		Helpers:        req.Helpers,
		ValuesYAML:     req.Values,
		Strict:         req.Strict,
		DefaultMissing: req.DefaultMissing,
		InjectGuard:    req.InjectGuard,
		GuardMarker:    req.GuardMarker,
	}
	if len(req.Files) > 0 {
		opts.Files = templr.FilesMap(req.Files)
	}

	res, err := templr.RenderSingle(opts)
	if err != nil {
		return toJS(out{Error: err.Error()})
	}
	return toJS(out{Output: res.Output})
}

func toJS(v any) js.Value { b, _ := json.Marshal(v); return js.ValueOf(string(b)) }

func main() {
	js.Global().Set("templrRender", js.FuncOf(render))
	select {}
}
