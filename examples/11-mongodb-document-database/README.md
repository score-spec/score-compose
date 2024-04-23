# 11 - MongoDB document database

As an alternative to the `postgres` and `redis` datastores, there is also a `mongodb` resource provisioner as an example
of a document database. This uses the official image from https://hub.docker.com/_/mongo.

```
resources:
  db:
    type: mongodb
```

The provided outputs are `host`, `port`, `username`, `password`, `connection`. The latter is equal to a mongodb 
connection string like `mongodb://${resources.db.username}:${resources.db.password}@${resources.db.host}:${resources.db.port}/}`.
