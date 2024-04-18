# 10 - AMQP resource provisioning

Many applications may require a connection to an AMQP protocol message broker. score-compose comes with a built-in
provider for RabbitMQ using the public docker image.

When this provider is used, it will generate a single RabbitMQ instance with a v-host per AMQP resource definition.

```
resources:
  bus:
    type: amqp
```

The provided outputs are `host`, `port`, `vhost`, `username`, and `password` and these can be assembled together into a 
URI like `amqp://${resources.bus.username}:${resources.bus.password}@${resources.bus.host}:${resources.bus.port}/${resources.bus.vhost}`.

The generated user has administrator permissions on the vhost.

For more complicated configuration, it is recommended to copy the default rabbitmq provisioner to your own provisioners
file and modify the configuration.

## Accessing the AMQP and management ports

By tagging your `amqp` resources with the publish-port annotations you can publish the ports on `localhost`:

```yaml
resources:
  bus:
    type: amqp
    metadata:
      annotations:
        "compose.score.dev/publish-port": "5672"
        "compose.score.dev/publish-management-port": "15672"
```

The default user with username `guest`, password `guest` will be available. This is very useful for debugging applications.
