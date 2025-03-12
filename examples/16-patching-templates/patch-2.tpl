{{ range $name, $cfg := .Compose.services }}
    {{ if and (eq $cfg.image "rabbitmq:3-management-alpine") (eq $cfg.restart "always") }}
- op: set
  path: services.{{ $name }}.ports
  value:
  - target: 15672
    published: "15672"
  description: Expose the management port of the rabbitmq resource service
    {{ end }}
{{ end }}
