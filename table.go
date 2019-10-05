package main

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// DBTable represents a table in a database
type DBTable struct {
	Name      string
	goName    string
	Fields    []DBField
	priKeyCnt int
	fNames    []string
}

// DBField represents a field in a table
type DBField struct {
	ColumnName    string         `db:"COLUMN_NAME"`
	DataType      string         `db:"DATA_TYPE"`
	ColumnType    string         `db:"COLUMN_TYPE"`
	IsNullable    string         `db:"IS_NULLABLE"`
	ColumnKey     string         `db:"COLUMN_KEY"`
	Extra         string         `db:"EXTRA"`
	ColumnDefault sql.NullString `db:"COLUMN_DEFAULT"`
}

// Load will load a tables schema
func (t *DBTable) Load(db *sqlx.DB, dbName string) error {
	str := `
	SELECT COLUMN_NAME, DATA_TYPE, COLUMN_TYPE, COLUMN_DEFAULT, IS_NULLABLE, COLUMN_KEY, EXTRA
	FROM INFORMATION_SCHEMA.COLUMNS
	WHERE TABLE_NAME = ?
		AND TABLE_SCHEMA = ?
	ORDER BY ORDINAL_POSITION
	`
	t.goName = singular(goName(t.Name))
	if err := db.Select(&t.Fields, str, t.Name, dbName); err != nil {
		return err
	}
	if len(t.Fields) == 0 {
		return fmt.Errorf("Table not found")
	}
	t.priKeyCnt = 0
	for _, f := range t.Fields {
		if f.ColumnKey == "PRI" {
			t.priKeyCnt++
		}
	}
	return nil
}

// GoStruct returns a go structure for the table
func (t DBTable) GoStruct() string {
	structStr := fmt.Sprintf("// %s is a database struct\n", t.goName)
	structStr += fmt.Sprintf("type %s struct {\n", t.goName)
	for _, fld := range t.Fields {
		f := goName(fld.ColumnName)
		nullable := false
		if fld.IsNullable == "YES" {
			nullable = true
		}
		ty, _ := goType(fld.DataType, nullable)
		structStr += fmt.Sprintf("\t%s %s `db:\"%s\" json:\"%s\"`\n", f, ty, fld.ColumnName, fld.ColumnName)
	}
	structStr += "}\n"
	return structStr
}

// GoImports returns a go import statement
func (t DBTable) GoImports() string {
	imap := make(map[string]bool)
	for _, fld := range t.Fields {
		nullable := false
		if fld.IsNullable == "YES" {
			nullable = true
		}
		_, imp := goType(fld.DataType, nullable)
		for _, im := range imp {
			imap[im] = true
		}
	}

	imports := ""
	if len(imap) > 0 {
		imports = "import (\n"
		for k := range imap {
			imports += "\t\"" + k + "\"\n"
		}
		imports += ")\n"
	}
	return imports
}

// GenCRUD generates create, read, update and delete functions
func (t DBTable) GenCRUD() string {
	if t.priKeyCnt == 0 {
		return ""
	}
	builder := strings.Builder{}
	if t.priKeyCnt == 1 {
		builder.WriteString(t.GenCreate())
		builder.WriteString(t.GenUpdate())
		builder.WriteString(t.GenDelete())
		builder.WriteString(t.GenRead())
		return builder.String()
	}

	builder.WriteString(t.GenSet())
	builder.WriteString(t.GenRemove())
	builder.WriteString(t.GenRead())
	return builder.String()
}

