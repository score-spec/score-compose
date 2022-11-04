# 03 - Dependencies

Score uses `resources` section to describe workload's dependencies. This mechanism can be used to spin-up multi-service setups with `docker-compose`.

For example, `service-a.yaml` score file describes a service that has two dependencies: `service-b` (another workload) and a PostgreSQL database instance:

```yaml
apiVersion: score.dev/v1b1

metadata:
  name: service-a

containers:
  service-a:
    image: busybox
    command: ["/bin/sh"]
    args: ["-c", "while true; do echo service-a: Hello $${FRIEND}! Connecting to $${CONNECTION_STRING}...; sleep 10; done"]
    variables:
      FRIEND: ${resources.env.NAME}
      CONNECTION_STRING: postgresql://${resources.db.user}:${resources.db.password}@${resources.db.host}:${resources.db.port}/${resources.db.name}

resources:
  env:
    type: environment
    properties:
      NAME:
        type: string
        default: World
  db:
    type: postgres
    properties:
      host:
        default: localhost
      port:
        default: 5432
      name:
        default: postgres
      user:
        secret: true
      password:
        secret: true
  service-b:
    type: workload
```

The second workload is described in `service-b.yaml` file and has no any additional dependencies:

```yaml
apiVersion: score.dev/v1b1

metadata:
  name: service-b

containers:
  service-b:
    image: busybox
    command: ["/bin/sh"]
    args: ["-c", "while true; do echo service-b: Hello $${FRIEND}!; sleep 5; done"]
    variables:
      FRIEND: ${resources.env.NAME}

resources:
  env:
    type: environment
    properties:
      NAME:
        type: string
        default: World
```

To prepare executable `docker-compose` configuration files, convert both score files with `score-compose` CLI tool:

```console
$ score-compose run -f ./service-b.yaml -o ./service-b.compose.yaml
$ score-compose run -f ./service-a.yaml -o ./service-a.compose.yaml --env-file ./.env
```

Resulting output file  `service-a.compose.yaml` would include two dependencies on compose services `db` and `service-b`.
Both should be up and running before `service-a` could start:

```yaml
services:
  service-a:
    command:
      - -c
      - 'while true; do echo service-a: Hello $${FRIEND}! Connecting to $${CONNECTION_STRING}...; sleep 10; done'
    depends_on:
      db:
        condition: service_started
      service-b:
        condition: service_started
    entrypoint:
      - /bin/sh
    environment:
      CONNECTION_STRING: postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST-localhost}:${DB_PORT-5432}/${DB_NAME-postgres}
      NAME: ${NAME-World}
    image: busybox
```

One last step is to ensure there is a compose `db` service definition.
Common place to store non-score defined configuration and resources is a root `compose.yaml` file:

```yaml
services:
  db:
    image: postgres:alpine
    restart: always
    environment:
      - POSTGRES_USER=${DB_USER}
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    ports:
      - '5432:5432'
    volumes: 
      - db:/var/lib/postgresql/data

volumes:
  db:
    driver: local
```

Ensure the `.env` file has all the proper configuration values in palace:

```console
NAME=World
DB_HOST=localhost
DB_PORT=5432
DB_NAME=postgres
DB_USER=postgres
DB_PASSWORD=postgres
```

Now all files can be combined together to spin-up the whole application with `docker-compose`:

```console
$ docker-compose -f ./compose.yaml -f ./service-a.compose.yaml -f ./service-b.compose.yaml --env-file ./.env up

[+] Running 4/4
 ⠿ Network compose_default        Created                                                                                                                          0.0s
 ⠿ Container compose-db-1         Created                                                                                                                          0.1s
 ⠿ Container compose-service-b-1  Recreated                                                                                                                        0.1s
 ⠿ Container compose-service-a-1  Recreated                                                                                                                        0.1s
Attaching to compose-db-1, compose-service-a-1, compose-service-b-1
compose-service-b-1  | service-b: Hello World!
compose-db-1         | 
compose-db-1         | PostgreSQL Database directory appears to contain a database; Skipping initialization
compose-db-1         | 
compose-db-1         | 2022-10-25 04:53:58.528 UTC [1] LOG:  starting PostgreSQL 15.0 on x86_64-pc-linux-musl, compiled by gcc (Alpine 11.2.1_git20220219) 11.2.1 20220219, 64-bit
compose-db-1         | 2022-10-25 04:53:58.528 UTC [1] LOG:  listening on IPv4 address "0.0.0.0", port 5432
compose-db-1         | 2022-10-25 04:53:58.528 UTC [1] LOG:  listening on IPv6 address "::", port 5432
compose-db-1         | 2022-10-25 04:53:58.540 UTC [1] LOG:  listening on Unix socket "/var/run/postgresql/.s.PGSQL.5432"
compose-db-1         | 2022-10-25 04:53:58.551 UTC [23] LOG:  database system was shut down at 2022-10-25 04:52:28 UTC
compose-db-1         | 2022-10-25 04:53:58.562 UTC [1] LOG:  database system is ready to accept connections
compose-service-a-1  | service-a: Hello World! Connecting to postgresql://postgres:postgres@localhost:5432/postgres...
compose-service-b-1  | service-b: Hello World!
compose-service-b-1  | service-b: Hello World!
compose-service-a-1  | service-a: Hello World! Connecting to postgresql://postgres:postgres@localhost:5432/postgres...
compose-service-b-1  | service-b: Hello World!
```
