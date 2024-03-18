# 07 - Overrides

While `score-compose` and the Score specification in general attempts to provide enough functionality to represent most workloads, you may often find the need to override aspects of the Score spec, the provisioners, or the output `compose.yaml` file.

## Overriding the score file with `--overrides-file`

The `--overrides-file` is another yaml file following the Score specification which can be merged into the original:

```console
$ score-compose generate score.yaml --overrides-file development.score-overrides.yaml
```

For example, you may provide additional development-time variables to a workload:

```yaml
containers:
  web:
    variables:
      DEBUG: "true"
      DISABLE_TLS: "true"
```

## Overriding individual properties in the score file

This can also be done with the `--override-property` option to set a field:

```console
$ score-compose generate score.yaml --override-property 'containers.web.variables.DEBUG="true"'
```

Or remove it:

```console
$ score-compose generate score.yaml --override-property 'resources.thing='
```

## Overriding the compose file

Docker compose will also merge all compose files together. You can use this to specify additional configuration that is not natively supported in `score-compose`.

For example, service ports will not automatically be published on the local network, but an additional compose file can be used here:

```
services:
  web:
    ports:
      - "8080:8080"
```

This can be added in the `up` step:

```console
$ score-compose init
$ score-compose generate score.yaml
$ docker compose -f compose.yaml -f overrides.compose.yaml  up
```
