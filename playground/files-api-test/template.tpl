# Files API Test Template

## Test 1: Exists
config.json exists: {{ .Files.Exists "data/config.json" }}
missing.txt exists: {{ .Files.Exists "data/missing.txt" }}

## Test 2: Stat
{{- $stat := .Files.Stat "data/config.json" }}
File: {{ $stat.Name }}
Size: {{ $stat.Size }} bytes
Modified: {{ $stat.ModTime }}

## Test 3: Lines
Servers from servers.txt:
{{- range $idx, $line := .Files.Lines "data/servers.txt" }}
  {{ $idx }}: {{ $line }}
{{- end }}

## Test 4: ReadDir
Files in data/ directory:
{{- range .Files.ReadDir "data" }}
  - {{ . }}
{{- end }}

## Test 5: AsBase64
TLS Certificate (base64):
{{ .Files.AsBase64 "certs/tls.crt" | trunc 60 }}...

## Test 6: AsJSON
{{- $config := .Files.AsJSON "data/config.json" }}
Config loaded from JSON:
  App: {{ $config.app }}
  Version: {{ $config.version }}
  Replicas: {{ $config.replicas }}

## Test 7: AsYAML
{{- $values := .Files.AsYAML "data/values.yaml" }}
Values loaded from YAML:
  Database: {{ $values.database.host }}:{{ $values.database.port }}
  Environment: {{ $values.environment }}

## Test 8: GlobDetails
All files in data/:
{{- range .Files.GlobDetails "data/*" }}
  - {{ .Name }} ({{ .Size }} bytes, modified {{ .ModTime }})
{{- end }}

## Test 9: AsLines (alias for Lines)
{{- $lines := .Files.AsLines "data/servers.txt" }}
Total servers: {{ len $lines }}
