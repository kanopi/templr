{{/*
Helper templates with missing variables
*/}}

{{- define "app.fullname" -}}
{{ .Release.Name }}-{{ .Chart.Name }}
{{- end -}}

{{- define "app.labels" -}}
app: {{ include "app.fullname" . }}
version: {{ .Chart.Version }}
tier: {{ .tier }}
{{- end -}}

{{- define "app.selector" -}}
app: {{ include "app.fullname" . }}
{{- end -}}

{{- define "container.image" -}}
image: {{ .image.registry }}/{{ .image.repository }}:{{ .image.tag }}
{{- end -}}

{{- define "resource.limits" -}}
limits:
  memory: {{ .resources.limits.memory }}
  cpu: {{ .resources.limits.cpu }}
requests:
  memory: {{ .resources.requests.memory }}
  cpu: {{ .resources.requests.cpu }}
{{- end -}}
