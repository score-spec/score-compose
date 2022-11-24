*** Settings ***
Library    OperatingSystem

*** Variables ***
${SCORE_COMPOSE_EXEC}    go run ./cli
${RESOURCES}    e2e-tests/resources/

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

*** Keywords ***
Execute score-compose with ${argument} and verify output
    ${output}    Run    ${SCORE_COMPOSE_EXEC} ${argument}
    ${expected_output}    Get File    ${RESOURCES}${argument}-output.txt
    Should be equal    ${output}    ${expected_output}

Execute score-compose with ${argument} and verify output to be same as ${other_argument}
    ${output}    Run    ${SCORE_COMPOSE_EXEC} ${argument}
    ${expected_output}    Get File    ${RESOURCES}${other_argument}-output.txt
    Should be equal    ${output}    ${expected_output}