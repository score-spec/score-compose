*** Settings ***
Library    Process  # http://robotframework.org/robotframework/latest/libraries/Process.html
Library    String  # http://robotframework.org/robotframework/latest/libraries/String.html
Library    OperatingSystem  # http://robotframework.org/robotframework/latest/libraries/OperatingSystem.html

*** Variables ***
${SCORE_COMPOSE_EXEC}    go run ./cli
${RESOURCES_DIR}    e2e-tests/resources/


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
    ${expected_output}    Get File    ${RESOURCES_DIR}${ARGUMENT}-output.txt
    Should be equal    ${OUTPUT.stdout}    ${expected_output}

Vaildate output to be same as ${other_argument}
    ${expected_output}    Get File    ${RESOURCES_DIR}${other_argument}-output.txt
    Should be equal    ${OUTPUT.stdout}    ${expected_output}

Vaildate error
    ${expected_output}    Get File    ${RESOURCES_DIR}${ARGUMENT}-output.txt
    Should Contain    ${OUTPUT.stderr}    ${expected_output}

Execute score-compose run for ${example} example
    Execute score-compose with run -f ${EXAMPLES_DIR}${example}/score.yaml -o ${EXAMPLES_DIR}${example}/compose.yaml
    ${expected_output}    Get File    ${RESOURCES_DIR}example-${example}-output.txt
    Should be equal    ${OUTPUT.stdout}    ${expected_output}

Execute score-compose run for 03-dependencies example for service-b
    Set Test Variable    ${path}    ${EXAMPLES_DIR}03-dependencies/
    Execute score-compose with run -f ${path}service-b.yaml -o ${path}service-b.compose.yaml
    ${expected_output}    Get File    ${RESOURCES_DIR}example-03-dependencies-service-b-output.txt
    Should be equal    ${OUTPUT.stdout}    ${expected_output}

Execute score-compose run for 03-dependencies example for service-a
    Set Test Variable    ${path}    ${EXAMPLES_DIR}03-dependencies/
    Execute score-compose with run -f ${path}service-a.yaml -o ${path}service-a.compose.yaml --env-file ${path}.env
    ${expected_output}    Get File    ${RESOURCES_DIR}example-03-dependencies-service-a-output.txt
    Should be equal    ${OUTPUT.stdout}    ${expected_output}

Execute score-compose run for 04-extras example for web-app
    Set Test Variable    ${path}    ${EXAMPLES_DIR}04-extras/
    Execute score-compose with run -f ${path}score.yaml -o ${path}web-app.compose.yaml
    ${expected_output}    Get File    ${RESOURCES_DIR}example-04-extras-output.txt
    Should be equal    ${OUTPUT.stdout}    ${expected_output}

Verify output of ${example} via docker compose convert
    ${docker-compose-output}    Run
    ...    docker compose -f ${EXAMPLES_DIR}${example}/compose.yaml convert -q
    Should Be Empty    ${docker-compose-output}
    Remove File    ${EXAMPLES_DIR}${example}/compose.yaml

Verify output of 03-dependencies example via docker compose convert
    Set Test Variable    ${path}    ${EXAMPLES_DIR}03-dependencies/
    ${docker-compose-output}    Run
    ...    docker compose -f ${path}compose.yaml -f ${path}service-a.compose.yaml -f ${path}service-b.compose.yaml --env-file ${path}.env convert -q
    Should Be Empty    ${docker-compose-output}
    Remove Files    ${path}service-a.compose.yaml    ${path}service-b.compose.yaml    ${path}.env

Verify output of 04-extras example via docker compose convert
    Set Test Variable    ${path}    ${EXAMPLES_DIR}04-extras/
    ${docker-compose-output}    Run
    ...    docker compose -f ${path}compose.yaml -f ${path}web-app.compose.yaml convert -q
    Should Be Empty    ${docker-compose-output}
    Remove File    ${path}web-app.compose.yaml