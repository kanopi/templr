{{ include "banner" . }}

image: {{ .image | default "busybox:latest" }}
tags:
{{- range .tags }}
- {{ . }}
{{- end }}
