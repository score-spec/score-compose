*** Settings ***
Library    OperatingSystem

*** Variables ***
${SCORE_COMPOSE_EXEC}    go run ./cli

*** Test Cases ***
Where am I?
    ${output}    Run    pwd
    Log    ${output}

Where is score-compose?
    ${output}    Run    ls -la /home
    Log    ${output}
    ${output}    Run    ls -la /home/runner/work/score-compose/score-compose/dist/
    Log    ${output}

Verify score-compose --help
    ${output}    Run    ${SCORE_COMPOSE_EXEC} --help
    Log    ${output}
    Should contain    ${output}    Complete documentation is available at https://score.dev
    Should contain    ${output}    completion
    Should contain    ${output}    help
    Should contain    ${output}    run

*** Keywords ***
