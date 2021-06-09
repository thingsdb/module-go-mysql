package main

type Fetch string

const (
	// ProtoModuleConf - when configuration data for the module is received
	DbStats Fetch = "Stats"

	// ProtoModuleConfOk - respond after successfully configuring the module
	LastInsertId Fetch = "LastInsertId"

	// ProtoModuleConfErr - respond with a configuration error
	RowsAffected Fetch = "RowsAffected"

	// ProtoModuleReq - when a request is received
	Columns Fetch = "Columns"

	// ProtoModuleRes is used to respond to a ProtoModuleReq package
	Rows Fetch = "Rows"
)