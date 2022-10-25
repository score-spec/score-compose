# 01 - Environment Variables

When `docker-compose` spins-up the service, it is possible to pass some information from the host to the container via the environment variables.

Compose specification uses a special `environment` resource type to support such cases:

```yaml
apiVersion: score.sh/v1b1

metadata:
  name: hello-world

containers:
  hello:
    image: busybox
    command: ["/bin/sh"]
    args: ["-c", "echo Hello $${FRIEND}!"]
    variables:
      FRIEND: ${resources.env.NAME}

resources:
  env:
    type: environment
    properties:
      NAME:
        type: string
        default: World
```

To convert `score.yaml` file into runnable `compose.yaml` use a `score-compose` CLI tool:

```console
$ score-compose run -f ./score.yaml -o ./compose.yaml
```

Output `compose.yaml` file would contain a single service definition and utilize a host environment variable called `NAME`:

```yaml
services:
  hello-world:
    command:
      - -c
      - echo Hello $${FRIEND}!
    entrypoint:
      - /bin/sh
    environment:
      FRIEND: ${NAME-World}
    image: busybox
```

Running this service with `docker-compose`:

```console
$ NAME=John docker-compose -f ./compose.yaml up hello-world

[+] Running 2/2
 ⠿ Network compose_default          Created                                                                                                                                               0.0s
 ⠿ Container compose-hello-world-1  Created                                                                                                                                               0.1s
Attaching to compose-hello-world-1
compose-hello-world-1  | Hello John!
compose-hello-world-1 exited with code 0
```

## Using `.env` files

For workloads relying on many environment variables it is convenient to manage all required settings in one place, the `.env` file.

`score-compose` CLI tool would produce a template for the `.env` file when `--env-file` parameter is set:

```console
$ score-compose run -f ./score.yaml -o ./compose.yaml --env-file ./.env
```

For the example above the `.env` file would include only one variable:

```yaml
NAME=World
```

Once the `.env` is populated with all the values (usually such file is generated automatically by the configuration managements system or with the CI/CD automation scripts), it can be fed to `docker-compose`:

```console
$ NAME=John docker-compose -f ./compose.yaml --env-file ./.env up hello-world
```
