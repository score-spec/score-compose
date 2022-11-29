*** Settings ***
Documentation    Shared resources for all test suites.

Library    Process  # http://robotframework.org/robotframework/latest/libraries/Process.html
Library    String  # http://robotframework.org/robotframework/latest/libraries/String.html
Library    OperatingSystem  # http://robotframework.org/robotframework/latest/libraries/OperatingSystem.html


*** Variables ***
${SCORE_COMPOSE_EXEC}    go run ./cli
${RESOURCES_DIR}    e2e-tests/resources/
${OUTPUTS_DIR}    ${RESOURCES_DIR}outputs/


*** Keywords ***
Execute score-compose with ${argument}
    @{cmd}    Split Command Line    ${SCORE_COMPOSE_EXEC} ${argument}
    ${output}    Run Process    @{cmd}
    Log    stdout: ${output.stdout}
    Log    stderr: ${output.stderr}
    Set Test Variable    ${OUTPUT}    ${output}
    Set Test Variable    ${ARGUMENT}    ${argument}

Exit code is ${rc}
    Should be equal as strings    ${OUTPUT.rc}    ${rc}

Vaildate output
    IF    "${RESOURCES_DIR}" in "${ARGUMENT}"
        ${ARGUMENT}    Remove String    ${ARGUMENT}    ${RESOURCES_DIR}
    END
    ${expected_output}    Get File    ${OUTPUTS_DIR}${ARGUMENT}-output.txt
    Should be equal    ${OUTPUT.stdout}    ${expected_output}

Vaildate output to be same as ${other_argument}
    ${expected_output}    Get File    ${OUTPUTS_DIR}${other_argument}-output.txt
    Should be equal    ${OUTPUT.stdout}    ${expected_output}

Vaildate error
    ${expected_output}    Get File    ${OUTPUTS_DIR}${ARGUMENT}-output.txt
    Should Contain    ${OUTPUT.stderr}    ${expected_output}

Execute score-compose run for ${example} example
    Execute score-compose with run -f ${EXAMPLES_DIR}${example}/score.yaml -o ${EXAMPLES_DIR}${example}/compose.yaml
    ${expected_output}    Get File    ${OUTPUTS_DIR}example-${example}-output.txt
    Should be equal    ${OUTPUT.stdout}    ${expected_output}

Execute score-compose run for 03-dependencies example for service-b
    Set Test Variable    ${PATH}    ${EXAMPLES_DIR}03-dependencies/
    Execute score-compose with run -f ${PATH}service-b.yaml -o ${PATH}service-b.compose.yaml
    ${expected_output}    Get File    ${OUTPUTS_DIR}example-03-dependencies-service-b-output.txt
    Should be equal    ${OUTPUT.stdout}    ${expected_output}

Execute score-compose run for 03-dependencies example for service-a
    Set Test Variable    ${PATH}    ${EXAMPLES_DIR}03-dependencies/
    Execute score-compose with run -f ${PATH}service-a.yaml -o ${PATH}service-a.compose.yaml --env-file ${PATH}.env
    ${expected_output}    Get File    ${OUTPUTS_DIR}example-03-dependencies-service-a-output.txt
    Should be equal    ${OUTPUT.stdout}    ${expected_output}

Execute score-compose run for 04-extras example for web-app
    Set Test Variable    ${PATH}    ${EXAMPLES_DIR}04-extras/
    Execute score-compose with run -f ${PATH}score.yaml -o ${PATH}web-app.compose.yaml
    ${expected_output}    Get File    ${OUTPUTS_DIR}example-04-extras-output.txt
    Should be equal    ${OUTPUT.stdout}    ${expected_output}

Verify output of ${example} via docker compose convert
    ${docker_compose_output}    Run
    ...    docker compose -f ${EXAMPLES_DIR}${example}/compose.yaml convert -q
    Should Be Empty    ${docker_compose_output}
    Remove File    ${EXAMPLES_DIR}${example}/compose.yaml

Verify output of 03-dependencies example via docker compose convert
    Set Test Variable    ${PATH}    ${EXAMPLES_DIR}03-dependencies/
    ${docker_compose_output}    Run
    ...    docker compose -f ${PATH}compose.yaml -f ${PATH}service-a.compose.yaml -f ${PATH}service-b.compose.yaml --env-file ${PATH}.env convert -q
    Should Be Empty    ${docker_compose_output}
    Remove Files    ${PATH}service-a.compose.yaml    ${PATH}service-b.compose.yaml    ${PATH}.env

Verify output of 04-extras example via docker compose convert
    Set Test Variable    ${PATH}    ${EXAMPLES_DIR}04-extras/
    ${docker_compose_output}    Run
    ...    docker compose -f ${PATH}compose.yaml -f ${PATH}web-app.compose.yaml convert -q
    Should Be Empty    ${docker_compose_output}
    Remove File    ${PATH}web-app.compose.yaml
