# 04 - Ports and Volumes

In advanced setups the workload might need to use volumes or serve incoming requests on selected ports.

Such requirements can be expressed in `score.yaml` file:

```yaml
apiVersion: score.dev/v1b1

metadata:
  name: web-app

service:
  ports:
    www:
      port: 8000
      targetPort: 80

containers:
  hello:
    image: nginx
    volumes:
      - source: data
        target: /usr/share/nginx/html
        readOnly: true
```

To convert `score.yaml` file into runnable `web-app.compose.yaml` use a `score-compose` CLI tool:

```console
$ score-compose run -f ./score.yaml -o ./web-app.compose.yaml
```

Output `web-app.compose.yaml` file would contain a single service definition:

```yaml
services:
  web-app:
    image: nginx
    ports:
      - target: 80
        published: "8000"
    volumes:
      - type: volume
        source: data
        target: /usr/share/nginx/html
```

This compose service configuration references a volume called `data`, that should be available in the target environment by the time the service starts.

If running the service with the Docker, the volume can be created manually ([read more](https://docs.docker.com/storage/volumes/#create-and-manage-volumes)).

For example, to share the host file system contents, the volume can use a `local` driver as shown in the root `compose.yaml` file:

```yaml
volumes:
  data:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: ${PWD}/data
```

Running `web-app` service with `docker-compose`:

```console
$ docker-compose -f ./compose.yaml -f ./web-app.compose.yaml up

[+] Running 2/0
 ⠿ Volume "compose_data"        Created                                                                                                                                           0.0s
 ⠿ Container compose-web-app-1  Created                                                                                                                                           0.1s
Attaching to compose-web-app-1
```

And a quick test with `curl`:

```console
$ curl http://localhost:8000
Hello World!
```