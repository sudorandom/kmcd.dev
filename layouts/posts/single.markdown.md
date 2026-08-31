---
title: {{ .Title | jsonify }}
date: {{ .Date.Format "2006-01-02T15:04:05Z07:00" }}
url: {{ .Permalink }}
{{- with .Params.tags }}
tags: {{ . | jsonify }}
{{- end }}
{{- with .Params.categories }}
categories: {{ . | jsonify }}
{{- end }}
{{- with .Description }}
description: {{ . | jsonify }}
{{- end }}
---

{{ .RawContent }}
