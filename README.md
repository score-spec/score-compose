![Score banner](docs/images/banner.png)

# ![Score](docs/images/logo.svg) Score overview

Score aims to improve developer producticity and experience by reducing the risk of configurtaion inconsistencies between local and remote environments. It provides developer-centric workload specification (`score.yaml`) which captures a workloads runtime requirements in a platform-agnostic manner.

The `score.yaml` specification file can be executed against a _Score Implementation CLI_, a conversion tool for application developers to generate environment specific configuration. In combination with environment specific parameters, the CLI tool can run your workload in the target environment by generating a platform-specific configuration file such as `docker-compose.yaml` or a Helm `values.yaml`. Learn more [here](https://github.com/score-spec/spec#-what-is-score).

## ![Installation](docs/images/install.svg) Installation

To install `score-compose`, follow the instructions as described in our [installation guide](https://docs.score.dev/docs/get-started/install/).

## ![Get Started](docs/images/overview.svg) Get Started

If you already have a `score.yaml` file defined, you can simply run the following command:

```bash
# Prepare a compose.yaml file
score-compose run -f /tmp/score.yaml -o /tmp/compose.yaml
```

- `run` tells the CLI to translate the Score file to a Docker Compose file.
- `-f` is the path to the Score file.
- `--env` specifies the path to the output file.

If you're just getting started, follow [this guide](https://docs.score.dev/docs/get-started/score-compose-hello-world/) to run your first Hello World program with `score-compose`.

## ![Get involved](docs/images/get-involved.svg) Get involved

- Give the project a star!
- Contact us via Email:
  - team@score.dev
  - abuse@score.dev
- See our [documentation](https://docs.score.dev).

## ![Contributing](docs/images/contributing.svg) Contributing

- Write a [blog post](score.dev/blog).
- Provide feedback on our [road map and releases board](https://github.com/score-spec/spec/blob/main/roadmap.md#get-involved).
- Contribute.

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are greatly appreciated.

If you have a suggestion that would make this better, please fork the repo and create a pull request. You can also open an issue with the tag `enhancement`.

1. Fork the Project.
2. Create your Feature Branch. `git checkout -b feature/feature-name`
3. Commit your Changes. `git commit -s -m "Add some AmazingFeature"`
4. Push to the Branch. `git push origin feature/feature-name`
5. Open a Pull Request.

Read [CONTRIBUTING](CONTRIBUTING.md) for more information.

### Documentation

You can find our documentation at [docs.score.dev](https://docs.score.dev/docs).

### Roadmap

See [Roadmap](https://github.com/score-spec/spec/blob/main/roadmap.md). You can [submit an idea](https://github.com/score-spec/spec/blob/main/roadmap.md#get-involved) anytime.

### License

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

### Code of conduct

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](code_of_conduct.md)