![Score banner](docs/images/banner.png)

# ![Score](docs/images/logo.svg) Score overview

_Score_ provides a developer-centric and platform-agnostic workload specification to improve developer productivity and experience. Score eliminates configuration inconsistencies between local and remote environments.

The _Platform CLI_ is a conversion tool for application developers to generate an environment specific configuration. In combination with environment specific parameters, The Platform CLI tool can run your Workload in the target environment by generating the target platform's configuration file.

## ![Installation](docs/images/install.svg) Installation

### Step 1. Download

1. Open <https://github.com/score-spec/score-compose/releases> in your browser.

2. Find the current release by scrolling down and looking for the green tag that reads _Latest_.

3. Download the latest compressed Tar file for your operating system.
   1. The name will be something like `score-compose_x.y_osx-amd64.tar.gz`, where `x.y` is the release number, `osx` is the operating system.
   2. By default, the tarball will be saved to your `~/Downloads` directory. If you choose to use a different location, you'll need to change that in the following steps.

**Results** You should see something similar to the following output.

```bash
Saving to: `score-compose_x.y_osx-amd64.tar.gz`

score-compose_x.y.0 100%[===================>]   2.85M  5.28MB/s    in 0.5s
```

#### Step 2: Install into your `local` directory

In your terminal, enter the following to create the `score-spec` directory.

```bash
cd /usr/local/
# create the directory if needed
sudo mkdir -pv score-spec
```

Extract the compressed Tar file.

```bash
sudo tar -xvzf ~/Downloads/score-compose_0.1.0_darwin_arm64.tar.gz
```

**Results** You should see the following output.

```bash
x LICENSE
x README.md
x score-compose
```

### Step 3: Export PATH

To export `PATH`, run the following command.

```bash
export PATH=$PATH:/usr/local/score-spec
```

### Step 4: Verify installation

To verify installation, run the following command.

```bash
score-compose --version
```

The previous command returns the following output.

```bash
score-compose x.y.z
```

**Results** You've successfully installed the Platform CLI.

## ![Overview](docs/images/overview.svg) Overview

The Score specification file resolves configuration inconsistencies between environments. Compose a `score.yaml` file that describes how to run your Workload. As a platform-agnostic declaration file, `score.yaml` creates a single source of truth on Workload profiles and works to integrate with any platform or tooling.

### Use the Platform CLI tool

```bash
# Generate compose.yaml with score-compose
score-compose run -f /tmp/score.yaml -o /tmp/compose.yaml

# Run the service with docker-compose
docker-compose -f /tmp/compose.yaml up backend
```

## ![Manifesto](docs/images/manifesto.svg) Score manifesto

- Enable local development without risk of configuration inconsistencies in remote environments.
- Offer default configuration while allowing for a large degree of customization.
- Establish a single source of truth for application configuration.
- Separate environment specific from environment agnostic configuration.
- Enable environment agnostic declaration of infrastructure dependencies.
- Enable application centric rather than infrastructure centric development.
- Abstract away infrastructural complexity without sacrificing transparency.

For more information, see the [Score manifesto](https://score.dev/manifesto).

## ![Get involved](docs/images/get-involved.svg) Get involved

- Give the project a star!
- Contact us via Email:
  - team@score.dev
  - abuse@score.dev
- See our [documentation](https://docs.score.dev).

## ![Contributing](docs/images/contributing.svg) Contributing

- Write a [blog](score.dev/blog).
- Provide feedback on our [road map and releases board](https://github.com/orgs/score-spec/projects).
- Contribute.

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are greatly appreciated.

If you have a suggestion that would make this better, please fork the repo and create a pull request. You can also open an issue with the tag `enhancement`.

1. Fork the Project.
2. Create your Feature Branch. `git checkout -b feature/feature-name`
3. Commit your Changes. `git commit -s -m "Add some AmazingFeature"`
4. Push to the Branch. `git push origin feature/feature-name`
5. Open a Pull Request.

Read [CONTRIBUTING](CONTRIBUTING.md) for more information.

## ![License](docs/images/license.svg) License

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

## ![Code of conduct](docs/images/code-of-conduct.svg) Code of conduct

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](code_of_conduct.md)
