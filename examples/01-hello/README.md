# 01 - Hello World!

In this basic example there is a simple compose service based on `busybox` Docker image described in a `score.yaml` file:

```yaml
apiVersion: score.dev/v1b1

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
$ score-compose init
$ score-compose generate score.yaml
```

The `init` will create the `.score-compose` directory. The `generate` command will add the input `score.yaml` workload to the `.score-compose/state.yaml` state file and regenerate the output `compose.yaml`.

```yaml
name: 01-hello
services:
    hello-world-hello:
        command:
            - -c
            - while true; do echo Hello World!; sleep 5; done
        entrypoint:
            - /bin/sh
        hostname: hello-world
        image: busybox
```

This `compose.yaml` can then be run directly, and you can watch the output of the logs.

```console
$ docker compose up -d
[+] Running 1/2
 ⠼ Network 01-hello_default                Created
 ✔ Container 01-hello-hello-world-hello-1  Started
$ docker logs -f 01-hello-hello-world-hello-1
Hello World!
Hello World!
Hello World!
Hello World!
^C%
$ docker compose down
```

If you make modifications to the `score.yaml` file, run `score-compose generate score.yaml` to regenerate the output.
