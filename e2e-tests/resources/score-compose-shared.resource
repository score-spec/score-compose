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

