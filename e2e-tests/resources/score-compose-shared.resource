*** Settings ***
Library    OperatingSystem


*** Variables ***
${SCORE_COMPOSE_EXEC}    go run ./cli
${RESOURCES_DIR}    e2e-tests/resources/


*** Keywords ***
Execute score-compose with ${argument}
    ${output}    Run    ${SCORE_COMPOSE_EXEC} ${argument}
    [Return]    ${output}

Execute score-compose with ${argument} and verify output
    ${output}    Run    ${SCORE_COMPOSE_EXEC} ${argument}
    ${expected_output}    Get File    ${RESOURCES_DIR}${argument}-output.txt
    Should be equal    ${output}    ${expected_output}

Execute score-compose with ${argument} and verify output to be same as ${other_argument}
    ${output}    Run    ${SCORE_COMPOSE_EXEC} ${argument}
    ${expected_output}    Get File    ${RESOURCES_DIR}${other_argument}-output.txt
    Should be equal    ${output}    ${expected_output}

Execute score-compose run for ${example} example
    ${output}    Execute score-compose with run -f ${EXAMPLES_DIR}${example}/score.yaml -o ${EXAMPLES_DIR}${example}/compose.yaml
    ${expected_output}    Get File    ${RESOURCES_DIR}example-${example}-output.txt
    Should be equal    ${output}    ${expected_output}

Execute score-compose run for 03-dependencies example for service-b
    Set Test Variable    ${path}    ${EXAMPLES_DIR}03-dependencies/
    ${output}    Execute score-compose with run -f ${path}service-b.yaml -o ${path}service-b.compose.yaml
    ${expected_output}    Get File    ${RESOURCES_DIR}example-03-dependencies-service-b-output.txt
    Should be equal    ${output}    ${expected_output}

Execute score-compose run for 03-dependencies example for service-a
    Set Test Variable    ${path}    ${EXAMPLES_DIR}03-dependencies/
    ${output}    Execute score-compose with run -f ${path}service-a.yaml -o ${path}service-a.compose.yaml --env-file ${path}.env
    ${expected_output}    Get File    ${RESOURCES_DIR}example-03-dependencies-service-a-output.txt
    Should be equal    ${output}    ${expected_output}

Execute score-compose run for 04-extras example for web-app
    Set Test Variable    ${path}    ${EXAMPLES_DIR}04-extras/
    ${output}    Execute score-compose with run -f ${path}score.yaml -o ${path}web-app.compose.yaml
    ${expected_output}    Get File    ${RESOURCES_DIR}example-04-extras-output.txt
    Should be equal    ${output}    ${expected_output}

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