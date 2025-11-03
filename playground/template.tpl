app: {{ .app | default "demo" }}
replicas: {{ .replicas | default 1 }}
features:
{{- range .features }}
  - {{ . }}
{{- end }}
