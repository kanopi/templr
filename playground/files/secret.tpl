kind: Secret
data:
  cert: |
{{ (.Files.Get "certs/tls.crt") | trim | indent 4 }}
