package main

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

func retMsg(num int64) string {
	label := "rows"
	if num < 2 {
		label = "row"
	}
	return fmt.Sprintf("%d %s", num, label)
}
