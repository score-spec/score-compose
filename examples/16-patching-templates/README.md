# 16 - Compose Patching Templates

A common requirement is for users to slightly modify or adjust the output of the conversion process. This can be done
by providing one or more patching templates at `init` time. These patching templates generate JSON patches which are
applied on top of the output compose file, just before being written. Patching templates have access to the current
compose spec as `.Compose`, the map of workload name to Score Spec as `.Workloads`, and can use any functions from [Masterminds/sprig](https://github.com/Masterminds/sprig).

Each patch produced by a template looks like a yaml/json blob with keys `op` (set or delete), `path` 
(the .-separated json path. use backslash to escape), a `value` to set, and an optional `description`.

Example paths:

```
services.some\.thing     # patches the some.thing service
services.foo.ports.0     # modifies the first item in the ports array
services.foo.ports.-1    # adds to the end of the ports array
something.:0000.xyz      # patches the xyz item in the "0000" item of something (: escapes a numeric index)
```

This example shows how you might use these.

In [patch-1.tpl](./patch-1.tpl), we describe a patch which updates all Score workload services to have a read only
root file system. This may more accurate represent the default production security configuration and reduce local
testing drift.

```
{{ range $name, $cfg := .Compose.services }}
    {{ if ne (dig "annotations" "compose.score.dev/workload-name" "" $cfg) "" }}
- op: set
  path: services.{{ $name }}.read_only
  value: true
  description: Set services to read only root fs
    {{ end }}
{{ end }}
```

In [patch-2.tpl](./patch-2.tpl), we slightly modify the output of the `amqp` resource provisioning so that the
management port is exposed on the Rabbitmq container. This means we didn't need to write our own provisioner just to
adjust that value.

```
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
```

And finally, in [patch-3.tpl](./patch-3.tpl), we exposed a debugging port from the score workload.

```
- op: set
  patch: services.hello-world-hello.ports
  value:
  - target: 9999
    published: 9999
```
