{{- define "app.metadata" -}}
name: {{ .app.name }}
version: {{ .app.version }}
{{- end -}}

{{- define "app.config" -}}
environment: {{ .environment }}
region: {{ .region }}
{{- end -}}

{{- define "nested.deep" -}}
  level1: {{ .level1 }}
  level2:
    value: {{ .level2.value }}
    nested: {{ .level2.nested.deep }}
{{- end -}}

# Using templates with missing values
metadata:
  {{- include "app.metadata" . | nindent 2 }}

config:
  {{- include "app.config" . | nindent 2 }}

deep:
{{- include "nested.deep" . | nindent 2 }}

# Direct reference to missing var after templates
directRef: {{ .directMissing }}
