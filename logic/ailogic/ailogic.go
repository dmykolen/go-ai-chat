package ailogic

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/kr/pretty"
	"github.com/samber/lo"
	"gitlab.dev.ict/golang/libs/gohttp"
)

const (
	Width = 80
)

var (
	sep = strings.Repeat("#", Width)
	HttpCl = gohttp.NewHttpClient(gohttp.WithPRX(gohttp.ProxyAstelit), gohttp.WithTO(30)).Client
)

var PrettyPrintStruct = func(v interface{}) string {
	return fmt.Sprintf("\n%s\n%# v\n%s\n", sep, pretty.Formatter(v), sep)
}

var db *sql.DB = dbMemorySqlite()
var err error

func SqLiteHST() *sql.DB {
	return db
}

func DBMemory(ds ...string) *sql.DB {
	if db == nil {
		db = dbMemorySqlite(ds...)
	}
	return db
}

func dbMemorySqlite(ds ...string) *sql.DB {
	os.MkdirAll("./data", 0755)
	dbCon, err := sql.Open("sqlite3", lo.TernaryF(len(ds) > 0 && ds[0] != "", func() string { return ds[0] }, func() string { return "data/history.db" }))
	if err != nil {
		panic(err)
	}
	// db = dbCon
	return dbCon
}
