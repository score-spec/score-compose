# score-compose end-to-end test suite

This end-to-end test suite ensures continuous functionality of the score-compose CLI as well as the functionality of the included examples. 

## Contribution

Contribution to this test suite requires basic understanding of the [RobotFramework](www.robotframework.org). As well as the used libraries.

- [Process Library](https://robotframework.org/robotframework/latest/libraries/Process.html)
- [OperatingSystem Library](http://robotframework.org/robotframework/latest/libraries/OperatingSystem.html)
- [String Library](http://robotframework.org/robotframework/latest/libraries/String.html)

Which are extending the list of Robotframework's [build-in keywords](http://robotframework.org/robotframework/latest/libraries/BuiltIn.html). 
For more information, see the [RobotFramework user guide](http://robotframework.org/robotframework/latest/RobotFrameworkUserGuide.html).

## Structure and scope of tests

The end to end test suite consists of two main parts.
1. `score-compose-cli.robot` covering all cli commands and ensuring their outputs
2. `score-compose-examples.robot` covering all examples from the examples folder in this repo.

All shared resources are located in the `resources` folder.
`resources/score-compose-shared.resource` contains all custom keywords, variables and libraries.
Expected outputs can be found under `resources/outputs`. These outputs can be re-generated via a utility script [utility/generate-expected-outputs.py](utility/generate-expected-outputs.py)

## Running the tests locally

Robotframework is a Python based generic testing framework. It is recommended to use a [Python virtual environment](https://docs.python.org/3/library/venv.html) to have a clean local setup.

### Environment setup

1. Clone the repository [score-compose](https://github.com/score-spec/score-compose) from Github.
2. Install Python 3.10 on your machine and then install `pip`.
3. Start a new virtual environment on the root folder of this project, using Python 3.10, and activate it.
   ```bash
   pip3 install virtualenv
   virtualenv venv
   source venv/bin/activate
   ```
4. First tool to install would be the [`pip-tools`](https://github.com/jazzband/pip-tools).
   A set of command line tools to help you keep your pip-based packages fresh, even when you've pinned them.

   This way we can have the essential packages for this project pinned in the `requirements.in` and with that you can construct the `requirements.txt` file that should be used when installing packages. When you want to update some packages, all you need to do is update the `requirements.in` file and the re-compile the `requirements.txt` one.
   
   Run `python -m pip install pip-tools` inside your activated pip environment.
   When you need to compile a new version of the requirements.txt, do `pip-compile --output-file=- > requirements.txt`
   inside your terminal.
5. Install the pip libraries required:
   ```bash
    pip install -r e2e-tests/requirements.txt
   ```

### Running the tests

The tests require the score-compose CLI to be executed by `go run ./cli`. Therefore [go](https://go.dev/) needs to be installed.

The tests can be run via `robot e2e-tests`.
