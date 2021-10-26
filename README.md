# MySQL ThingsDB Module (Go)

MySQL module written using the [Go language](https://golang.org). This module can be used to communicate with MySQL

## Installation

Install the module by running the following command in the `@thingsdb` scope:

```javascript
new_module('mysql', 'github.com/thingsdb/module-go-mysql');
```

Optionally, you can choose a specific version by adding a `@` followed with the release tag. For example: `@v0.1.0`.

## Configuration

The MySQL module requires a configuration with the following properties:

Property           | Type                              | Description
-------------------| --------------------------------- | -----------
dsn                | str (required)                    | Data source name; DSN format: `username:password@protocol(address)/dbname?param=value`
conn_max_lifetime  | int (time unit: minute; optional) | Maximum amount of time a connection may be reused.
max_idle_conn      | int (optional)                    | Maximum number of connections in the idle connection pool.
max_open_conn      | int (optional)                    | Maximum number of open connections to the database.
max_idle_time_conn | int (time unit: minute; optional) | Maximum amount of time a connection may be idle.

Example configuration:

```javascript
set_module_conf('mysql', {
    dsn: "username:password@/dbname",
    conn_max_lifetime: 3,
    max_idle_conn: 10,
    max_open_conn: 1
});
```

## Exposed functions

Name                            | Description
------------------------------- | -----------
[affected_rows](#affected_rows) | Make a query request that returns the number of affected rows.
[get_db_stats](#get_db_stats)   | Get the database statistics.
[insert_rows](#insert_rows)     | Insert rows and get in return the last inserted ID and number of inserted rows.
[query_rows](#query_rows)       | Make a query request that returns a array with maps containing the column names and corresponding values.

### AffectedRows

#### Arguments

 Argument       | Type                  | Description
--------------- | --------------------- | -----------
`affected_rows` | `Query` (required)    | Thing with `affected_rows` properties, see [Query](#Query).
`transaction`   | `bool` (optional)     | Indicates if the query needs to be wrapped in transaction statements or not.
`settings`      | `Settings` (optional) | Thing with `settings` properties, see [Settings](#Settings).

#### Example:

```javascript
mysql.affected_rows({
    query: 'UPDATE pet SET name='Marley' WHERE name='Fleddy';',
    params: ['dog'],
}, false, {deep: 1, timeout: 10}).then(|res| {
    res; // just return the response.
});
```

### GetDbStats

#### Arguments

No arguments required.

The following database statistics are returned:

* Idle: the number of idle connections.
* InUse: the number of connections currently in use.
* MaxIdleClosed: the total number of connections closed due to SetMaxIdleConns.
* MaxIdleTimeClosed: the total number of connections closed due to SetConnMaxIdleTime.
* MaxLifetimeClosed: the total number of connections closed due to SetConnMaxLifetime.
* MaxOpenConnections: maximum number of open connections to the database.
* OpenConnections: the number of established connections both in use and idle.
* WaitCount: the total number of connections waited for.
* WaitDuration: the total time blocked waiting for a new connection.

#### Example:

```javascript
mysql.get_db_stats().then(|res| {
    res; // just return the response.
});
```

### InsertRows

#### Arguments

 Argument     | Type                  | Description
------------- | --------------------- | -----------
`insert_rows` | `Query` (required)    | Thing with `insert_rows` properties, see [Query](#Query).
`transaction` | `bool` (optional)     | Indicates if the query needs to be wrapped in transaction statements or not.
`settings`    | `Settings` (optional) | Thing with `settings` properties, see [Settings](#Settings).

#### Example:

```javascript
mysql.insert_rows({
    query: 'INSERT INTO pet VALUES(?, ?);',
    params: ['Fleddy', 'dog'],
}).then(|res| {
    res; // just return the response.
});
```

### QueryRows

#### Arguments

 Argument     | Type                  | Description
------------- | --------------------- | -----------
`query_rows`  | `Query` (required)    | Thing with `query_rows` properties, see [Query](#Query).
`transaction` | `bool` (optional)     | Indicates if the query needs to be wrapped in transaction statements or not.
`settings`    | `Settings` (optional) | Thing with `settings` properties, see [Settings](#Settings).

#### Example:

```javascript
mysql.query_rows({
    query: 'SELECT * FROM pet WHERE species = ?;',
    params: ['dog'],
}).then(|res| {
    res; // just return the response.
});
```

### Types

#### Query

 Argument | Type                | Description
--------- | ------------------- | -----------
`query`   | `string` (required) | Query string or template query string with `?`.
`params`  | `array` (optional)  | The values that need to be inserted in the query at `?`.
`next`    | `Next` (optional)   | Thing with `next` properties, see [Next](#Next).

#### Next

 Argument       | Type               | Description
--------------- | ------------------ | -----------
`affected_rows` | `Query` (required) | Thing with `affected_rows` properties, see [Query](#Query).
`insert_rows`   | `Query` (required) | Thing with `insert_rows` properties, see [Query](#Query).
`query_rows`    | `Query` (required) | Thing with `query_rows` properties, see [Query](#Query).
`next`          | `Next` (optional)  | Thing with `next` properties, see [Next](#Next).

#### Settings

 Argument | Type                 | Description
--------- | -------------------- | -----------
`deep`    | `int` (optional)     | Deep value of the thing with `affected_rows` properties (Default: 3).
`timeout` | `integer` (optional) | Provide a custom timeout in seconds (Default: 10 seconds).
