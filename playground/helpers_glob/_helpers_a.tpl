{{- define "templr.vars" -}}
{{ toYaml (dict "foo" "bar") }}
{{- end -}}
