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

func TestRunQuery(t *testing.T) {
	tests := []struct {
		name    string
		request *reqMySQL
		result  func(res interface{}) bool
		wantErr bool
	}{
		{
			name: "Check QueryRows",
			request: &reqMySQL{
				QueryRows: &QueryRows{
					Query: "SELECT * FROM users WHERE name = 'Ted';",
				},
				Timeout: 10,
			},
			result: func(res interface{}) bool {
				resT, ok := res.([]map[string]interface{})
				return ok && resT[0]["name"] == "Ted"
			},
			wantErr: false,
		},
		{
			name: "Check InsertRows",
			request: &reqMySQL{
				InsertRows: &InsertRows{
					Query:  "INSERT INTO users VALUES(?, ?);",
					Params: []interface{}{4, "Lily"},
				},
				Timeout: 10,
			},
			result: func(res interface{}) bool {
				str, ok := res.(string)
				if ok {
					return str == "1 row inserted, last inserted ID: 0"
				}
				return ok
			},
			wantErr: false,
		},
		{
			name: "Check ",
			request: &reqMySQL{
				AffectedRows: &AffectedRows{
					Query: "UPDATE users SET name = 'Teddy' WHERE name = 'Ted';",
				},
				Timeout: 10,
			},
			result: func(res interface{}) bool {
				str, ok := res.(string)
				if ok {
					return str == "1 row affected"
				}
				return ok
			},
			wantErr: false,
		},
		{
			name: "Error: MySQL requires either `query_rows`, `insert_rows`, `affected_rows`, or `get_db_stats`",
			request: &reqMySQL{
				Timeout: 10,
			},
			wantErr: true,
		},
		{
			name: "Error: MySQL requires either `query_rows`, `insert_rows`, `affected_rows`, or `get_db_stats, not more then one",
			request: &reqMySQL{
				QueryRows: &QueryRows{
					Query: "SELECT * FROM users WHERE name = 'Ted';",
				},
				InsertRows: &InsertRows{
					Query:  "INSERT INTO users VALUES(?, ?);",
					Params: []interface{}{5, "Siri"},
				},
				Timeout: 10,
			},
			wantErr: true,
		},
		{
			name: "Error query syntax",
			request: &reqMySQL{
				QueryRows: &QueryRows{
					Query: "SELECT * FROM;",
				},
				Timeout: 10,
			},
			wantErr: true,
		},
		{
			name: "Error unknown table",
			request: &reqMySQL{
				QueryRows: &QueryRows{
					Query: "SELECT * FROM unknown;",
				},
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

			num := 0
			var fn func(stmt *sql.Stmt, ctx context.Context) (interface{}, error)
			var q *Query
			if tt.request.QueryRows != nil {
				q = (*Query)(tt.request.QueryRows)
				fn = tt.request.QueryRows.run
				num++
			}

			if tt.request.InsertRows != nil {
				q = (*Query)(tt.request.InsertRows)
				fn = tt.request.InsertRows.run
				num++
			}

			if tt.request.AffectedRows != nil {
				q = (*Query)(tt.request.AffectedRows)
				fn = tt.request.AffectedRows.run
				num++
			}

			if tt.request.GetDBStats {
				if num == 0 {
					return
				}
				num++
			}

			if num == 0 {
				return
			}

			if num > 1 {
				return
			}

			if tt.request.Timeout == 0 {
				tt.request.Timeout = 10
			}

			ctx, cancelfunc := context.WithTimeout(context.Background(), time.Duration(tt.request.Timeout)*time.Second)
			defer cancelfunc()

			res, err := q.handleQuery(db, ctx, fn)

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
