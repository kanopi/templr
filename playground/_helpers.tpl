{{- define "templr.vars" -}}
{{- /* Safe lookups to avoid strict errors */ -}}
{{- $images   := (get . "images")   | default (dict) -}}
{{- $mariadb  := (get . "mariadb")  | default (dict) -}}
{{- $globalEnv := (get $images "env")   | default (dict) -}}
{{- $serviceEnv := (get $mariadb "env") | default (dict) -}}
{{- $env := mustMerge $globalEnv $serviceEnv -}}

{{- $nameVal := (get . "name") -}}
{{- $nameSlug := "" -}}
{{- if $nameVal -}}
  {{- $nameSlug = replace (lower (printf "%v" $nameVal)) " " "-" -}}
{{- end -}}

{{ toYaml (dict
  "env" $env
  "nameSlug" $nameSlug
) }}
{{- end -}}
