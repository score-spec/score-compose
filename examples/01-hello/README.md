# 01 - Hello World!

In this basic example there is a simple compose service based on `busybox` Docker image described in a `score.yaml` file:

To convert `score.yaml` file into runnable `compose.yaml` use a `score-compose` CLI tool:

```
$> score-compose run -f ./score.yaml -o ./compose.yaml
```

Output `compose.yaml` file would contain a single service definition:

```
services:
  hello-world:
    command:
      - Hello World!
    entrypoint:
      - /bin/echo
    image: busybox
```

Running this service with `docker-compose`:

```
$> docker-compose -f ./compose.yaml up hello-world

[+] Running 2/2
 ⠿ Network compose_default          Created                                                                                                                                               0.0s
 ⠿ Container compose-hello-world-1  Created                                                                                                                                               0.1s
Attaching to compose-hello-world-1
compose-hello-world-1  | Hello World!
compose-hello-world-1 exited with code 0
```
