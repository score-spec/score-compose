# 13 - Kafka Topic

The `kafka-topic` provisioner ensures that there is a Kafka topic with 3 partitions available in Docker.

See [score.yaml](score.yaml) as an example. This app launches a Redpanda Console configured for the given Kafka broker. A Score dns and route resource are used to expose it locally outside of Docker.

This provisioner also supports an annotation for exposing the broker port for access from outside Docker:

```yaml
resources:
  bus:
    type: kafka-topic
    metadata:
      annotations:
        "compose.score.dev/publish-port": "9092"
```
