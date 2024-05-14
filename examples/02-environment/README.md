# 02 - Environment Variables

When `docker-compose` spins-up the service, it is possible to pass some information from the host to the container via the environment variables. These variables can then be accessed by programs running in the container.

The `containers.*.variables` section supports placeholder interpolation using `${..}` syntax. This can be used to access outputs from metadata or resources. The placeholder syntax can be escaped with a double-$ like `$${..}`.

The `environment` resource to collect them from the current shell:

```yaml
apiVersion: score.dev/v1b1

metadata:
  name: hello-world

containers:
  hello:
    image: busybox
    command: ["/bin/sh"]
    args: ["-c", "while true; do echo $${GREETING} $${NAME}!; sleep 5; done"]
    variables:
      GREETING: Hello
      NAME: ${resources.env.NAME}
      WORKLOAD_NAME: ${metadata.name}

resources:
  env:
    type: environment
```

Like [example 01](../01-hello), we use `generate` to build a compose file:

```console
$ score-compose init
$ score-compose generate score.yaml
```

And it returns

```yaml
name: 02-environment
services:
    hello-world-hello:
        annotations:
            compose.score.dev/workload-name: hello-world
        command:
            - -c
            - while true; do echo $${GREETING} $${NAME}!; sleep 5; done
        entrypoint:
            - /bin/sh
        environment:
            GREETING: Hello
            NAME: ${NAME}
            WORKLOAD_NAME: hello-world
        hostname: hello-world
        image: busybox
```

Now we can set this variable using a `.env` file (see below) or provide it when we run `docker compose`:

```console
$ NAME=John docker-compose -f ./compose.yaml up hello-world
[+] Running 2/2
 ⠿ Network compose_default          Created
 ⠿ Container compose-hello-world-1  Created
Attaching to compose-hello-world-1
compose-hello-world-1  | Hello John!
compose-hello-world-1  | Hello John!
```

## Using `.env` files

For workloads relying on many environment variables it is convenient to manage all required settings in one place, the `.env` file.

`docker compose` will load `.env` from the current directory or any other file passed as `--env-file`:

```console
$ cat .env
NAME=John
```

```console
$ NAME=John docker-compose -f ./compose.yaml up hello-world
[+] Running 2/2
 ⠿ Network compose_default          Created
 ⠿ Container compose-hello-world-1  Created
Attaching to compose-hello-world-1
compose-hello-world-1  | Hello Bob!
compose-hello-world-1  | Hello Bob!
```

`score-compose` can generate the initial env file for you if you're not sure what variables are used. To do this, specify the `--env-file` flag when running the `generate` subcommand.

```
$ score-compose generate --env-file sample.env
$ cat sample.env
NAME=
```
