package dbinfocollector

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/godror/godror"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
	gl "gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/utils"
	"gorm.io/gorm"
	"gorm.io/hints"
)

const regExpTblsCustomerModel = "^CM_.*"

var d *DBOracleGorm

// var d DBSchemaInfoProvider
var ctx = context.Background()
var log = gl.Defult()

func init() {
	d = NewDBOracleGorm(DSN(os.Getenv("DB_URL_TM_CIM")), Logger(log), DebugMode(false), DictsInclude(), DictsFilters(ScopeDictTablesFilterByRegExp(RegExpDict_CustomerModel), ScopeDictTablesFilterByNumRows(100), ScopeExcludeBkpTmp))
	d.SetFormatter(&DbFormatterClaude{})
}

func TestDBOracleGorm_GetTablesInfo(t *testing.T) {
	os.MkdirAll(t.Name(), 0755)
	tests := []struct {
		name, tablePattern string
	}{
		// {"1", "CM_%"},
		{"1", regExpTblsCustomerModel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.GetTablesInfoByPattern(ctx, tt.tablePattern)
			t.Log(err)
			os.WriteFile(t.Name()+".json", utils.Json(got), 0644)
			os.WriteFile(t.Name()+".txt", []byte(d.FormatTables(ctx, got)), 0644)
		})
	}
}

func Test_DbSelect_by_patterns(t *testing.T) {
	var tableNames []string
	t.Run("1", func(t *testing.T) {
		tx := d.Debug().Raw(getTableNamesQuery, "CM_%")
		t.Log(tx.ToSQL(func(x *gorm.DB) *gorm.DB { return x.Scan(&tableNames) }))
		tx.Pluck("TABLE_NAME", &tableNames)
		t.Logf("len=%d TABLES=[%#v]", len(tableNames), tableNames)
	})
	t.Run("2", func(t *testing.T) {
		tx := d.Debug().Raw(getTableNamesQueryByRegExp, "^CM_.*")
		t.Log(tx.ToSQL(func(x *gorm.DB) *gorm.DB { return x.Scan(&tableNames) }))
		tx.Pluck("TABLE_NAME", &tableNames)
		t.Logf("len=%d TABLES=[%#v]", len(tableNames), tableNames)
	})
}

func TestDBOracleGorm_GetTableInfo(t *testing.T) {
	os.MkdirAll(t.Name(), 0755)
	tests := []struct {
		name, table string
	}{
		{"1", "CM_ACCOUNT_ATTRIBUTE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.GetTableInfo(ctx, tt.table)
			t.Log(err)
			t.Log(utils.JsonPrettyStr(got))
			t.Log(d.FormatTableInfo(ctx, got))
		})
	}
}

func TestDBOracleGorm_GetDicts(t *testing.T) {
	var res2 []map[string]interface{}
	tx2 := d.G().
		Clauses(hints.New("PARALLEL(4)")).
		Debug().Session(&gorm.Session{PrepareStmt: true}).
		Table("CM_DOCUMENT_TYPE").
		Select("CODE", "DESCRIPTION", "IS_SERIES", "IS_NUMBER").
		Order("code DESC")
	t.Log("To_SQL ---->", tx2.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Scan(&res2) }))
	tx2.Scan(&res2)
	t.Logf("res2: isNil=%t Len=%d", res2 == nil, len(res2))
	t.Logf("res2: %# v", pretty.Formatter(res2))

	// res2 = []map[string]interface{}{}
	res2 = []map[string]interface{}{}
	tx2.Limit(4).Scan(&res2)
	t.Logf("res2: isNil=%t Len=%d", res2 == nil, len(res2))
	t.Logf("res2: %# v", pretty.Formatter(res2))
}

