package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-txdb"
	_ "github.com/DATA-DOG/go-txdb"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/romanyx/polluter"
	"github.com/stretchr/testify/assert"
)

var dbname string = "test"

const mysqlSchema = `
CREATE TABLE IF NOT EXISTS users (
	id integer NOT NULL,
	name varchar(255) NOT NULL
);
`

const input = `
users:
- id: 1
  name: Jake
- id: 2
  name: Sarah
- id: 3
  name: Ted
`

func init() {
	err := newMySQL()
	if err != nil {
		log.Fatalf("prepare mysql: %v\n", err)
	}

	txdb.Register("mysqltx", "mysql", fmt.Sprintf("test:test@/%s", dbname))
}

func newMySQL() error {
	db, err := sql.Open("mysql", "test:test@/")
	if err != nil {
		return err
	}
	defer db.Close()

	if _, err := db.Exec("CREATE DATABASE IF NOT EXISTS " + dbname); err != nil {
		return err
	}

	if _, err := db.Exec("USE " + dbname); err != nil {
		return err
	}

	if _, err := db.Exec(mysqlSchema); err != nil {
		return errors.Wrap(err, "failed to create schema")
	}

	return nil
}

func TestExecQuery(t *testing.T) {
	tests := []struct {
		name    string
		request *reqMySQL
		result  func(res interface{}) bool
		wantErr bool
	}{
		{
			name: "Check fetch Rows",
			request: &reqMySQL{
				Query:   "SELECT * FROM users WHERE name = 'Ted';",
				Fetch:   Rows,
				Timeout: 10,
			},
			result: func(res interface{}) bool {
				resT, ok := res.([]map[string]interface{})
				return ok && resT[0]["name"] == "Ted"
			},
			wantErr: false,
		},
		{
			name: "Check fetch Columns",
			request: &reqMySQL{
				Query:   "SELECT * FROM users;",
				Fetch:   Columns,
				Timeout: 10,
			},
			result: func(res interface{}) bool {
				resT, ok := res.(map[string]interface{})
				return ok && resT["name"] == "VARCHAR"
			},
			wantErr: false,
		},
		{
			name: "Check fetch LastInsertId",
			request: &reqMySQL{
				Query:   "INSERT INTO users VALUES(?, ?);",
				Params:  []interface{}{4, "Lily"},
				Fetch:   LastInsertId,
				Timeout: 10,
			},
			result: func(res interface{}) bool {
				_, ok := res.(int64)
				return ok
			},
			wantErr: false,
		},
		{
			name: "Check fetch RowsAffected",
			request: &reqMySQL{
				Query:   "UPDATE users SET name = 'Teddy' WHERE name = 'Ted';",
				Fetch:   RowsAffected,
				Timeout: 10,
			},
			result: func(res interface{}) bool {
				resT, ok := res.(int64)
				return ok && resT == 1
			},
			wantErr: false,
		},
		{
			name: "Error unknown fetch type",
			request: &reqMySQL{
				Query:   "SELECT * FROM users;",
				Fetch:   "Unknown",
				Timeout: 10,
			},
			wantErr: true,
		},
		{
			name: "Error query syntax",
			request: &reqMySQL{
				Query:   "SELECT * FROM;",
				Fetch:   "Unknown",
				Timeout: 10,
			},
			wantErr: true,
		},
		{
			name: "Error unknown table",
			request: &reqMySQL{
				Query:   "SELECT * FROM users;",
				Fetch:   "Unknown",
				Timeout: 10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db, cleanup := exampleSuite(t)
			defer cleanup()

			ctx, cancelfunc := context.WithTimeout(context.Background(), time.Duration(tt.request.Timeout)*time.Second)
			defer cancelfunc()

			res, err := execQuery(db, ctx, tt.request)

			if tt.wantErr && err == nil {
				assert.NotNil(t, err)
				return
			}

			if !tt.wantErr && err != nil {
				assert.Nil(t, err)
				return
			}

			if err == nil {
				if equal := tt.result(res); !equal {
					t.Fatalf("Got unexpected result: %v", res)
				}
			}
		})
	}
}

func exampleSuite(t *testing.T) (*sql.DB, func() error) {
	db, cleanup := prepareMySQLDB(t)

	p := polluter.New(polluter.MySQLEngine(db))

	if err := p.Pollute(strings.NewReader(input)); err != nil {
		t.Fatalf("failed to pollute: %s", err)
	}

	return db, cleanup
}

func prepareMySQLDB(t *testing.T) (db *sql.DB, cleanup func() error) {
	cName := fmt.Sprintf("connection_%d", time.Now().UnixNano())
	db, err := sql.Open("mysqltx", cName)

	if err != nil {
		t.Fatalf("open mysqltx connection: %s", err)
	}

	return db, db.Close
}
