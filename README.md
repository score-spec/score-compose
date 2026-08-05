[![ci](https://github.com/score-spec/score-compose/actions/workflows/ci.yaml/badge.svg)](https://github.com/score-spec/score-compose/actions/workflows/ci.yaml)
[![release](https://github.com/score-spec/score-compose/actions/workflows/release.yaml/badge.svg)](https://github.com/score-spec/score-compose/actions/workflows/release.yaml)
[![good first issues](https://img.shields.io/github/issues-search/score-spec/score-compose?query=type%3Aissue%20is%3Aopen%20label%3A%22good%20first%20issue%22&label=good%20first%20issues&style=flat&logo=github)](https://github.com/score-spec/score-compose/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fscore-spec%2Fscore-compose.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fscore-spec%2Fscore-compose?ref=badge_shield)
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/score-spec/score-compose/badge)](https://scorecard.dev/viewer/?uri=github.com/score-spec/score-compose)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/9900/badge)](https://www.bestpractices.dev/projects/9900)

<img src="docs/images/banner.png"/>

# score-compose

`score-compose` is a reference implementation of the [Score specification](https://github.com/score-spec/spec) for [Docker compose](https://docs.docker.com/compose/), primarily used for local development. It supports most aspects of the Score specification and provides a powerful resource provisioning system for supplying and customising the dynamic configuration of attached services such as databases, queues, storage, and other network or storage APIs.

## Feature support

`score-compose` supports as many Score features as possible, however there are certain parts that don't fit well in a local Docker case and are not supported:

| Feature                                                                      | Support | Impact                                                                                                                                                                                                                                                                                                                                              |
|------------------------------------------------------------------------------|---------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `containers.*.resources.limits` / `containers.*.resources.requests`          | none    | **Limits will be validated but ignored.** While the compose specification has some support for this, it is requires particular Docker versions that cannot be relied on. *This should have no impact on Workload execution*.                                                                                                                        |
| `containers.*.livenessProbe.httpGet` / `containers.*.readinessProbe.httpGet` | none    | **Probes will be validated but ignored.** The Score specification details both command execution and HTTP probes, but the compose specification only supports direct command execution. We cannot convert between the two reliably, so the `exec` mode is ignored *This should have no impact on Workload execution*. Tracked in [#86](https://github.com/score-spec/score-compose/issues/86). |

## Resource support

`score-compose` comes with out-of-the-box support for:

| Type          | Class   | Params                 | Output                                                                                                                                                          |
| ------------- | ------- | ---------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| environment   | default | (none)                 | `${KEY}`                                                                                                                                                        |
| service-port  | default | `workload`, `port`     | `hostname`, `port`                                                                                                                                              |
| volume        | default | (none)                 | `source`                                                                                                                                                        |
| redis         | default | (none)                 | `host`, `port`, `username`, `password`                                                                                                                          |
| postgres      | default | (none)                 | `host`, `port`, `name` (aka `database`), `username`, `password`                                                                                                 |
| mysql         | default | (none)                 | `host`, `port`, `name` (aka `database`), `username`, `password`                                                                                                 |
| s3            | default | (none)                 | `endpoint`, `access_key_id`, `secret_key`, `bucket`, with `region=""`, `aws_access_key_id=<access_key_id>`, and `aws_secret_key=<secret_key>` for compatibility |
| dns           | default | (none)                 | `host`                                                                                                                                                          |
| route         | default | `host`, `path`, `port` |                                                                                                                                                                 |
| amqp          | default | (none)                 | `host`, `port`, `vhost`, `username`, `password`                                                                                                                 |
| mongodb       | default | (none)                 | `host`, `port`, `username`, `password`, `connection`                                                                                                            |
| kafka-topic   | default | (none)                 | `host`, `port`, `name`, `num_partitions`                                                                                                                        |
| elasticsearch | default | (none)                 | `host`, `port`, `username`, `password`                                                                                                                          |
| mssql         | default | (none)                 | `server`, `port`, `connection`, `database`, `username`, `password`                                                                                              |

These can be found in the default provisioners file. You are encouraged to write your own provisioners and add them to the `.score-compose` directory (with the `.provisioners.yaml` extension) or contribute them upstream to the [default.provisioners.yaml](internal/command/default.provisioners.yaml) file.

## Installation

To install `score-compose`, follow the instructions as described in our [installation guide](https://docs.score.dev/docs/score-implementation/score-compose/#installation). You will also need a recent version of Docker and the Compose plugin installed. Read more [here](https://docs.docker.com/compose/install/). Alternatively, the generated files can be run with Podman, see [Podman support](#podman-support).

## Podman support

`score-compose` never talks to a container engine itself: `score-compose generate` only writes a [Compose specification](https://compose-spec.io/) compliant `compose.yaml`. This means the generated file can also be run with [Podman](https://podman.io/) instead of Docker, as long as the compose implementation used supports the constructs described below. There are three common ways to do this:

1. **`podman compose`** - a thin wrapper that runs an external compose provider against the Podman socket. When both providers are installed it prefers `docker-compose`, the compose specification reference implementation, which supports everything `score-compose` generates (recommended):

   ```console
   $ systemctl --user enable --now podman.socket   # enable the rootless Podman API socket
   $ score-compose init
   $ score-compose generate score.yaml
   $ podman compose up -d
   ```

2. **`docker compose` (or `docker-compose`) pointed at the Podman socket** - the same engine as above, without the wrapper:

   ```console
   $ export DOCKER_HOST=unix://$XDG_RUNTIME_DIR/podman/podman.sock
   $ docker compose up -d
   ```

3. **`podman-compose`** - the native Python implementation. Basic Workloads work, but check the notes on `depends_on` below before relying on it for provisioned database resources.

### Compose features the generated file relies on

`score-compose` emits some newer compose specification constructs, which set a minimum version for the compose implementation used to run the output:

| Construct                                                                                                       | Used for                                                                          | Podman notes                                                                                                                                                                                                              |
|-----------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `annotations` on services                                                                                       | every Workload service                                                            | requires `docker-compose` >= v2.20; older compose implementations may reject the file (see [#442](https://github.com/score-spec/score-compose/pull/442) for legacy compose support)                                        |
| long-form `depends_on` with `condition: service_healthy` / `service_completed_successfully` and `required: true` | the `wait-for-resources` service that delays Workloads until resources are ready | `docker-compose` >= v2.20; `podman-compose` >= 1.3.0 with Podman >= 4.6. Older `podman-compose` releases ignore the conditions **silently**, so Workloads can start before their databases are healthy                     |
| `network_mode: "service:..."`                                                                                   | multi-container Workloads sharing one network namespace                           | works through the Docker-compatible socket; support in `podman-compose` varies by version                                                                                                                                  |
| `healthcheck` on resource services                                                                              | database readiness checks                                                         | needs Podman >= 4.6 for health-condition waits                                                                                                                                                                              |
| relative bind mounts under `.score-compose/mounts/`                                                             | Workload files and provisioner-generated configuration                            | see the SELinux and rootless notes below                                                                                                                                                                                    |
| named volumes with `driver: local`                                                                              | database and resource storage                                                     | fully supported                                                                                                                                                                                                             |

### Known differences and gotchas

- **Rootless networking and service DNS**: `score-compose` injects compose service names as hostnames into resource outputs (for example `mysql://user:password@mysql-xxxxxx:3306/db`), so container-to-container DNS must work on the compose project network. Rootless Podman provides this via `aardvark-dns` on the `netavark` network backend (the default since Podman 4.x); `slirp4netns`/`pasta` only provide outbound connectivity and play no role in service-to-service DNS. If Workloads cannot resolve each other, check that `podman info --format '{{.Host.NetworkBackend}}'` reports `netavark` and that the `aardvark-dns` package is installed. Legacy CNI-based setups need the `dnsname` plugin instead.
- **Rootless UID remapping on bind mounts**: rootless Podman maps container users through the ranges in `/etc/subuid` and `/etc/subgid`. Files under `.score-compose/mounts/` are bind mounted into resource containers, so anything a container user writes into such a mount shows up on the host owned by a high subordinate UID (`ls -ln .score-compose/mounts` shows the numeric owners). Use `podman unshare` to inspect or `chown` those files from inside the user namespace. Database data is not affected since it is stored in named volumes.
- **SELinux labels (Fedora/RHEL/CentOS and other enforcing hosts)**: bind mounts need `:z`/`:Z` relabeling when SELinux is enforcing. `score-compose` does not emit these labels, and `docker-compose` pointed at the Podman socket will not add them either (`podman-compose` sometimes does). The symptom is resource containers failing with `permission denied` while reading files under `.score-compose/mounts/` - it looks like a `score-compose` bug but is an SELinux labeling gap. The clean fix is a [patching template](examples/16-patching-templates) that labels all generated bind mounts:

  ```
  {{ range $name, $svc := .Compose.services }}
  {{ range $i, $vol := $svc.volumes }}
  {{ if eq (dig "type" "" $vol) "bind" }}
  - op: set
    path: services.{{ $name }}.volumes.{{ $i }}.bind.selinux
    value: z
    description: label bind mount for SELinux ({{ $name }})
  {{ end }}
  {{ end }}
  {{ end }}
  ```

  Install it once with `score-compose init --patch-templates ./selinux-bind-mounts.tpl` and every subsequent `generate` adds `bind.selinux: z` to each bind mount. Alternatively, disabling label separation per service with `security_opt: ["label=disable"]` also works but is coarser. Findings from SELinux-enforcing hosts are being collected in [#129](https://github.com/score-spec/score-compose/issues/129) - reports welcome.
- **macOS and Windows**: Podman runs containers inside a Linux VM (`podman machine`, with port forwarding handled by `gvproxy`). Published ports are forwarded from the host through the VM, and bind mounts cross a file share, so the `.score-compose/` directory must live under a path that is shared with the machine (on macOS the home directory is shared by default) and file ownership and mount performance behave differently than on native Linux.
- **Short image names**: Score files that use short image names such as `image: nginx` rely on Podman's short-name resolution rules (`registries.conf`) when run with `podman-compose`, which may prompt or fail depending on the host configuration; through the Docker-compatible socket they default to `docker.io`. Prefer fully qualified references such as `docker.io/library/nginx:latest` in Score files. The built-in provisioners already use fully qualified images.
- **Privileged ports**: rootless Podman cannot publish host ports below 1024 by default, so `--publish 80:workload:8080` or a `route` resource published on port 80 fails with a permission error. Use a port >= 1024 or raise the limit with `sysctl net.ipv4.ip_unprivileged_port_start`.
- **`restart: always`**: resource services use restart policies, but without a daemon these only apply while Podman (or the Podman machine) is running. Containers are not restarted at boot unless you manage them with systemd units or [Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html).

The [examples](./examples) in this repository can be used as a test bed for all of the above; testing notes and findings are tracked in [#129](https://github.com/score-spec/score-compose/issues/129).

## Get started

**NOTE**: the following examples and guides relate to `score-compose >= 0.11.0`, check your version using `score-compose --version` and re-install if you're behind!

See the [examples](./examples) for more examples of using Score and provisioning resources:

- [01-hello](examples/01-hello) - a basic example of a Score Workload
- [02-environment](examples/02-environment) - an example of environment variables and the `type: environment` resource
- [03-files](examples/03-files) - mounting local files into the running Workload
- [04-multiple-workloads](examples/04-multiple-workloads) - examples of multiple containers and workloads together
- [05-volume-mounts](examples/05-volume-mounts) - an example of an "empty-dir" volume resource with `type: volume`
- [06-resource-provisioning](examples/06-resource-provisioning) - detailed example and information about resource provisioning and the operation of the `template://` and `cmd://` provisioners
- [07-overrides](examples/07-overrides) - details of how to override aspects of the input Score file and output Docker compose files
- [08-service-port-resource](examples/08-service-port-resource) - an example of using the `service-port` resource type to link between workloads
- [09-dns-and-route](examples/09-dns-and-route) - an example of using the `dns` and `route` resources to route http requests
- [10-amqp-rabbitmq](examples/10-amqp-rabbitmq) - an example the default `amqp` resource provisioner
- [11-mongodb-document-database](examples/11-mongodb-document-database) - an example the default `mongodb` resource provisioner
- [12-mysql-database](examples/12-mysql-database) - an example of the default `mysql` resource provisioner
- [13-kafka-topic](examples/13-kafka-topic) - an example of the default `kafka-topic` resource provisioner
- [14-elasticsearch](examples/14-elasticsearch) - an example of the default `elasticsearch` resource provisioner
- [15-mssql-database](examples/15-mssql-database) - an example of the default `mssql` resource provisioner
- [16-patching-templates](examples/16-patching-templates) - a guide to patch templates

If you're getting started, you can use `score-compose init` to create a basic `score.yaml` file in the current directory along with a `.score-compose/` working directory.

```
$ score-compose init --help
The init subcommand will prepare the current directory for working with score-compose and prepare any local
files or configuration needed to be successful.

A directory named .score-compose will be created if it doesn't exist. This file stores local state and generally should
not be checked into source control. Add it to your .gitignore file if you use Git as version control.

The project name will be used as a Docker compose project name when the final compose files are written. This name
acts as a namespace when multiple score files and containers are used.

Custom provisioners can be installed by uri using the --provisioners flag. The provisioners will be installed and take
precedence in the order they are defined over the default provisioners. If init has already been called with provisioners
the new provisioners will take precedence.

To adjust the way the compose project is generated, or perform post processing actions, you can use the --patch-templates
flag to provide one or more template files by uri. Each template file is stored in the project and then evaluated as a 
Golang text/template and should output a yaml/json encoded array of patches. Each patch is an object with required 'op' 
(set or delete), 'patch' (a dot-separated json path), a 'value' if the 'op' == 'set', and an optional 'description' for 
showing in the logs.

Usage:
  score-compose init [flags]

Examples:

  # Define a score file to generate
  score-compose init --file score2.yaml

  # Or override the docker compose project name
  score-compose init --project score-compose2

  # Or disable the default score file generation if you already have a score file
  score-compose init --no-sample

  # Optionally loading in provisoners from a remote url
  score-compose init --provisioners https://raw.githubusercontent.com/user/repo/main/example.yaml

  # Optionally adding a couple of patching templates
  score-compose init --patch-templates ./patching.tpl --patch-templates https://raw.githubusercontent.com/user/repo/main/example.tpl

URI Retrieval:
  The --provisioners and --patch-templates arguments support URI retrieval for pulling the contents from a URI on disk
  or over the network. These support:
    - Files       : file:///local/file or ./local/file
    - HTTP        : http://host/file
    - HTTPS       : https://host/file
    - Git (SSH)   : git-ssh://git@host/repo.git/file
    - Git (HTTPS) : git-https://host/repo.git/file
    - OCI         : oci://[registry/][namespace/]repository[:tag|@digest][#file]

Flags:
  -f, --file string                  The score file to initialize (default "./score.yaml")
  -h, --help                         help for init
      --no-sample                    Disable generation of the sample score file
      --patch-templates stringArray   Patching template files to include. May be specified multiple times. Supports URI retrieval.
  -p, --project string               Set the name of the docker compose project (defaults to the current directory name)
      --provisioners stringArray     Provisioner files to install. May be specified multiple times. Supports URI retrieval.

Global Flags:
      --quiet           Mute any logging output
  -v, --verbose count   Increase log verbosity and detail by specifying this flag one or more times
```

Once you have a `score.yaml` file created, modify it by following [this guide](https://docs.score.dev/docs/get-started/score-compose-hello-world/), and use `score-compose generate` to convert it into a Docker compose manifest:

```
The generate command will convert Score files in the current Score compose project into a combined Docker compose
manifest. All resources and links between Workloads will be resolved and provisioned as required.

By default this command looks for score.yaml in the current directory, but can take explicit file names as positional
arguments.

"score-compose init" MUST be run first. An error will be thrown if the project directory is not present.

Usage:
  score-compose generate [flags]

Examples:

  # Specify Score files
  score-compose generate score.yaml *.score.yaml

  # Regenerate without adding new score files
  score-compose generate

  # Provide overrides when one score file is provided
  score-compose generate score.yaml --override-file=./overrides.score.yaml --override-property=metadata.key=value

  # Publish a port exposed by a workload for local testing
  score-compose generate score.yaml --publish 8080:my-workload:80

  # Publish a port from a resource host and port for local testing, the middle expression is RESOURCE_ID.OUTPUT_KEY
  score-compose generate score.yaml --publish 5432:postgres#my-workload.db.host:5432

Flags:
      --build stringArray               An optional build context to use for the given container --build=container=./dir or --build=container={"context":"./dir"}
      --env-file string                 Location to store a skeleton .env file for compose - this will override existing content
  -h, --help                            help for generate
  -i, --image string                    An optional container image to use for any container with image == '.'
  -o, --output string                   The output file to write the composed compose file to (default "compose.yaml")
      --override-property stringArray   An optional set of path=key overrides to set or remove
      --overrides-file string           An optional file of Score overrides to merge in
  -p, --publish stringArray             An optional set of HOST_PORT:<ref>:CONTAINER_PORT to publish on the host system.

Global Flags:
      --quiet           Mute any logging output
  -v, --verbose count   Increase log verbosity and detail by specifying this flag one or more times
```

### Using the `--publish` (`-p`) flag

`score-compose` installs all workloads and resource services into the compose docker network but does not publish ports on the host by default. To access ports inside the network, the user must either exec into a target container, run a new `socat` container with published ports, use a `route` resource that publishes "public" ports, or modify the compose.yaml directly.

The `--publish` flag on the `generate` command can be used to automatically add published ports to the services in the compose.yaml file in a safe and dynamic way. Note that this is an _ephemeral_ flag not stored in the state file. As such it should be added to the _last_ invocation of `score-compose generate` to avoid it being overridden by subsequent calls.

- To publish a container port from a workload to the host, use `--publish HOST_PORT:<workload>:CONTAINER_PORT`.
- To publish a a port from one of the resource services, use `--publish HOST_PORT:<resource uid>.<output key>:CONTAINER_PORT`. Examples of this are:
  - `15432:postgres#my-workload.res.host:5432` - `host` is the output on the `postgres.default#my-workload.res` Postgres resource that contains the service hostname.
  - `9001:s3.default#storage.service:9001` - `service` is the service name output on the `s3.default#storage` S3 resource.

This deprecates the use of the `compose.score.dev/publish-port` resource metadata annotation in most cases. 

## Testing

Run the tests using `go test -v ./... -race`. If you do not have `docker` CLI installed locally or want the tests to run
faster, consider setting `NO_DOCKER=true` to skip any `docker compose` based validation during testing.

## Get in touch

Learn how to connect and engage with our community [here](https://github.com/score-spec/spec?tab=readme-ov-file#-get-in-touch).

### Contribution Guidelines and Governance

Our general contributor guidelines can be found in [CONTRIBUTING.md](CONTRIBUTING.md). Please note that some repositories may have additional guidelines. For more information on our governance model, please refer to [GOVERNANCE.md](https://github.com/score-spec/spec/blob/main/GOVERNANCE.md).

### Documentation

You can find our documentation at [docs.score.dev](https://docs.score.dev/docs).

### Roadmap

See [Roadmap](https://github.com/score-spec/spec/blob/main/roadmap.md). You can [submit an idea](https://github.com/score-spec/spec/blob/main/roadmap.md#get-involved) anytime.
