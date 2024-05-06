# 11 - MySQL database

There is also a `mysql` resource provisioner as an example
of a mysql database. This uses the official image from <https://hub.docker.com/_/mysql>.

```yaml
resources:
  db:
    type: mongodb
```

The provided outputs are `host`, `port`, `name` (aka `database`), `username`, `password`.
The latter is equal to a mysql
connection string like `mysql://${resources.db.username}:${resources.db.password}@${resources.db.host}:${resources.db.port}/${resources.db.name}`.
