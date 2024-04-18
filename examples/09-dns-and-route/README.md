# 09 - dns and route resources

In previous examples, we showed how Score workloads can export service ports for other workloads to communicate with. However, this only exposes ports internally over some internal network and doesn't guarantee that these ports can be exposed over a public or external network.

To do this, we have a `dns` resource to allocate an external dns name, and `route` to to specify an HTTP path to route to the specified service port. Notably this only supports tcp services that are hosting an HTTP protocol and won't work for other more complicated routing.

Our score file now includes these 2 resources:

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: hello-world
service:
  ports:
    web:
      port: 8080
      targetPort: 80
containers:
  web:
    image: nginx
    files:
      - target: /usr/share/nginx/html/my/fizz/path/index.html
        content: fizz
      - target: /usr/share/nginx/html/my/buzz/path/index.html
        content: buzz
resources:
  dns:
    type: dns
  route:
    type: route
    params:
      path: /my/[^/]+/path
      host: ${resources.dns.host}
      port: web
```

The `route` resource indicates the host from the dns resource, and the path to route. While the `web` port is the one exposed by the service.

This adds an additional nginx service to the compose file which contains an HTTP routing specification for the hostname and path combinations.

By default, this listens on http://localhost:8080 and in the example will route paths like `/my/fizz/path`, `/my/buzz/path/thing`

By default, this uses a `Prefix` route matching type so `/` can match `/any/request/path` but you can add a `score-compose.score.dev/route-provisioner-path-type: Exact` annotation to a Route to restrict this behavior to just an exact match.

When running this compose project, we can test these routes using curl (in this instance the generated dns name was `dnsmntq6e`):

```shell
$ curl http://dnsmntq6e.09-dns-and-route.localhost:8080/my/fizz/path/
fizz
$ curl http://dnsmntq6e.09-dns-and-route.localhost:8080/my/buzz/path/
buzz
```
