{{ range $name, $cfg := .Compose.services }}
    {{ if ne (dig "annotations" "compose.score.dev/workload-name" "" $cfg) "" }}
- op: set
  path: services.{{ $name }}.read_only
  value: true
  description: Set services to read only root fs
    {{ end }}
{{ end }}
