{{- define "shopkart.image" -}}
{{- printf "%s/%s:%s" .Values.global.imageRegistry .name .Values.global.imageTag -}}
{{- end -}}
