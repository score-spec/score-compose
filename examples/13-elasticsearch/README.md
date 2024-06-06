# 13 - Elasticsearch

There is also a `elasticsearch` resource provisioner as an example of a elasticsearch.
This uses the official image from <https://hub.docker.com/_/elasticsearch> with the tag `8.14.0`.

```yaml
resources:
  ecs:
    type: elasticsearch
    # Optionally, you can overwrite the port that will be published, elasticsearch stack version,
    # and elasticsearch memory limit by commenting out the following lines
    # metadata:
    #   annotations:
    #     "compose.score.dev/publish-port": "9201"
    #     "compose.score.dev/stack-version": "8.14.0"
    #     "compose.score.dev/es-mem-limit": "1073741824"
```

The provided outputs are `host`, `port`, `username`, `password`.

Since Elasticsearch version 8.0, the use of certificates is expected.
The certificate is created automatically and can be downloaded from the container.

```sh
$ docker cp [CONTAINER-NAME]:/usr/share/elasticsearch/config/certs/ca/ca.crt /tmp/
Successfully copied 3.07kB to /tmp/
```

Finally, the connection to Elasticsearch can be tested as follows:

```sh
$ curl --cacert /tmp/ca.crt -u [USERNAME]:[PASSWORD] https://localhost:[PORT]
{
  "name" : "elasticsearch",
  "cluster_name" : "cluster-ecs-evKTAl",
  "cluster_uuid" : "29A4gpNsTDGSVAyoJSjN2A",
  "version" : {
    "number" : "8.14.0",
    "build_flavor" : "default",
    "build_type" : "docker",
    "build_hash" : "8d96bbe3bf5fed931f3119733895458eab75dca9",
    "build_date" : "2024-06-03T10:05:49.073003402Z",
    "build_snapshot" : false,
    "lucene_version" : "9.10.0",
    "minimum_wire_compatibility_version" : "7.17.0",
    "minimum_index_compatibility_version" : "7.0.0"
  },
  "tagline" : "You Know, for Search"
}
```
