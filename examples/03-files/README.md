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