// GenSet generates a create function string
func (t DBTable) GenSet() string {
	flds := []string{}
	fval := []string{}
	for _, f := range t.Fields {
		switch {
		case f.ColumnDefault.String == "CURRENT_TIMESTAMP":
		default:
			flds = append(flds, f.ColumnName)
			fval = append(fval, ":"+f.ColumnName)
		}
	}
	builder := strings.Builder{}
	builder.WriteString("// Set will create a record\n")
	builder.WriteString(fmt.Sprintf("func (st *%s)Set(sdb %s) error {\n", t.goName, dbWrite))
	builder.WriteString("\tstr := `\n")
	builder.WriteString(fmt.Sprintf("\tINSERT INTO %s\n", t.Name))
	builder.WriteString(fmt.Sprintf("\t(%s)\n", strings.Join(flds, ", ")))
	builder.WriteString("\tVALUES\n")
	builder.WriteString(fmt.Sprintf("\t(%s)\n", strings.Join(fval, ", ")))
	builder.WriteString("\t`\n")
	builder.WriteString("\t_, err := sdb.NamedExec(str, st)\n")
	builder.WriteString("\treturn err\n")
	builder.WriteString("}\n")
	return builder.String()
}

// GenCreate generates a create function string
func (t DBTable) GenCreate() string {
	flds := []string{}
	fval := []string{}
	key := ""
	for _, f := range t.Fields {
		switch {
		case f.ColumnKey == "PRI":
			key = goName(f.ColumnName)
		case f.ColumnDefault.String == "CURRENT_TIMESTAMP":
		default:
			flds = append(flds, f.ColumnName)
			fval = append(fval, ":"+f.ColumnName)
		}
	}
	builder := strings.Builder{}
	builder.WriteString("// Create will create a record\n")
	builder.WriteString(fmt.Sprintf("func (st *%s)Create(sdb %s) error {\n", t.goName, dbWrite))
	builder.WriteString("\tstr := `\n")
	builder.WriteString(fmt.Sprintf("\tINSERT INTO %s\n", t.Name))
	builder.WriteString(fmt.Sprintf("\t(%s)\n", strings.Join(flds, ", ")))
	builder.WriteString("\tVALUES\n")
	builder.WriteString(fmt.Sprintf("\t(%s)\n", strings.Join(fval, ", ")))
	builder.WriteString("\t`\n")
	builder.WriteString("\tres, err := sdb.NamedExec(str, st)\n")
	builder.WriteString("\tif err != nil {\n")
	builder.WriteString("\t\treturn err\n")
	builder.WriteString("\t}\n")
	builder.WriteString(fmt.Sprintf("\tst.%s, err = res.LastInsertId()\n", key))
	builder.WriteString("\treturn err\n")
	builder.WriteString("}\n")
	return builder.String()
}

// GenUpdate generates an update function string
func (t DBTable) GenUpdate() string {
	flds := []string{}
	key := ""
	for _, f := range t.Fields {
		switch {
		case f.ColumnKey == "PRI":
			key = f.ColumnName
		case f.ColumnDefault.String == "CURRENT_TIMESTAMP":
		default:
			flds = append(flds, fmt.Sprintf("\t\t%s = :%s", f.ColumnName, f.ColumnName))
		}
	}
	builder := strings.Builder{}
	builder.WriteString("// Update will update a record\n")
	builder.WriteString(fmt.Sprintf("func (st *%s)Update(sdb %s) error {\n", t.goName, dbWrite))
	builder.WriteString("\tstr := `\n")
	builder.WriteString(fmt.Sprintf("\tUPDATE %s SET\n", t.Name))
	builder.WriteString(strings.Join(flds, ",\n"))
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("\tWHERE %s = :%s\n", key, key))
	builder.WriteString("\t`\n")
	builder.WriteString("\t_, err := sdb.NamedExec(str, st)\n")
	builder.WriteString("\treturn err\n")
	builder.WriteString("}\n")
	return builder.String()
}

// GenDelete generates a delete function
func (t DBTable) GenDelete() string {
	flds := []string{}
	key := ""
	gokey := ""
	for _, f := range t.Fields {
		switch {
		case f.ColumnKey == "PRI":
			key = f.ColumnName
			gokey = goName(f.ColumnName)
		case f.ColumnDefault.String == "CURRENT_TIMESTAMP":
		default:
			flds = append(flds, f.ColumnName)
		}
	}
	builder := strings.Builder{}
	builder.WriteString("// Delete will delete a record\n")
	builder.WriteString(fmt.Sprintf("func (st *%s)Delete(sdb %s) error {\n", t.goName, dbWrite))
	builder.WriteString(fmt.Sprintf("\tstr := \"DELETE FROM %s WHERE %s = ?\"\n", t.Name, key))
	builder.WriteString(fmt.Sprintf("\t_, err := sdb.Exec(str, st.%s)\n", gokey))
	builder.WriteString("\treturn err\n")
	builder.WriteString("}\n")
	return builder.String()
}

