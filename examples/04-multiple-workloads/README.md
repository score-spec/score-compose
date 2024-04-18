# 04 - Multiple Workloads and Containers

Score supports multiple containers in the same Workload. In `score-compose`, these containers are placed within the same network namespace in a manner similar to Kubernetes. Listening ports may not overlap.

```yaml
apiVersion: score.dev/v1b1

metadata:
  name: hello-world

containers:
  first:
    image: "nginx:latest"
    variables:
      NGINX_PORT: "8080"
  second:
    image: "nginx:latest"
    variables:
      NGINX_PORT: "8081"
```

```console
$ score-compose init
$ score-compose generate score.yaml
```

Score compose also supports multiple workloads in the same project directory. These can be added one at a time to the project since `score-compose generate` is _additive_ but must have independent workload names. Containers from different workloads run in different network namespaces.

A second `score2.yaml` file can be written:

```yaml
apiVersion: score.dev/v1b1

metadata:
  name: hello-world-2

containers:
  first:
    image: "nginx:latest"
    variables:
      NGINX_PORT: "8080"
```

```console
$ score-compose generate score2.yaml
```

The output compose file appears as below. Notice that all 3 containers are present and that the `hello-world-2` doesn't share its network with the other containers.

```yaml
name: 04-multiple-workloads
services:
    hello-world-2-first:
        environment:
            NGINX_PORT: "8080"
        image: nginx:latest
    hello-world-first:
        environment:
            NGINX_PORT: "8080"
        image: nginx:latest
    hello-world-second:
        environment:
            NGINX_PORT: "8081"
        image: nginx:latest
        network_mode: service:hello-world-first
```

View the `score-compose generate --help` text for more information about overriding properties and Score workload contents. If no special options are required, the generate command can add multiple Score files at once:

```console
score-compose generate score*.yaml
```
