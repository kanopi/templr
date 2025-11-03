{{- $_ := set . "newVar" "x" -}}
{{- $_ := setd . "nested.path" 123 -}}
{{- $m := mergeDeep (dict "a" 1 "b" (dict "c" 2)) (dict "b" (dict "d" 3)) -}}
newVar: {{ .newVar }}
nested:
{{ toYaml .nested | indent 2 }}
merged:
{{ toYaml $m | indent 2 }}
