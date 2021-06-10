package main

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type _DB interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
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

func execQuery(_db _DB, ctx context.Context, req *reqMySQL) (interface{}, error) {
	stmt, err := _db.PrepareContext(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("Failed to prepare query: %s", err)
	}
	defer stmt.Close() // Prepared statements take up server resources and should be closed after use.

	var ret interface{}
	switch fetch := req.Fetch; fetch {
	case LastInsertId:
		res, err := stmt.ExecContext(ctx, req.Params...)
		if err != nil {
			return nil, fmt.Errorf("Query has failed: %s", err)
		}

		ret, err = res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("Failed to get last insert ID: %s", err)
		}
	case RowsAffected:
		res, err := stmt.ExecContext(ctx, req.Params...)
		if err != nil {
			return nil, fmt.Errorf("Query has failed: %s", err)
		}

		ret, err = res.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("Failed to get affected rows: %s", err)
		}
	case Columns:
		rows, err := stmt.QueryContext(ctx, req.Params...)
		if err != nil {
			return nil, fmt.Errorf("Query has failed: %s", err)
		}
		defer rows.Close()

		names, err := rows.Columns()
		if err != nil {
			return nil, fmt.Errorf("Failed to get columns: %s", err)
		}
		types, err := rows.ColumnTypes()
		if err != nil {
			return nil, fmt.Errorf("Failed to get column types: %s", err)
		}

		row := make(map[string]interface{})
		for i, col := range names {
			row[col] = types[i].DatabaseTypeName()
		}
		ret = row

	case Rows:
		rows, err := stmt.QueryContext(ctx, req.Params...)
		if err != nil {
			return nil, fmt.Errorf("Query has failed: %s", err)
		}
		defer rows.Close()

		ret, err = returnRowsAsMap(rows)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("Fetch parameter unknown; valid options are `Columns`, `Rows`, `LastInsertId` or `RowsAffected`: %s", err)
	}

	return ret, nil
}
