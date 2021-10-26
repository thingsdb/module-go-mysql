package main

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type Next struct {
	AffectedRows *AffectedRows `msgpack:"affected_rows"`
	InsertRows   *InsertRows   `msgpack:"insert_rows"`
	Next         *Next         `msgpack:"next"`
	QueryRows    *QueryRows    `msgpack:"query_rows"`
}

type Query struct {
	Query  string        `msgpack:"query"`
	Params []interface{} `msgpack:"params"`
	Next   *Next         `msgpack:"next"`
}

type InsertRows Query
type QueryRows Query
type AffectedRows Query

type _DB interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

func (query *Query) handleQuery(_db _DB, ctx context.Context, fn func(stmt *sql.Stmt, ctx context.Context) (interface{}, error)) ([]interface{}, error) {
	stmt, err := _db.PrepareContext(ctx, query.Query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close() // Prepared statements take up server resources and should be closed after use.

	ret, err := fn(stmt, ctx)
	if err != nil {
		return nil, err
	}

	var res []interface{}
	res = append(res, ret)

	if query.Next != nil {
		next, err := query.Next.next(db, ctx)
		if err != nil {
			return nil, err
		}

		res = append(res, next...)
	}

	return res, nil
}

func (query *Query) handleTransaction(db *sql.DB, ctx context.Context, fn func(stmt *sql.Stmt, ctx context.Context) (interface{}, error)) (interface{}, error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable}) // Tx options?
	if err != nil {
		return nil, fmt.Errorf("Failed to start transaction: %s", err)
	}
	defer tx.Rollback() // The rollback will be ignored if the tx has been committed later in the function.

	ret, err := query.handleQuery(tx, ctx, fn)
	if err != nil {
		return nil, fmt.Errorf("Failed to execute transaction: %s", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("Failed to commit transaction: %s", err)
	}

	return ret, nil
}

func (query *Next) next(_db _DB, ctx context.Context) ([]interface{}, error) {
	num := 0
	var fn func(stmt *sql.Stmt, ctx context.Context) (interface{}, error)
	var q *Query
	if query.QueryRows != nil {
		q = (*Query)(query.QueryRows)
		fn = query.QueryRows.run
		num++
	}

	if query.InsertRows != nil {
		q = (*Query)(query.InsertRows)
		fn = query.InsertRows.run
		num++
	}

	if query.AffectedRows != nil {
		q = (*Query)(query.AffectedRows)
		fn = query.AffectedRows.run
		num++
	}

	if num == 0 {
		return nil, fmt.Errorf("Error: MySQL requires either `query_rows`, `insert_rows`, or `affected_rows``")
	}

	if num > 1 {
		return nil, fmt.Errorf("Error: MySQL requires either `query_rows`, `insert_rows`, or `affected_rows`, not more then one")
	}

	next, err := q.handleQuery(db, ctx, fn)
	if err != nil {
		return nil, err
	}

	return next, nil
}

func (query *QueryRows) run(stmt *sql.Stmt, ctx context.Context) (interface{}, error) {
	rows, err := stmt.QueryContext(ctx, query.Params...)
	if err != nil {
		return nil, fmt.Errorf("Query has failed: %s", err)
	}
	defer rows.Close()

	ret, err := returnRowsAsMap(rows)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (query *InsertRows) run(stmt *sql.Stmt, ctx context.Context) (interface{}, error) {
	res, err := stmt.ExecContext(ctx, query.Params...)
	if err != nil {
		return nil, fmt.Errorf("Query has failed: %s", err)
	}

	lastInsertId, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("Failed to get last insert ID: %s", err)
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("Failed to get affected rows: %s", err)
	}

	returnMessage := fmt.Sprintf("%s inserted, last inserted ID: %d", retMsg(affectedRows), lastInsertId)
	return returnMessage, nil
}

func (query *AffectedRows) run(stmt *sql.Stmt, ctx context.Context) (interface{}, error) {
	res, err := stmt.ExecContext(ctx, query.Params...)
	if err != nil {
		return nil, fmt.Errorf("Query has failed: %s", err)
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("Failed to get affected rows: %s", err)
	}

	returnMessage := fmt.Sprintf("%s affected", retMsg(affectedRows))
	return returnMessage, nil
}

func returnRowsAsMap(rows *sql.Rows) (interface{}, error) {
	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("Failed to get columns: %s", err)
	}

	// Make a slice for the values
	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	rowsMap := make([]map[string]interface{}, 0)
	for rows.Next() {
		// get RawBytes from data
		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, fmt.Errorf("Failed to scan rows: %s", err)
		}

		var value string
		row := make(map[string]interface{})
		for i, col := range values {
			// Here we can check if the value is nil (NULL value)
			if col == nil {
				value = "NULL"
			} else {
				value = string(col)
			}
			row[columns[i]] = value
		}
		rowsMap = append(rowsMap, row)
	}

	return rowsMap, nil
}
