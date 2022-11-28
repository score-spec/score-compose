*** Settings ***
Documentation    Verification for all CLI commands of score-compose. The test suite
...    exercises the cli and verifies its output to stored expected outputs.
Resource    resources/score-compose-shared.resource


*** Test Cases ***
Verify score-compose --help
    Execute score-compose with --help and verify output
    Execute score-compose with -h and verify output to be same as --help

Verify score-compose completion
    Execute score-compose with completion and verify output
    Execute score-compose with completion --help and verify output to be same as completion
    Execute score-compose with completion -h and verify output to be same as completion
    Execute score-compose with completion bash and verify output
    Execute score-compose with completion bash --help and verify output
    Execute score-compose with completion bash -h and verify output to be same as completion bash --help

