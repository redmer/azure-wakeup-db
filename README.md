# azure-wakeup-db

If your Azure (MSSQL) DB automatically pauses, `ghcr.io/redmer/azure-wakeup-db` re-awakens them.
That may be useful for workflows or ETL's that would otherwise quickly fail.

The error you might see is error `40613` or something like:

> `mssql: login error: Database '***' on server '***.database.windows.net' is not currently available.  Please retry the connection later.  If the problem persists, contact customer support, and provide them the session tracing ID of '***'.`

## Usage

```
docker run --rm ghcr.io/redmer/azure-wakeup-db --dsn <dsn>
```

Example output:

```
$ docker run --rm ghcr.io/redmer/azure-wake-up-db --server=hello-world.database.windows.net --database=general --user=kenobi --password='Ben123'
2025/04/07 17:12:20 attempt 1/15
2025/04/07 17:12:36 attempt 2/15 after 25s delay
2025/04/07 17:13:13 attempt 3/15 after 25s delay
2025/04/07 17:13:40 Connection successful: database is awake.
```

Options:

- `--dsn`: The connection string to the Azure DB. Can be specified in many formats:
  - `sqlserver://username:password@host/instance`
  - `server=localhost;user id=sa;database=master;app name=MyAppName`
  - `odbc:server=localhost;user id=sa;password={foo;bar}`
- Or use the following specific options. They will **not** be combined with the DSN.

  - `--server`: Database host
  - `--port`: Database port (default: 1433)
  - `--instance`: SQL Server instance name (optional)
  - `--database`: Database name
  - `--user`: Database username
  - `--password`: Database password

  All variants of the connection string described at [microsoft/go-mssqldb].
  Kerberos or EntraID is not supported.

- Environment variables:
  - `WAKEUP_DSN`: Full DSN in any of the above syntaxes.
  - Or use the following specific options. They will **not** be combined with the DSN.
    - `WAKEUP_SERVER`: Database host
    - `WAKEUP_DATABASE`: Database name
    - `WAKEUP_PORT`: Database port (default: 1433)
    - `WAKEUP_INSTANCE`: SQL Server instance name (optional)
    - `WAKEUP_USER`: Database username
    - `WAKEUP_PASSWORD`: Database password

[microsoft/go-mssqldb]: https://github.com/microsoft/go-mssqldb/blob/main/README.md#the-connection-string-can-be-specified-in-one-of-three-formats

## FAQ

- **PR's are welcome**
- **Why in Go?**
  - Because I wanted to try and write a small application in Go.
- **What would be a good PR?**
  - Other ways to authenticate
  - Combine DSN and named options.
  - The actual delays aren't 25 seconds, but they're printed as such.
