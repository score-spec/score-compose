# 05 - Volume Mounts

Many Score projects require volume storage. Volume storage may be a tmpfs, a host bind mount, or a persistent and stateful volume. All three can be represented in score-compose depending on the provisioner.

A default provisioner exists for a generic `volume` type which provisions a Docker volume and mounts it to the container.

As an example, the Score file may contain:

```yaml
containers:
  example:
    volumes:
      - source: ${resources.data}
        target: /data
resources:
  data:
    type: volume
```

The `${resources.data}` placeholder resolves to "volume.default#<workload>.data" (the uid of the resource) which is converted into outputs, and then into a Docker compose volume.

The outputs of the volume resource look like:

```yaml
type: volume
source: <generated docker volume name>
```

Since this provisions a stateful Docker volume, the state is persisted across restarts and may also be shared between containers.

## Writing a tmpfs provisioner

If the container does not need state to persist across restarts, we can use `tmpfs` volume. This isn't available as a default provisioner but can be easily added to a `.score-compose/0-custom-provisioners.yaml` file:

```yaml
- uri: template://custom-empty-dir-volume
  type: emptyDir
  outputs: |
    type: tmpfs
    tmpfs:
      size: 10000000
```

This kind of volume cannot be shared across containers.

## Writing a bind provisioner

In some cases you may need to bind mount a volume or file from the host. This could be written as a custom provisioner too!

```yaml
- uri: template://custom-bind-mount
  type: host-logs
  outputs: |
    type: bind
    source: /var/log
    bind:
      propagation: rprivate
```

Note that this would usually have a custom type since the typed-provisioner defines what source directory from the host is mounted.
