# 09 - dns and route resources

In previous examples, we showed how Score workloads can export service ports for other workloads to communicate with. However, this only exposes ports internally over some internal network and doesn't guarantee that these ports can be exposed over a public or external network.

To do this, we have a `dns` resource to allocate an external dns name, and `route` to to specify an HTTP path to route to the specified service port. Notably this only supports tcp services that are hosting an HTTP protocol and won't work for other more complicated routing.

Our score file now includes these 2 resources:

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: hello-world
service:
  web:
    port: 8080
    targetPort: 80
containers:
  web:
    image: nginx
resources:
  dns:
    type: dns
  route:
    type: route
    params:
      path: /subpath
      host: ${resources.dns.host}
      port: web
```

The `route` resource indicates the host from the dns resource, and the sub path to route. While the `web` port is the one exposed by the service.

This adds an additional nginx service to the compose file which contains an HTTP routing specification for the hostname and path combinations.

By default, this listens on http://localhost:8080.

## Limitations

Technically, the `dns` resource should produce random or appropriate hostnames for each dns resource. However, this implementation always generates localhost since we don't by default create real DNS names or modify the local /etc/hosts file.

This means that if there are multiple workloads with overlapping path routing, they will fail or only one will resolve correctly.

To get around this, you may inject your own dns resource provisioner with appropriate behavior for your environment, the only requirement is that it returns a `host` output string that is resolvable.
