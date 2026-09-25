# 03 - Files

Score workloads can bind-mount files into running containers. Files using `content` support `${..}` placeholder interpolation just like the container variables section. Files using a `source` path do not.

This is invaluable when configuring containers whose images you don't control or cannot modify further.

In the Score file below, two files are mounted, one with inline `content`, the other sourced from the local directory relative to the input Score file. Binary file content, or files that contain non-utf-8 characters can be provided via the a base64 encoded `binaryContent` source property but does understandably doesn't support placeholder replacements.s

```yaml
apiVersion: score.dev/v1b1

metadata:
  name: hello-world

containers:
  hello:
    image: busybox
    command: ["/bin/sh"]
    args: ["-c", "while true; do cat /fileA.txt; cat /fileB.txt; cat /fileC.bin; sleep 5; done"]
    files:
      - target: /fileA.txt
        source: fileA.txt
      - target: /fileB.txt
        content: |
          I am ${metadata.name}
      - target: /fileC.bin
        binaryContent: aGVsbG8gd29ybGQ=
```

As usual, we run this with `init` and `generate score.yaml`. After generate, you may notice that the files have been written to disk at `.score-compose/mounts/files`.

```console
$ ls .score-compose/mounts/files/
hello-world-files-0-fileA.txt  hello-world-files-1-fileB.txt
```

And when we run it:

```console
$ docker compose up
[+] Running 2/2
 ✔ Network 03-files_default                Created                                         0.1s
 ✔ Container 03-files-hello-world-hello-1  Cre...                                          0.1s
Attaching to hello-world-hello-1
hello-world-hello-1  | This is fileA.
hello-world-hello-1  | I am hello-world
hello-world-hello-1  | hello worldThis is fileA.
hello-world-hello-1  | I am hello-world
hello-world-hello-1  | hello worldThis is fileA.
```

`files[*].noExpand` is supported to disable placeholder interpolation in inline content. `files[*].mode` is not yet supported, see [#88](https://github.com/score-spec/score-compose/issues/88).

## Custom placeholder delimiters (experimental)

Score's default `${...}` placeholder syntax can clash with config formats that use the same notation themselves, such as Spring Boot properties, shell scripts, or Docker Compose env vars. When you need literal `${...}` to survive in the rendered file, you can ask Score to look for a different delimiter pair by setting two annotations on the workload metadata:

- `compose.score.dev/experiment-placeholder-start`
- `compose.score.dev/experiment-placeholder-end`

Both must be set together. When set, file content is expanded using those delimiters instead of `${` and `}`; any literal `${...}` in the file is left alone. `noExpand: true` on a file still takes priority, and `binaryContent` is never expanded.

```yaml
apiVersion: score.dev/v1b1

metadata:
  name: spring-app
  annotations:
    compose.score.dev/experiment-placeholder-start: "<%{"
    compose.score.dev/experiment-placeholder-end: "}%>"

containers:
  app:
    image: my-spring-app
    files:
      - target: /app/application.properties
        content: |
          # left alone (Spring Boot will resolve these at runtime)
          spring.datasource.url=jdbc:postgresql://${DB_HOST}:${DB_PORT}/${DB_NAME}

          # expanded by Score
          app.name=<%{metadata.name}%>
```

There is no escape syntax for the custom delimiters in this experimental version. If your file legitimately contains the chosen start/end strings, pick a different pair.

This is a `compose.score.dev/experiment-*` annotation, meaning it is opt-in and may change before becoming a stable part of the spec. See [score-spec/spec#108](https://github.com/score-spec/spec/issues/108) for background.
