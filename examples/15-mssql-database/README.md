# 12 - MSSQL Server database

There is also a `mssql` resource provisioner as an example
of a mssql database. This uses the official image from <https://hub.docker.com/r/microsoft/mssql-serverl>.

```yaml
resources:
  db:
    type: mssql
    # Optionally, you can overwrite the port that will be published
    # by commenting out the following lines
    # metadata:
      # annotations:
        # "compose.score.dev/publish-port": "1434"
```

The provided outputs are `server`, `port`, `connection`, `database`, `username`, `password`.
The latter is equal to a mssql
connection string like `Server=${resources.db.server}; Database=${resources.db.database}; User=${resources.db.username}; Password=${resources.db.password};`.
