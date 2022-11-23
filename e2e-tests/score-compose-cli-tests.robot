*** Settings ***
Library    OperatingSystem

*** Variables ***
${SCORE_COMPOSE_EXEC}    go run ./cli
${RESOURCES}    e2e-tests/resources/

*** Test Cases ***
Verify score-compose --help
    Execute score-compose with --help and verfiy output
    Execute score-compose with -h and verfiy output to be same as --help

Verify score-compose completion
    Execute score-compose with completion and verfiy output
    Execute score-compose with completion --help and verfiy output to be same as completion
    Execute score-compose with completion -h and verfiy output to be same as completion
    Execute score-compose with completion bash and verfiy output
    Execute score-compose with completion bash --help and verfiy output
    Execute score-compose with completion bash -h and verfiy output to be same as completion bash --help

*** Keywords ***
Execute score-compose with ${argument} and verfiy output
    ${output}    Run    ${SCORE_COMPOSE_EXEC} ${argument}
    ${expected_output}    Get File    ${RESOURCES}${argument}-output.txt
    Should be equal    ${output}    ${expected_output}

Execute score-compose with ${argument} and verfiy output to be same as ${other_argument}
    ${output}    Run    ${SCORE_COMPOSE_EXEC} ${argument}
    ${expected_output}    Get File    ${RESOURCES}${other_argument}-output.txt
    Should be equal    ${output}    ${expected_output}