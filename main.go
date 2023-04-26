package main

import (
	"flag"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func main() {
	user := "root"
	pass := ""
	host := "127.0.0.1"
	port := 3306
	dbName := ""
	tblName := ""
	nullPkg := "sql"
	public := false
	flag.StringVar(&user, "user", "root", "User for login")
	flag.StringVar(&dbName, "name", "", "Database name")
	flag.StringVar(&pass, "pass", "", "Password to use when connecting to server")
	flag.StringVar(&host, "host", "127.0.0.1", "Connect to host")
	flag.IntVar(&port, "port", 3306, "Port number to use for connection")
	flag.StringVar(&tblName, "table", "", "Table name to create CRUD")
	flag.StringVar(&nullPkg, "null", "sql", "Null package")
	flag.StringVar(&dbRead, "dbread", "*sqlx.DB", "DB read package")
	flag.StringVar(&dbWrite, "dbwrite", "*sqlx.DB", "DB write package")
	flag.BoolVar(&public, "public", public, "Should the functions be publicly accessable")
	flag.Parse()

	if dbName == "" || host == "" || pass == "" || user == "" || port == 0 || tblName == "" || nullPkg == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	setNullPkg(nullPkg)

	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v", user, pass, host, port, dbName)
	sdb, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	tbl := DBTable{Name: tblName}
	if err := tbl.Load(sdb, dbName, public); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(tbl.GoStruct())
	fmt.Println(tbl.GenCRUD())
}
