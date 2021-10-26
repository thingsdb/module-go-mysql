package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("Go MySQL Tutorial")

	// Open up our database connection.
	// I've set up a database on my local machine using phpmyadmin.
	// The database is called testDb
	db, err := sql.Open("mysql", "anja:pass@/test")

	// if there is an error opening the connection, handle it
	if err != nil {
		panic(err.Error())
	}

	// defer the close till after the main function has finished
	// executing
	defer db.Close()

	tx, err := db.Begin() // Tx options?
	if err != nil {
		fmt.Println(err.Error())
	}
	defer tx.Rollback() // The rollback will be ignored if the tx has been committed later in the function.

	// perform a db.Query insert
	stmt, err := db.Prepare("UPDATE users SET name='test' WHERE name='TEST';")
	if err != nil {
		fmt.Println(err.Error())
		return

	}
	defer stmt.Close()

	res, err := stmt.Exec()
	// if there is an error inserting, handle it
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// perform a db.Query insert
	stmt2, err := db.Prepare("UPDATE users SET name=@A WHERE name='TEST';")
	if err != nil {
		fmt.Println(err.Error())
		return

	}
	defer stmt.Close()

	res2, err := stmt2.Exec()
	// if there is an error inserting, handle it
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		fmt.Println(err.Error())
	}

	// // perform a db.Query insert
	// stmt, err := db.Prepare("UPDATE users, (SELECT id, name FROM users) AS us SET users.name = 'siri' WHERE users.id = us.id;")
	// if err != nil {
	// 	panic(err.Error())
	// }
	// defer stmt.Close()

	// res, err := stmt.Exec()
	// // if there is an error inserting, handle it
	// if err != nil {
	// 	panic(err.Error())
	// }

	// rows, err := stmt.Query()
	// // // be careful deferring Queries if you are using transactions
	// if err != nil {
	// 	panic(err.Error())
	// }

	// ret, err := returnRowsAsMap(rows)
	// if err != nil {
	// 	panic(err.Error())
	// }

	// lastInsertedID, err := res.LastInsertId()
	// if err != nil {
	// 	panic(err.Error())
	// }

	// rowsAffected, err := res.RowsAffected()
	// if err != nil {
	// 	panic(err.Error())
	// }

	fmt.Println(res, res2)

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
