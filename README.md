# score-compose

`score-compose` is an implementation of the Score Workload specification for [Docker compose](https://docs.docker.com/compose/).

## ![Score](docs/images/logo.svg) Score overview

<img src="docs/images/banner.png" width="500px"/>

Score aims to improve developer productivity and experience by reducing the risk of configuration inconsistencies between local and remote environments. It provides developer-centric workload specification (`score.yaml`) which captures a workloads runtime requirements in a platform-agnostic manner. Learn more [here](https://github.com/score-spec/spec#-what-is-score).

The `score.yaml` specification file can be executed against a _Score Implementation CLI_, a conversion tool for application developers to generate environment specific configuration. In combination with environment specific parameters, the CLI tool can run your workload in the target environment by generating a platform-specific configuration file. The `score-compose` CLI is a reference implementation used to generate `docker-compose.yaml` files.

## ![Installation](docs/images/install.svg) Installation

To install `score-compose`, follow the instructions as described in our [installation guide](https://docs.score.dev/docs/get-started/install/).

You will also need a recent version of Docker and the Compose plugin installed. [Read more here](https://docs.docker.com/compose/install/).

## ![Get Started](docs/images/overview.svg) Get Started

If you're getting started, you can use `score-compose init` to create a basic `score.yaml` file in the current directory along with a `.score-compose/` working directory.

```
$ score-compose init --help
The init subcommand will prepare the current directory for working with score-compose and prepare any local
files or configuration needed to be successful.

A directory named .score-compose will be created if it doesn't exist. This file stores local state and generally should
not be checked into source control. Add it to your .gitignore file if you use Git as version control.

The project name will be used as a Docker compose project name when the final compose files are written. This name
acts as a namespace when multiple score files and containers are used.

Usage:
  score-compose init [flags]

Flags:
  -f, --file string      The score file to initialize (default "./score.yaml")
  -h, --help             help for init
  -p, --project string   Set the name of the docker compose project (defaults to the current directory name)

Global Flags:
      --quiet           Mute any logging output
  -v, --verbose count   Increase log verbosity and detail by specifying this flag one or more times
```

Once you have a `score.yaml` file created, modify it by following [this guide](https://docs.score.dev/docs/get-started/score-compose-hello-world/), and use `score-compose run` to convert it into a Docker compose manifest:

```
Translate the SCORE file to docker-compose configuration

Usage:
  score-compose run [--file=score.yaml] [--output=compose.yaml] [flags]

Flags:
      --build string           Replaces 'image' name with compose 'build' instruction
      --env-file string        Location to store sample .env file
  -f, --file string            Source SCORE file (default "./score.yaml")
  -h, --help                   help for run
  -o, --output string          Output file
      --overrides string       Overrides SCORE file (default "./overrides.score.yaml")
  -p, --property stringArray   Overrides selected property value
      --skip-validation        DEPRECATED: Disables Score file schema validation
      --verbose                Enable diagnostic messages (written to STDERR)
```

## ![Get involved](docs/images/get-involved.svg) Get involved

- Give the project a star!
- Contact us via Email:
  - team@score.dev
  - abuse@score.dev
- See our [documentation](https://docs.score.dev)

## ![Contributing](docs/images/contributing.svg) Contributing

- Write a [blog post](https://score.dev/blog)
- Provide feedback on our [roadmap](https://github.com/score-spec/spec/blob/main/roadmap.md#get-involved)
- Contribute

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are greatly appreciated.

If you have a suggestion that would make this better, please fork the repo and create a pull request. You can also open an issue with the tag `enhancement`.

1. Fork the Project.
2. Create your Feature Branch. `git checkout -b feature/feature-name`
3. Commit your Changes. `git commit -s -m "Add some AmazingFeature"`
4. Push to the Branch. `git push origin feature/feature-name`
5. Open a Pull Request.

Read [CONTRIBUTING](CONTRIBUTING.md) for more information.

### Testing

Run the tests using `go test -v ./... -race`. If you do not have `docker` CLI installed locally or want the tests to run
faster, consider setting `NO_DOCKER=true` to skip any `docker compose` based validation during testing.

### Documentation

You can find our documentation at [docs.score.dev](https://docs.score.dev/docs).

### Roadmap

See [Roadmap](https://github.com/score-spec/spec/blob/main/roadmap.md). You can [submit an idea](https://github.com/score-spec/spec/blob/main/roadmap.md#get-involved) anytime.

### License

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

### Code of conduct

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)
