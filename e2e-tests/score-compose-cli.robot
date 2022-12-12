*** Settings ***
Documentation    Verification for all CLI commands of score-compose. The test suite
...    exercises the cli and verifies its output to stored expected outputs.
Resource    resources/score-compose-shared.resource


*** Test Cases ***
Verify score-compose --help
    Execute score-compose with --help
    Exit code is 0
    Vaildate output
    Execute score-compose with -h
    Exit code is 0
    Vaildate output to be same as --help
    Execute score-compose with help
    Exit code is 0
    Vaildate output to be same as --help

Verify score-compose --version
    Execute score-compose with --version
    Exit code is 0
    Vaildate output
    Execute score-compose with -v
    Exit code is 0
    Vaildate output to be same as --version

Verify score-compose completion
    Execute score-compose with completion
    Exit code is 0
    Vaildate output
    Execute score-compose with completion --help
    Exit code is 0
    Vaildate output to be same as completion
    Execute score-compose with completion -h
    Exit code is 0
    Vaildate output to be same as completion

Verify score-compose completion bash
    Execute score-compose with completion bash
    Exit code is 0
    Vaildate output
    Execute score-compose with completion bash --help
    Exit code is 0
    Vaildate output
    Execute score-compose with completion bash -h
    Exit code is 0
    Vaildate output to be same as completion bash --help

Verify score-compose completion fish
    Execute score-compose with completion fish
    Exit code is 0
    Vaildate output
    Execute score-compose with completion fish --help
    Exit code is 0
    Vaildate output
    Execute score-compose with completion fish -h
    Exit code is 0
    Vaildate output to be same as completion fish --help

Verify score-compose completion powershell
    Execute score-compose with completion powershell
    Exit code is 0
    Vaildate output
    Execute score-compose with completion powershell --help
    Exit code is 0
    Vaildate output
    Execute score-compose with completion powershell -h
    Exit code is 0
    Vaildate output to be same as completion powershell --help

Verify score-compose completion zsh
    Execute score-compose with completion zsh
    Exit code is 0
    Vaildate output
    Execute score-compose with completion zsh --help
    Exit code is 0
    Vaildate output
    Execute score-compose with completion zsh -h
    Exit code is 0
    Vaildate output to be same as completion zsh --help

Verify score-compose run
    Execute score-compose with run --help
    Exit code is 0
    Vaildate output
    Execute score-compose with run -h
    Exit code is 0
    Vaildate output to be same as run --help
    Execute score-compose with run -f ${RESOURCES_DIR}example-score.yaml
    Exit code is 0
    Vaildate output
    Execute score-compose with run -f ${RESOURCES_DIR}example-score.yaml --build test
    Exit code is 0
    Vaildate output
    Execute score-compose with run -f ${RESOURCES_DIR}example-score.yaml --overrides ${RESOURCES_DIR}overrides.yaml
    Exit code is 0
    Vaildate output

Verify score-compose run (error cases)
    Execute score-compose with run
    Exit code is 1
    Vaildate error
    Execute score-compose with run --verbose
    Exit code is 1
    Vaildate error

Verify score-compose handles unknown commands
    Execute score-compose with unknown
    Exit code is 1
    Vaildate error