func TestDBOracleGorm_GetDicts2(t *testing.T) {
	var res2 []map[string]interface{}
	tx2 := d.
		Clauses(hints.New("PARALLEL(4)")).
		// Debug().
		Session(&gorm.Session{PrepareStmt: true}).
		Table("CM_DOCUMENT_TYPE").
		Select("CODE", "DESCRIPTION", "IS_SERIES", "IS_NUMBER").
		Order("code DESC")
	t.Log("To_SQL ---->", tx2.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Scan(&res2) }))

	c, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	c = ctxWithLog(c, log.Rec())

	t.Run("1", func(t *testing.T) {
		var rows []map[string]interface{}
		tx := tx2.Scan(&rows)
		help_log_map_res(t, tx, rows)
	})
	t.Run("2", func(t *testing.T) {
		t.Log(getDictTables(c, d.DB, "TM_CIM", ScopeDictTablesFilterByRegExp(RegExpDict_CustomerModel), ScopeDictTablesFilterByNumRows(20), ScopeExcludeBkpTmp))
	})

	t.Run("3", func(t *testing.T) {
		t.Log(getDictTables(c, d.DB, "TM_CIM"))
	})
	t.Run("4", func(t *testing.T) {
		for _, tbl := range getDictTables(c, d.DB, "TM_CIM", ScopeDictTablesFilterByRegExp(RegExpDict_CustomerModel), ScopeDictTablesFilterByNumRows(100), ScopeExcludeBkpTmp) {
			t.Log(tbl)
			var buf bytes.Buffer
			CSVTable(c, getTableContent(context.Background(), d.DB, tbl), &buf)
			d.l.Warnf("BUF: Available=%d Cap=%d Len=%d", buf.Available(), buf.Cap(), buf.Len())
			t.Log(buf.String())
			t.Log(strings.Repeat("-", 80))
		}
	})
	t.Run("5", func(t *testing.T) {
		var buf strings.Builder
		buf.WriteString("\n---\n# DICT tables:\n")

		for _, tbl := range d.GetDictTables(c, "TM_CIM") {
			var csvBuffer bytes.Buffer
			CSVTable(c, d.GetTableContent(c, tbl), &csvBuffer)
			buf.WriteString("\n## " + tbl + "\n" + csvBuffer.String())
		}
		t.Log(buf.String())
	})
}

func help_log_map_res(t *testing.T, transaction *gorm.DB, res []map[string]interface{}) {
	t.Helper()
	var dummy interface{}
	t.Log("To_SQL ---->", transaction.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Scan(&dummy) }))
	t.Logf("res: isNil=%t Len=%d", res == nil, len(res))
	t.Logf("res: %# v", pretty.Formatter(res))
}

func TestDBOracleGorm_PrepareDBInfoForLLM(t *testing.T) {
	files, _ := filepath.Glob("FULL_DATA*")
	for _, file := range files {
		t.Log("Remove file:", file)
		os.Remove(file)
	}
	ctx := context.Background()

	t.Run("1", func(t *testing.T) {
		d = NewDBOracleGorm(DSN(os.Getenv("DB_URL_TM_CIM")), Logger(log), DebugMode(false), DictsInclude(), DictsFilters(ScopeDictTablesFilterByRegExp(RegExpDict_CustomerModel), ScopeDictTablesFilterByNumRows(100), ScopeExcludeBkpTmp))
		d.SetFormatter(&DbFormatterClaude{})
		os.WriteFile("FULL_DATA_1.txt", []byte(d.PrepareDBInfoForLLM(ctx, regExpTblsCustomerModel)), 0644)
	})

	t.Run("2", func(t *testing.T) {
		d = NewDBOracleGorm(DSN(os.Getenv("DB_URL_TM_CIM")), Logger(log), DebugMode(true), DictsInclude(), DictsFilters(ScopeDictTablesFilterByRegExp(RegExpDict_CustomerModel), ScopeDictTablesFilterByNumRows(100), ScopeExcludeBkpTmp))
		d.SetFormatter(&DbFormatterOAI{})
		os.WriteFile("FULL_DATA_2.txt", []byte(d.PrepareDBInfoForLLM(ctx, regExpTblsCustomerModel)), 0644)
	})

	t.Run("3", func(t *testing.T) {
		d = NewDBOracleGorm(DSN(os.Getenv("DB_URL_TM_CIM")), Logger(log), DictsInclude())
		os.WriteFile("FULL_DATA_3.txt", []byte(d.PrepareDBInfoForLLM(ctx, regExpTblsCustomerModel)), 0644)
	})

	t.Run("4", func(t *testing.T) {
		d = NewDBOracleGorm(DSN(os.Getenv("DB_URL_TM_CIM")), Logger(log))
		os.WriteFile("FULL_DATA_4.txt", []byte(d.PrepareDBInfoForLLM(ctx, regExpTblsCustomerModel)), 0644)
	})
}
func TestDBOracleGorm_Test1(t *testing.T) {
	var buf strings.Builder
	t.Log("BUF ==>", buf)
	t.Logf("BUF=[%s] isEmpty=%t", buf.String(), buf.String() == "")
	t.Log(d.PrepareDBInfoForLLM(ctx))
}

func TestDBOracleGorm_Test2(t *testing.T) {
	r, e := RunSQLQuery(ctx, d.DB, "SELECT * FROM CM_ACCOUNT_ATTRIBUTE fetch first 10 rows only")
	assert.NoError(t, e)
	t.Log(r)
	t.Log(utils.JsonPrettyStr(r))
}

func RunSQLQuery(ctx context.Context, db *gorm.DB, query string) (res []map[string]interface{}, err error) {
	err = db.Debug().Raw(query).Scan(&res).Error
	return
}
