# 01 - Hello World!

In this basic example there is a simple compose service based on `busybox` Docker image described in a `score.yaml` file:

```yaml
apiVersion: score.sh/v1b1

metadata:
  name: hello-world

containers:
  hello:
    image: busybox
    command: ["/bin/sh"]
    args: ["-c", "while true; do echo Hello World!; sleep 5; done"]
```

To convert `score.yaml` file into runnable `compose.yaml` use a `score-compose` CLI tool:

```console
$ score-compose run -f ./score.yaml -o ./compose.yaml
```

Output `compose.yaml` file would contain a single service definition:

```yaml
services:
  hello-world:
    command:
      - -c
      - 'while true; do echo Hello World!; sleep 5; done'
    entrypoint:
      - /bin/sh
    image: busybox
```

Running this service with `docker-compose`:

```console
$ docker-compose -f ./compose.yaml up hello-world

[+] Running 2/2
 ⠿ Network compose_default          Created                                                                                                                                               0.0s
 ⠿ Container compose-hello-world-1  Created                                                                                                                                               0.1s
Attaching to compose-hello-world-1
compose-hello-world-1  | Hello World!
compose-hello-world-1  | Hello World!
```
