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

By default, this listens on http://localhost:8080 and in the example will route paths like `/my/fizz/path`, `/my/buzz/path/thing`. The port 8080 unfortunately cannot be changed without modifying the provisioner in the default provisioners file after running `score-compose init`.

By default, this uses a `Prefix` route matching type so `/` can match `/any/request/path` but you can add a `compose.score.dev/route-provisioner-path-type: Exact` annotation to a Route to restrict this behavior to just an exact match.

When running this compose project, we can test these routes using curl (in this instance the generated dns name was `dnsmntq6e`):

```shell
$ curl http://dnsmntq6e.localhost:8080/my/fizz/path/
fizz
$ curl http://dnsmntq6e.localhost:8080/my/buzz/path/
buzz
```

## Adjusting Nginx configuration

The default `route` provisioner generates an nginx config file. If you need to adjust any of the settings or send particular headers, you can use the following snippet annotations.

- `compose.score.dev/route-provisioner-server-snippet` - indents and then adds the lines in the "server" block of nginx.
- `compose.score.dev/route-provisioner-location-snippet` - indents and then adds the lines to every "location" block of nginx.

It is recommended not to include these directly in your Score file and instead add them using the `--overrides-file` or `--override-property` flags when calling `score-compose generate`:

```
$ score-compose generate score.yaml \
    --override-property 'resources.route.metadata.annotations.compose\.score\.dev/route-provisioner-server-snippet="client_max_body_size  20m;"'
```

## Fixed DNS names

If you want a fixed DNS name, you can add a new DNS provisioner to your `.score-compose/0-custom-provisioners.yaml` file:

```yaml
- uri: template://custom-dns
  type: dns
  outputs: |
    host: api.localhost
```
