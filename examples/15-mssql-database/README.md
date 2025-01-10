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
        # "compose.score.dev/mssql-pid": "Developer" refer to https://mcr.microsoft.com/artifact/mar/mssql/server/about
```

The provided outputs are `host`, `port`, `connection`, `database`, `username`, `password`, `pid`.
The latter is equal to a mssql
connection string like `Server=${resources.db.host}; Database=${resources.db.database}; User=${resources.db.username}; Password=${resources.db.password};`.
