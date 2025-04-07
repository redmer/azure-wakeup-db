# azure-wakeup-db

If your Azure DB returns 40613 because it was paused, this GitHub action re-awakens it. Useful for workflows or ETL's that would otherwise quickly fail.

## Usage

```
docker run --rm ghcr.io/redmer/azure-wake-up-db --dsn <dsn>
```

Options:

- `--dsn`: The connection string to the Azure DB. Can be specified in many formats:
  - `sqlserver://username:password@host/instance`
  - `server=localhost;user id=sa;database=master;app name=MyAppName`
  - `odbc:server=localhost;user id=sa;password={foo;bar}`
- Or use the following specific options. They _will not_ be combined with DSN.
  - `--server`: Database host
  - `--port`: Database port (default: 1433)
  - `--instance`: SQL Server instance name (optional)
  - `--database`: Database name
  - `--user`: Database username
  - `--password`: Database password

  All variants of the connection string described at [microsoft/go-mssqldb].
  Kerberos or EntraID is not supported.

- Environment variables:
  - `WAKEUP_DSN`: Full DSN
  - Or use the following specific options. They _will not_ be combined with DSN.
    - `WAKEUP_SERVER`: Database host
    - `WAKEUP_DATABASE`: Database name
    - `WAKEUP_PORT`: Database port (default: 1433)
    - `WAKEUP_INSTANCE`: SQL Server instance name (optional)
    - `WAKEUP_USER`: Database username
    - `WAKEUP_PASSWORD`: Database password


[microsoft/go-mssqldb]: https://github.com/microsoft/go-mssqldb/blob/main/README.md#the-connection-string-can-be-specified-in-one-of-three-formats
