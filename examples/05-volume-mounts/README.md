# 05 - Volume Mounts

Along with `environment`, `volume` is one of the more common resource types supported by score-compose:

```yaml
resources:
  data:
    type: volume
```

This provisions an empty Docker volume with a random name. This can be mounted into the container volumes section:

```yaml
containers:
  example:
    volumes:
      - source: ${resources.data}
        target: /data
```

**NOTE**: the `${resources.data}` placeholder resolves to "data" (the name of the resource) and is linked as the source of the volume.

The volumes are persisted across container starts and stops and can be used for supporting stateful applications.

```console
$ score-compose init
$ score-compose generate score.yaml
...
$ docker compose up
[+] Running 3/3
 ✔ Network 05-volume-mounts_default                Created                                                                                                                                                                                                0.1s
 ✔ Volume "hello-world-data-iIlzJ5"                Created                                                                                                                                                                                                0.0s
 ✔ Container 05-volume-mounts-hello-world-first-1  Created                                                                                                                                                                                                0.1s
Attaching to hello-world-first-1
...
```
