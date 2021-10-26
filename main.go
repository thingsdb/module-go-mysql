package main

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	timod "github.com/thingsdb/go-timod"
	"github.com/vmihailenco/msgpack"
)

var db *sql.DB = nil
var mux sync.Mutex

type confMySQL struct {
	Dsn             string `msgpack:"dsn"`
	ConnMaxLifetime int64  `msgpack:"conn_max_lifetime"`
	MaxOpenConn     int    `msgpack:"max_open_conn"`
	MaxIdleConn     int    `msgpack:"max_idle_conn"`
	MaxIdleTimeConn int64  `msgpack:"max_idle_time_conn"`
}

type reqMySQL struct {
	GetDBStats   bool          `msgpack:"get_db_stats"`
	InsertRows   *InsertRows   `msgpack:"insert_rows"`
	QueryRows    *QueryRows    `msgpack:"query_rows"`
	RowsAffected *RowsAffected `msgpack:"rows_affected"`
	Timeout      int           `msgpack:"timeout"`
	Transaction  bool          `msgpack:"transaction"`
}

func handleConf(config *confMySQL) {
	mux.Lock()
	defer mux.Unlock()

	if db != nil {
		db.Close()
	}

	var err error
	db, err = sql.Open("mysql", config.Dsn)
	if err != nil {
		timod.WriteConfErr()
		return
	}

	if config.MaxIdleTimeConn != 0 {
		db.SetConnMaxLifetime(time.Minute * time.Duration(config.ConnMaxLifetime))
	}

	if config.MaxIdleTimeConn != 0 {
		db.SetMaxOpenConns(config.MaxOpenConn)
	}

	if config.MaxIdleTimeConn != 0 {
		db.SetMaxIdleConns(config.MaxIdleConn)
	}

	if config.MaxIdleTimeConn != 0 {
		db.SetConnMaxIdleTime(time.Minute * time.Duration(config.MaxIdleTimeConn))
	}

	timod.WriteConfOk()
}

func onModuleReq(pkg *timod.Pkg) {
	mux.Lock()
	defer mux.Unlock()

	if db == nil || db.Ping() != nil {
		timod.WriteEx(
			pkg.Pid,
			timod.ExOperation,
			"Error: MySQL is not connected; please check the module configuration")
		return
	}

	var req reqMySQL
	err := msgpack.Unmarshal(pkg.Data, &req)
	if err != nil {
		timod.WriteEx(
			pkg.Pid,
			timod.ExBadData,
			"Error: Failed to unpack MySQL request")
		return
	}

	num := 0
	var fn func(stmt *sql.Stmt, ctx context.Context) (interface{}, error)
	var q *Query
	if req.QueryRows != nil {
		q = (*Query)(req.QueryRows)
		fn = req.QueryRows.run
		num++
	}

	if req.InsertRows != nil {
		q = (*Query)(req.InsertRows)
		fn = req.InsertRows.run
		num++
	}

	if req.RowsAffected != nil {
		q = (*Query)(req.RowsAffected)
		fn = req.RowsAffected.run
		num++
	}

	if req.GetDBStats {
		if num == 0 {
			ret := db.Stats()
			timod.WriteResponse(pkg.Pid, ret)
			return
		}
		num++
	}

	if num == 0 {
		timod.WriteEx(
			pkg.Pid,
			timod.ExOperation,
			"Error: MySQL requires either `query_rows`, `insert_rows`, `rows_affected`, or `get_db_stats`")
		return
	}

	if num > 1 {
		timod.WriteEx(
			pkg.Pid,
			timod.ExBadData,
			"Error: MySQL requires either `query_rows`, `insert_rows`, `rows_affected`, or `get_db_stats, not more then one")
		return
	}

	if req.Timeout == 0 {
		req.Timeout = 10
	}

	ctx, cancelfunc := context.WithTimeout(context.Background(), time.Duration(req.Timeout)*time.Second)
	defer cancelfunc()

	var ret interface{}
	if req.Transaction {
		ret, err = q.handleTransaction(db, ctx, fn)
	} else {
		ret, err = q.handleQuery(db, ctx, fn)
	}

	if err != nil {
		timod.WriteEx(
			pkg.Pid,
			timod.ExOperation,
			err.Error())
		return
	}

	timod.WriteResponse(pkg.Pid, ret)
}

func handler(buf *timod.Buffer, quit chan bool) {
	for {
		select {
		case pkg := <-buf.PkgCh:
			switch timod.Proto(pkg.Tp) {
			case timod.ProtoModuleConf:
				var conf confMySQL
				err := msgpack.Unmarshal(pkg.Data, &conf)
				if err == nil {
					handleConf(&conf)
				} else {
					log.Println("Error: Failed to unpack MySQL configuration")
					timod.WriteConfErr()
				}

			case timod.ProtoModuleReq:
				onModuleReq(pkg)

			default:
				log.Printf("Error: Unexpected package type: %d", pkg.Tp)
			}
		case err := <-buf.ErrCh:
			// In case of an error you probably want to quit the module.
			// ThingsDB will try to restart the module a few times if this
			// happens.
			log.Printf("Error: %s", err)
			quit <- true
		}
	}
}

func main() {
	// Starts the module
	timod.StartModule("mysql", handler)

	if db != nil {
		db.Close()
	}
}
