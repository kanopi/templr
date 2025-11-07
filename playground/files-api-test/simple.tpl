# Simple Files API Test

## Test Exists
config.json exists: {{ .Files.Exists "data/config.json" }}
missing.txt exists: {{ .Files.Exists "data/missing.txt" }}

## Test Get
{{ .Files.Get "data/servers.txt" }}
