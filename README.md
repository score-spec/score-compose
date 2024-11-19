<img src="docs/images/banner.png"/>

# score-compose

`score-compose` is a reference implementation of the [Score specification](https://github.com/score-spec/spec) for [Docker compose](https://docs.docker.com/compose/), primarily used for local development. It supports most aspects of the Score specification and provides a powerful resource provisioning system for supplying and customising the dynamic configuration of attached services such as databases, queues, storage, and other network or storage APIs.

See the [`score-compose` documentation](https://docs.score.dev/docs/score-implementation/score-compose/) to get started and see the supported features.

## Feature support

`score-compose` supports as many Score features as possible, however there are certain parts that don't fit well in a local Docker case and are not supported:

| Feature                                                             | Support | Impact                                                                                                                                                                                                                                                                                                                                              |
|---------------------------------------------------------------------|---------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `containers.*.resources.limits` / `containers.*.resources.requests` | none    | **Limits will be validated but ignored.** While the compose specification has some support for this, it is requires particular Docker versions that cannot be relied on. *This should have no impact on Workload execution*.                                                                                                                        |
| `containers.*.livenessProbe` / `containers.*.readinessProbe`        | none    | **Probes will be validated but ignored.** The Score specification only details K8s-like HTTP probes, but the compose specification only supports direct command execution. We cannot convert between the two reliably. *This should have no impact on Workload execution*. Tracked in [#86](https://github.com/score-spec/score-compose/issues/86). |

### Using the `--publish` flag

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

Learn how to connect and engage with our community [here](https://score.dev/community).

### Contribution Guidelines and Governance

Our general contributor guidelines can be found in [CONTRIBUTING.md](CONTRIBUTING.md). Please note that some repositories may have additional guidelines. For more information on our governance model, please refer to [GOVERNANCE.md](https://github.com/score-spec/spec/blob/main/GOVERNANCE.md).

### Documentation

You can find our documentation at [docs.score.dev](https://docs.score.dev/docs).

### Roadmap

See [Roadmap](https://github.com/score-spec/spec/blob/main/roadmap.md). You can [submit an idea](https://github.com/score-spec/spec/blob/main/roadmap.md#get-involved) anytime.

### License

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fscore-spec%2Fscore-compose.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fscore-spec%2Fscore-compose?ref=badge_shield)

### Code of conduct

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)
