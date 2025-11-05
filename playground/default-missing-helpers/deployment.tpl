apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "app.fullname" . }}
  labels:
    {{- include "app.labels" . | nindent 4 }}
spec:
  replicas: {{ .replicas }}
  selector:
    matchLabels:
      {{- include "app.selector" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "app.labels" . | nindent 8 }}
    spec:
      containers:
      - name: {{ .container.name }}
        {{- include "container.image" . | nindent 8 }}
        resources:
          {{- include "resource.limits" . | nindent 10 }}