// GenRemove generates a remove function
func (t DBTable) GenRemove() string {
	keys := []string{}
	gokeys := []string{}
	for _, f := range t.Fields {
		if f.ColumnKey == "PRI" {
			keys = append(keys, fmt.Sprintf("\t%s = ?", f.ColumnName))
			gokeys = append(gokeys, "st."+goName(f.ColumnName))
		}
	}
	builder := strings.Builder{}
	builder.WriteString("// Remove will delete a record\n")
	builder.WriteString(fmt.Sprintf("func (st *%s)Remove(sdb %s) error {\n", t.goName, dbWrite))
	builder.WriteString("\tstr := `\n")
	builder.WriteString(fmt.Sprintf("\tDELETE FROM %s WHERE\n", t.Name))
	builder.WriteString(strings.Join(keys, ",\n"))
	builder.WriteString("\n\t`\n")
	builder.WriteString(fmt.Sprintf("\t_, err := sdb.Exec(str, %s)\n", strings.Join(gokeys, ", ")))
	builder.WriteString("\treturn err\n")
	builder.WriteString("}\n")
	return builder.String()
}

// GenRead generates a read function
func (t DBTable) GenRead() string {
	keys := []string{}
	gokeys := []string{}
	for _, f := range t.Fields {
		if f.ColumnKey == "PRI" {
			keys = append(keys, fmt.Sprintf("%s = ?", f.ColumnName))
			gokeys = append(gokeys, "st."+goName(f.ColumnName))
		}
	}
	builder := strings.Builder{}
	builder.WriteString("// Read will Read a record\n")
	builder.WriteString(fmt.Sprintf("func (st *%s)Read(sdb %s) error {\n", t.goName, dbRead))
	builder.WriteString(fmt.Sprintf("\tstr := \"SELECT * FROM %s WHERE %s\"\n", t.Name, strings.Join(keys, " AND ")))
	builder.WriteString(fmt.Sprintf("\treturn sdb.Get(st, str, %s)\n", strings.Join(gokeys, ", ")))
	builder.WriteString("}\n")
	return builder.String()
}

// Dump will print the table's fields
func (t DBTable) Dump() {
	fmt.Println(t.Name)
	for _, f := range t.Fields {
		fmt.Printf("%+v\n", f)
	}
}

func getTables(db *sqlx.DB, name string) ([]DBTable, error) {
	tblNames, err := tableList(db, name)
	if err != nil {
		return nil, err
	}
	elist := []string{}
	tables := []DBTable{}
	for _, tbl := range tblNames {
		fields, err := tableFields(db, name, tbl)
		if err != nil {
			elist = append(elist, err.Error())
			continue
		}
		table := DBTable{Name: tbl, Fields: fields}
		tables = append(tables, table)
	}
	if len(elist) > 0 {
		return tables, fmt.Errorf(strings.Join(elist, ", "))
	}
	return tables, nil
}

func tableList(db *sqlx.DB, dbname string) ([]string, error) {
	str := "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ?"
	tnames := []string{}
	err := db.Select(&tnames, str, dbname)
	return tnames, err
}

func tableFields(db *sqlx.DB, dbName, tblName string) ([]DBField, error) {
	str := `
	SELECT COLUMN_NAME, DATA_TYPE, COLUMN_TYPE, IS_NULLABLE, COLUMN_KEY, EXTRA
	FROM INFORMATION_SCHEMA.COLUMNS
	WHERE TABLE_NAME = ?
		AND TABLE_SCHEMA = ?
	ORDER BY ORDINAL_POSITION`

	fields := []DBField{}
	err := db.Select(&fields, str, tblName, dbName)
	return fields, err
}
