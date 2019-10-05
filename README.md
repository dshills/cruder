# CRUDER will generate CRUD Go functions for a MySQL table

## Usage

### -dbread string
		DB read package (default "*sqlx.DB")
### -dbwrite string
		DB write package (default "*sqlx.DB")
### -host string
		Connect to host (default "127.0.0.1")
### -name string
		Database name
### -null string
		Null package (default "sql")
### -pass string
		Password to use when connecting to server
### -port int
		Port number to use for connection (default 3306)
### -table string
		Table name to create CRUD
### -user string
		User for login (default "root")
