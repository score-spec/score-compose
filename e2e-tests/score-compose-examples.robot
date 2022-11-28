*** Settings ***
Documentation    Verification for examples of score-compose.
Resource    resources/score-compose-shared.resource


*** Variables ***
${EXAMPLES_DIR}    examples/


*** Test Cases ***
Verify score-compose example 01-hello
    ${output}    Execute score-compose with run -f ${EXAMPLES_DIR}01-hello/score.yaml -o ${EXAMPLES_DIR}01-hello/compose.yaml
    ${expected_output}    Get File    ${RESOURCES_DIR}example-01-hello-output.txt
    Should be equal    ${output}    ${expected_output}
    ${docker-compose-output}    Run    docker compose -f ${EXAMPLES_DIR}01-hello/compose.yaml convert -q
    Should Be Empty    ${docker-compose-output}
    Remove File    ${EXAMPLES_DIR}01-hello/compose.yaml


