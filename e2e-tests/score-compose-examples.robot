*** Settings ***
Documentation    Verification for examples of score-compose.
Resource    resources/score-compose-shared.resource


*** Variables ***
${EXAMPLES_DIR}    examples/


*** Test Cases ***
Verify score-compose 01-hello example
    Execute score-compose run for 01-hello example
    Verify output of 01-hello via docker compose convert

Verify score-compose 02-environment example
    Execute score-compose run for 02-environment example
    Verify output of 02-environment via docker compose convert

Verify score-compose 03-dependencies example
    Execute score-compose run for 03-dependencies example for service-b
    Execute score-compose run for 03-dependencies example for service-a
    Verify output of 03-dependencies example via docker compose convert

Verify score-compose 04-extras example
    Execute score-compose run for 04-extras example for web-app
    Verify output of 04-extras example via docker compose convert