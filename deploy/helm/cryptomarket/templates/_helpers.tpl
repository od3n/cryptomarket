{{/*
Expand the name of the chart.
*/}}
{{- define "cryptomarket.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "cryptomarket.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "cryptomarket.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "cryptomarket.labels" -}}
helm.sh/chart: {{ include "cryptomarket.chart" . }}
app.kubernetes.io/part-of: cryptomarket
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
{{- end }}

{{/*
Selector labels for a component
*/}}
{{- define "cryptomarket.selectorLabels" -}}
app.kubernetes.io/name: {{ .name }}
app.kubernetes.io/instance: {{ .root.Release.Name }}
{{- end }}

{{/*
Component labels
*/}}
{{- define "cryptomarket.componentLabels" -}}
{{ include "cryptomarket.labels" .root }}
app.kubernetes.io/name: {{ .name }}
app.kubernetes.io/component: {{ .component }}
app.kubernetes.io/instance: {{ .root.Release.Name }}
{{- end }}

{{/*
Image tag - defaults to appVersion
*/}}
{{- define "cryptomarket.imageTag" -}}
{{- .tag | default .root.Chart.AppVersion }}
{{- end }}

{{/*
Full image reference
*/}}
{{- define "cryptomarket.image" -}}
{{- $registry := .root.Values.global.imageRegistry | default "" }}
{{- if $registry }}
{{- printf "%s/%s:%s" $registry .image.repository (include "cryptomarket.imageTag" .) }}
{{- else }}
{{- printf "%s:%s" .image.repository (include "cryptomarket.imageTag" .) }}
{{- end }}
{{- end }}

{{/*
Namespace
*/}}
{{- define "cryptomarket.namespace" -}}
{{- .Values.global.namespace | default .Release.Namespace }}
{{- end }}
