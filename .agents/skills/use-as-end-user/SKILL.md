---
name: score-compose
description: Help end users run a Score workload locally with score-compose, the Score reference implementation for Docker Compose. Use when someone wants to turn a score.yaml into a compose.yaml, run an app and its dependencies (Postgres, Redis, S3, RabbitMQ, ...) locally with Docker Compose, wire resources into container variables, expose ports, override a Score file, write a custom provisioner, or debug score-compose init/generate errors.
---

# score-compose for end users

`score-compose` converts one or more [Score](https://docs.score.dev) workload files (`score.yaml`) into a single Docker Compose project, and provisions the resources they declare (databases, queues, buckets, ...) as extra Compose services. It is meant for **local development and testing**.

The user writes the *what* (containers, ports, resources) in Score. `score-compose` produces the *how* (`compose.yaml`) and `docker compose` runs it.

## Prerequisites

- Docker with the Compose plugin (`docker compose version`).
- `score-compose` on the PATH (`score-compose --version`). Install options: `brew install score-spec/tap/score-compose`, a binary from the [GitHub releases](https://github.com/score-spec/score-compose/releases), or the [install guide](https://docs.score.dev/docs/score-implementation/score-compose/#installation). The repo's docs assume a recent release.

If the CLI is missing, tell the user how to install it. Don't download or install anything on their behalf without asking.

## The core loop

```bash
score-compose init                 # once per directory: creates .score-compose/ and a sample score.yaml
# edit score.yaml
score-compose generate score.yaml  # writes compose.yaml
docker compose up -d
docker compose logs -f
docker compose down                # add -v to also delete volume data
```

Things worth knowing about this loop:

- `init` must come first. `generate` fails with "state directory does not exist" otherwise.
- `init` flags: `-f/--file` (sample file name), `-p/--project` (Compose project name, defaults to the directory name), `--no-sample` (don't write a sample `score.yaml`, use this when one already exists), `--provisioners <uri>`, `--patch-templates <uri>`, `--no-default-provisioners`. Running `init` again in an initialised directory is safe: an existing `zz-default.provisioners.yaml` is kept.
- `.score-compose/` holds state: workloads added so far, generated resource names and passwords, provisioner files. Keep it out of git (the repo's own `.gitignore` already ignores `.score-compose/` and `*compose.yaml`). Re-running `generate` reuses the stored passwords and names, so the output is stable. Deleting `.score-compose/` resets everything.
- **State accumulates.** `generate a.yaml` then `generate b.yaml` gives a project with both workloads. After editing a Score file, run `generate` on it again to update that workload. `generate` with no arguments regenerates from the stored workloads.
- `-o/--output` changes the output file. `-o -` prints to stdout.

## Writing a score.yaml

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: my-app                       # workload name; also the hostname inside the Compose network
containers:
  web:
    image: nginx:alpine              # or "." plus --image / --build, see below
    variables:
      LOG_LEVEL: debug
      DB_URL: postgres://${resources.db.username}:${resources.db.password}@${resources.db.host}:${resources.db.port}/${resources.db.name}
    files:
      - target: /etc/app/config.txt
        content: "name=${metadata.name}"
    volumes:
      - source: ${resources.data}
        target: /data
service:
  ports:
    http:
      port: 8080
      targetPort: 80
resources:
  db:
    type: postgres
  data:
    type: volume
```

Placeholders and escaping, the most common source of confusion:

- `${resources.<name>.<output>}` and `${metadata.<key>}` are resolved by `score-compose` at generate time, in `variables` and in inline file `content`. Escape a literal dot in a key with a backslash.
- To pass a literal `${FOO}` through to the container or to Compose, write `$${FOO}`. This is required in `command`/`args` strings that use shell variables, for example `args: ["-c", "echo $${GREETING}"]`.
- Files with `source:` are not interpolated. `content:` is, unless `noExpand: true`. `binaryContent:` is base64 and is not interpolated.

Resource identity:

- Each resource entry is independent by default: two workloads that each declare `type: redis` get two Redis services.
- Add `id: shared-cache` to the resource in each workload to **share** one instance across workloads.
- Resource UIDs look like `redis.default#shared-cache` (shared) or `redis.default#my-app.cache` (private to workload `my-app`, resource name `cache`). `class` defaults to `default`.

## Built-in resource types

`score-compose init` installs provisioners for: `environment`, `service-port`, `volume`, `redis`, `postgres`, `mysql`, `mssql`, `mongodb`, `s3` (MinIO), `amqp` (RabbitMQ), `kafka-topic`, `elasticsearch`, `dns`, `route`.

Always confirm what a project has available rather than trusting a list:

```bash
score-compose provisioners list            # type, class, params, outputs, description
score-compose resources list               # provisioned resource UIDs
score-compose resources get-outputs 'postgres.default#my-app.db' --format yaml
```

See [references/resources.md](references/resources.md) for outputs, params and examples per type.

## Common tasks

### Reach a service from the host

Nothing is published to the host by default. Use `--publish` on `generate`:

```bash
# workload container port  ->  HOST_PORT:<workload>:CONTAINER_PORT
score-compose generate score.yaml --publish 8080:my-app:80
# resource service port   ->  HOST_PORT:<resource uid>.<output key>:CONTAINER_PORT
score-compose generate score.yaml --publish '15432:postgres.default#my-app.db.host:5432'
```

`--publish` is not stored in state. Put it on the **last** `generate` call, or a later `generate` will drop it. Quote values containing `#`. Host port must be greater than 1. Alternatives: a `route` resource, or an extra Compose file (below).

### Build your own image

Use `image: .` in the Score file and tell `generate` where the image comes from:

```bash
score-compose generate score.yaml --image my-app:dev                 # use a prebuilt image
score-compose generate score.yaml --build web=./app                  # build container "web" from ./app
score-compose generate score.yaml --build 'web={"context":"./app","dockerfile":"Dockerfile.dev"}'
```

`--image` and `--build` only work when a single Score file is being generated. A container with `image: .` and neither flag is an error.

### Override Score or Compose without touching score.yaml

```bash
score-compose generate score.yaml --overrides-file dev.score.yaml
score-compose generate score.yaml --override-property 'containers.web.variables.DEBUG="true"'
score-compose generate score.yaml --override-property 'resources.thing='   # remove a key
```

Note the flag is `--overrides-file` (plural), even though the help text example spells it `--override-file`. Both override flags require **exactly one** Score file on the command line.

For things Score cannot express (extra ports, profiles, `privileged`), layer another Compose file at run time:

```bash
docker compose -f compose.yaml -f compose.override.yaml up
```

For repeatable post-processing of the generated file, use patch templates, see [references/customising.md](references/customising.md).

### Environment variables from the host

`type: environment` resolves `${resources.env.NAME}` to `${NAME}` in the Compose file, so Compose reads it from the shell or a `.env` file at `docker compose up` time. `score-compose generate --env-file sample.env` writes a skeleton listing the variables the workloads use.

### Multiple workloads, service-to-service calls

Generate them together, or one after another, then use a `service-port` resource to find the other workload:

```yaml
resources:
  api:
    type: service-port
    params: { workload: workload-a, port: web }   # port is the key under service.ports in workload-a
# then ${resources.api.hostname} and ${resources.api.port}
```

Compose does not order workload startup by this link, so the caller should retry until the dependency is up. Resource services (databases, etc.) are waited on via a generated `wait-for-resources` service.

### Custom resource types or different defaults

Drop a `*.provisioners.yaml` file into `.score-compose/`, or install one with `init --provisioners <uri>`. Files load in lexicographic order and the first match on type/class/id wins, so a custom file sorts ahead of `zz-default.provisioners.yaml`. Details and examples: [references/customising.md](references/customising.md).

## What is not supported

- `containers.*.resources.limits/requests`: validated, then ignored.
- `containers.*.livenessProbe/readinessProbe` with `httpGet`: validated, then ignored. `exec` probes are not converted either.
- `files[*].mode`: not supported yet.

These do not change whether the workload runs, but behaviour can differ from a real cluster.

## Troubleshooting

See [references/troubleshooting.md](references/troubleshooting.md) for common errors and fixes. A quick checklist when something looks wrong:

1. `score-compose --version` — is it recent enough for the feature being used?
2. `score-compose generate ... -v` (repeat `-v` for more detail) shows what was provisioned and why.
3. Read the generated `compose.yaml`, and `docker compose config` to see the fully resolved result.
4. `score-compose resources get-outputs <uid>` to see what a placeholder actually resolves to.
5. When a stale state is suspected, `rm -rf .score-compose compose.yaml`, then `init` and `generate` again. This regenerates passwords and resource names, and volume data is only dropped with `docker compose down -v`.

## Working style for the assistant

- Prefer commands the user can copy-paste in order, and show the expected effect (the Compose service names, the file written).
- Read the user's `score.yaml` before suggesting changes. Validate placeholders against `provisioners list` output when the project may have custom provisioners.
- Never put real secrets in `score.yaml`. Resource passwords from default provisioners are random and local-only.
- This tool targets local development. If the user asks about deploying to Kubernetes, point to [score-k8s](https://github.com/score-spec/score-k8s) with the same Score file.
- Docs: <https://docs.score.dev>. Worked examples: the `examples/` directory of this repo (01-hello through 16-patching-templates).
