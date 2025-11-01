{{- $root := . -}}
{{- range $root.Files.Glob "files/*.txt" -}}
- {{ . }} ({{ len (($root.Files.Get .)) }} bytes)
{{- end -}}
