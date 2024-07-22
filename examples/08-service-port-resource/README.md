# 08 - Service Port resource example

The `service-port` resource type can be used to link between workloads using the advertised service ports. In score-compose we resolve this to the workload hostname and the target port of the named service port. Errors are thrown if the named workload or port doesn't exist yet.

Other Score implementations may map this to a hostname and port for an appropriate internal load balancer. 

In this example, we have two workloads:

**A - an nginx server that advertises its http port as a service port**

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: workload-a
containers:
  example:
    image: nginx
service:
  ports:
    web:
      port: 8080
      targetPort: 80
```

**and B - a service that wants to call workload A**

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: workload-b
containers:
  example:
    image: busybox
    command: ["/bin/sh"]
    args: ["-c", "while true; do wget $${DEPENDENCY_URL} || true; sleep 5; done"]
    variables:
      DEPENDENCY_URL: "http://${resources.dependency.hostname}:${resources.dependency.port}"
resources:
  dependency:
    type: service-port
    params:
      workload: workload-a
      port: web
```

Here we use a service dependency to identify the service port on `workload-a` and translate that into the target port to be called.

When we run `score-compose init; score-compose generate score*.yaml`, the resulting compose file contains:

```yaml
name: temptest
services:
    workload-a-example:
        annotations:
            compose.score.dev/workload-name: workload-a
        hostname: workload-a
        image: nginx
    workload-b-example:
        annotations:
            compose.score.dev/workload-name: workload-b
        command:
            - -c
            - while true; do wget $${DEPENDENCY_URL} || true; sleep 5; done
        entrypoint:
            - /bin/sh
        environment:
            DEPENDENCY_URL: http://workload-a-example:80
        hostname: workload-b
        image: busybox
```

And when run, the logs show `workload-b` requesting the index page from the nginx server every 5 seconds.

**NOTE**: no dependency relationship is created between the workloads, because Score assumes these workloads may start or restart in any order. Like all good software, the services should be implemented in a way that allows them to start up without the dependency being immediately available. In this case we use `|| true` in the wget statement to ensure `workload-b` retries the request.

## Accessing workload ports

Normally, the ports exposed by a workload, including the `service` ports are not accessible on the docker host. However the `--publish` flag exists to allow this.

The flag can take the form `HOST_PORT:<workload name>:CONTAINER_PORT`, where the container port can be _any_ port on the container you wish to map to the host. This skips any service port mappings.

In the example above, we can publish the `workload-a` port, by calling `generate` with `--publish 8080:workload-a:80`. This modifies the output `compose.yaml` file with an addition ports section:

```yaml
ports:
  - target: 80
    published: "8080"
```
