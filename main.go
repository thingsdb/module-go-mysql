// Demo is a ThingsDB module which may be used as a template to build modules.
//
// This module simply extract a given `message` property from a request and
// returns this message.
//
// For example:
//
//     // Create the module (@thingsdb scope)
//     new_module('DEMO', 'demo', nil, nil);
//
//     // When the module is loaded, use the module in a future
//     future({
//       module: 'DEMO',
//       message: 'Hi ThingsDB module!',
//     }).then(|msg| {
//	      `Got the message back: {msg}`
//     });
//
package main

import (
	"context"
	"database/sql"
	"fmt"
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
	Dsn string         `msgpack:"dsn"`
	ConnMaxLifetime int64         `msgpack:"conn_max_lifetime"`
	MaxOpenConn int         `msgpack:"max_open_conn"`
	MaxIdleConn  int `msgpack:"max_idle_conn"`
	MaxIdleTimeConn  int `msgpack:"max_idle_time_conn"`
}

type reqMySQL struct {
	Query   string `msgpack:"query"`
	Params  []interface{} `msgpack:"params"`
	Fetch Fetch `msgpack:"fetch"` // []string?
	Transaction bool `msgpack:"transaction"`
	Timeout int `msgpack:"timeout"`
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
		// TODO
		timod.WriteConfErr()
		panic(err.Error())  // Just for example purpose. You should use proper error handling instead of panic
	}

	db.SetConnMaxLifetime(time.Minute * time.Duration(config.ConnMaxLifetime))
	db.SetMaxOpenConns(config.MaxOpenConn)
	db.SetMaxIdleConns(config.MaxIdleConn)

	if config.MaxIdleTimeConn {
		db.SetConnMaxIdleTime(time.Minute * time.Duration(config.MaxIdleTimeConn))
	}

	timod.WriteConfOk()
}

func handleQuery(pkg *timod.Pkg, req *reqMySQL) {
	if req.Timeout == 0 {
		req.Timeout = 10
	}

	ctx, cancelfunc := context.WithTimeout(context.Background(), time.Duration(req.Timeout) * time.Second)
    defer cancelfunc()

	ret, err := execQuery(db, ctx, req)
	if err != nil {
		timod.WriteEx(
			pkg.Pid,
			timod.ExOperation,
			err.Error())
		return
	}

	timod.WriteResponse(pkg.Pid, ret)
}

func handleTransaction(pkg *timod.Pkg, req *reqMySQL) {
	if req.Timeout == 0 {
		req.Timeout = 10
	}

	ctx, cancelfunc := context.WithTimeout(context.Background(), time.Duration(req.Timeout) * time.Second)
    defer cancelfunc()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable}) // Tx options?
	if err != nil {
		timod.WriteEx(
			pkg.Pid,
			timod.ExOperation,
			fmt.Sprintf("Failed to start transaction: %s", err))
		return
	}
	defer tx.Rollback() // The rollback will be ignored if the tx has been committed later in the function.

	ret, err := execQuery(tx, ctx, req)
	if err != nil {
		timod.WriteEx(
			pkg.Pid,
			timod.ExOperation,
			fmt.Sprintf("Failed to execute transaction: %s", err))
		return
	}

	if err := tx.Commit(); err != nil {
		timod.WriteEx(
			pkg.Pid,
			timod.ExOperation,
			fmt.Sprintf("Failed to commit transaction: %s", err))
		return
	}

	timod.WriteResponse(pkg.Pid, ret)
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

	if req.Fetch == DbStats {
		ret := db.Stats()
		timod.WriteResponse(pkg.Pid, ret)
		return
	}

	if req.Query == "" {
		timod.WriteEx(
			pkg.Pid,
			timod.ExBadData,
			"Error: MySQL requires `query`")
		return
	}

	if req.Transaction {
		handleTransaction(pkg, &req)
	} else {
		handleQuery(pkg, &req)
	}
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