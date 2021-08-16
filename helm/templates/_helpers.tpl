{{/* vim: set filetype=mustache: */}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "helper.fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/*
Shared labels
*/}}
{{- define "helper.labels" }}
app: "{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}"
chart: "{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}"
release: "{{ .Release.Name }}"
heritage: "{{ .Release.Service }}"
{{- end }}

{{/*
webhook server name
*/}}
{{- define "helper.webhook-server-name" }}
{{- .Values.service.name }}.{{ .Release.Namespace }}.svc
{{- end }}

