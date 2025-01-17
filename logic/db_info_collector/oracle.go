package dbinfocollector

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	oracle "github.com/godoes/gorm-oracle"
	_ "github.com/godror/godror"
	"github.com/samber/lo"
	gl "gitlab.dev.ict/golang/libs/gologgers"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DBOracleGorm implements DBSchemaInfoProvider interface
type DBOracleGorm struct {
	*gorm.DB
	isDebug       bool
	l             *gl.Logger
	dsn           string
	fmtr          Formatter
	dictsInclude  bool
	dictsFilters  []filter
	defTblPattern string
}

var _ DBSchemaInfoProvider = (*DBOracleGorm)(nil)

// OptFunc is a function that configures the DBOracleGorm instance.
type OptFunc func(*DBOracleGorm)

func DSN(dsn string) OptFunc           { return func(d *DBOracleGorm) { d.dsn = dsn } }
func DebugMode(isDebug bool) OptFunc   { return func(d *DBOracleGorm) { d.isDebug = isDebug } }
func Logger(l *gl.Logger) OptFunc      { return func(d *DBOracleGorm) { d.l = l } }
func DictsInclude() OptFunc            { return func(d *DBOracleGorm) { d.dictsInclude = true } }
func DictsFilters(f ...filter) OptFunc { return func(d *DBOracleGorm) { d.dictsFilters = f } }
func DefaultTbl(regExp string) OptFunc { return func(d *DBOracleGorm) { d.defTblPattern = regExp } }

func defaultDBOracleGorm() *DBOracleGorm {
	return &DBOracleGorm{
		isDebug:      false,
		fmtr:         &DbFormatterClaude{},
		dictsInclude: false,
		dictsFilters: []filter{
			ScopeDictTablesFilterByRegExp(RegExpDict),
			ScopeDictTablesFilterByNumRows(100),
		},
	}
}

// NewDBOracleGorm creates a new instance of DBOracleGorm with the given DSN and optional configurations.
func NewDBOracleGorm(opts ...OptFunc) *DBOracleGorm {
	d := defaultDBOracleGorm()
	for _, opt := range opts {
		opt(d)
	}
	if d.dsn == "" || d.l == nil {
		panic("dsn or logger is empty")
	}
	r := d.l.Rec("DB")
	r.Infof("Init DBPool with log_mode=%d", lo.Ternary(d.isDebug, logger.Info, logger.Error))
	db, err := gorm.Open(Open(d.dsn), &gorm.Config{
		Logger: logger.New(r, logger.Config{SlowThreshold: 2 * time.Second, ParameterizedQueries: true, Colorful: false, LogLevel: lo.Ternary(d.isDebug, logger.Info, logger.Error)}),
		// PrepareStmt: true, // cache prepared statements
	})
	if err != nil {
		r.Panicf("gorm.Open - panic! Error: %v", err)
	}

	d.DB = db
	return d
}

func (d *DBOracleGorm) PrepareDBInfoForLLM(ctx context.Context, tblPattern ...string) (res string) {
	rec := d.l.RecWithCtx(ctx, "DB")
	ctx = ctxWithLog(ctx, rec)
	rec.Info("START prepare DB info for LLM")
	defer rec.Info("END prepare DB info for LLM")

	tables, err := d.GetTablesInfoByPattern(ctx, lo.Ternary(len(tblPattern) > 0, tblPattern, []string{d.defTblPattern})[0])
	if err != nil {
		rec.Error("GetTablesInfoByPattern", err)
		return ""
	}
	res = d.FormatTables(ctx, tables)

	rec.Info("Include DICT tables =", d.dictsInclude)
	if d.dictsInclude {
		var buf strings.Builder
		buf.WriteString("\n---\n# DICT tables:\n")

		for _, tbl := range d.GetDictTables(ctx, "TM_CIM") {
			var csvBuffer bytes.Buffer
			CSVTable(ctx, d.GetTableContent(ctx, tbl), &csvBuffer)
			buf.WriteString("\n## " + tbl + "\n" + csvBuffer.String())
		}
		res += buf.String()
	}
	return
}

func (d *DBOracleGorm) SetFormatter(formatter Formatter) {
	d.fmtr = formatter
}

func (d *DBOracleGorm) FormatTableInfo(ctx context.Context, tableInfo TableInfo) string {
	if d.fmtr == nil {
		panic("formatter is not set")
	}
	return d.fmtr.FormatTableInfo(ctx, tableInfo)
}

func (d *DBOracleGorm) FormatTables(ctx context.Context, tables []TableInfo) string {
	if d.fmtr == nil {
		panic("formatter is not set")
	}
	return d.fmtr.FormatTables(ctx, tables)
}

func (d *DBOracleGorm) GetTablesInfoByPattern(ctx context.Context, tableNamePattern string) (tables []TableInfo, err error) {
	t := time.Now()
	log := d.l.RecWithCtx(ctx, "DB")
	log.Infof("START collect info for tables by pattern --> [%s]", tableNamePattern)
	defer func() {
		log.WithData(gl.M{"elapsed": time.Since(t).Milliseconds()}).Infof("END collect info for tables by pattern --> [%s] Tables count = %d", tableNamePattern, len(tables))
	}()

	var tableNames []string
	err = d.Raw(getTableNamesQueryByRegExp, tableNamePattern).Pluck("TABLE_NAME", &tableNames).Error
	if err != nil {
		log.Errorf("failed to get table names: %v", err)
		return
	}
	log.Debugf("Tables => %v", tableNames)

	for _, tableName := range tableNames {
		table, err := d.GetTableInfo(ctx, tableName)
		if err != nil {
			log.Errorf("failed to get table info for table %s: %v", tableName, err)
			continue
		}
		tables = append(tables, table)
	}
	return
}

func (d *DBOracleGorm) GetTableInfo(ctx context.Context, tableName string) (tbl TableInfo, err error) {
	log := d.l.RecWithCtx(ctx, "DB")
	log.Debug("Processing table --> ", tableName)
	tbl.TableName = tableName

	err = d.Raw(getColumnsQuery, map[string]interface{}{"tableName": tableName}).Scan(&tbl.Columns).Error
	if err != nil {
		log.Errorf("failed to get columns for table %s: %v", tableName, err)
		return
	}

	err = d.Raw(getPrimaryKeysQuery, map[string]interface{}{"tableName": tableName}).Scan(&tbl.PrimaryKeys).Error
	if err != nil {
		log.Errorf("failed to get primary keys for table %s: %v", tableName, err)
		return
	}

	err = d.Raw(getForeignKeysQuery, map[string]interface{}{"tableName": tableName}).Scan(&tbl.ForeignKeys).Error
	if err != nil {
		log.Errorf("failed to get foreign keys for table %s: %v", tableName, err)
		return
	}

	err = d.Raw(getUniqueConstraintsQuery, map[string]interface{}{"tableName": tableName}).Scan(&tbl.UniqueConstraints).Error
	if err != nil {
		log.Errorf("failed to get unique constraints for table %s: %v", tableName, err)
		return
	}

	err = d.Raw(getCheckConstraintsQuery, map[string]interface{}{"tableName": tableName}).Scan(&tbl.CheckConstraints).Error
	if err != nil {
		log.Errorf("failed to get check constraints for table %s: %v", tableName, err)
		return
	}

	err = d.Table("all_tables").
		Where("TABLE_NAME = ?", tableName).
		Select("NUM_ROWS").
		Scan(&tbl).Error
	if err != nil {
		log.Errorf("failed to get row count for table %s: %v", tableName, err)
		return
	}
	return
}

func (d *DBOracleGorm) GetDictTables(ctx context.Context, owner string, scopes ...func(*gorm.DB) *gorm.DB) (res []string) {
	return getDictTables(ctx, d.DB, owner, d.dictsFilters...)
}

func (d *DBOracleGorm) GetTableContent(ctx context.Context, tblName string) (res []map[string]interface{}) {
	return getTableContent(ctx, d.DB, tblName)
}

func (d *DBOracleGorm) G() *gorm.DB {
	return d.DB
}

func ScopeDictTablesFilterByRegExp(regexp string) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		return tx.Where("REGEXP_LIKE(table_name,?)", regexp)
	}
}
func ScopeExcludeBkpTmp(tx *gorm.DB) *gorm.DB {
	return tx.Where("not REGEXP_LIKE(table_name,'.*(BKP|TMP).*')")
}
func ScopeDictTablesFilterByNumRows(maxRowsInTbl int) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		return tx.Where("num_rows <= ?", maxRowsInTbl)
	}
}

func getDictTables(ctx context.Context, tx *gorm.DB, owner string, scopes ...func(*gorm.DB) *gorm.DB) (res []string) {
	log := logrec(ctx)
	if scopes == nil {
		scopes = []func(*gorm.DB) *gorm.DB{ScopeDictTablesFilterByRegExp(RegExpDict)}
	}
	tx1 := tx.
		Table("all_tables").
		Select("TABLE_NAME").
		Where("OWNER = ?", owner).
		Scopes(scopes...)
	log.Infof("SQL_QUERY: %s", tx1.WithContext(ctx).ToSQL(func(tx *gorm.DB) *gorm.DB { var v interface{}; return tx.Scan(&v) }))
	tx1.WithContext(ctx).Scan(&res)
	log.Infof("Collect dict tables size = %d", len(res))
	return
}

func getTableContent(ctx context.Context, tx *gorm.DB, tblName string) (res []map[string]interface{}) {
	logrec(ctx).Debug("Processing DICT table --> ", tblName)
	tx1 := tx.WithContext(ctx).Table(tblName).Scan(&res)
	if tx1.Error != nil {
		logrec(ctx).Error("Error", tx1.Error)
	}
	return res
}

func CSVTable(ctx context.Context, records []map[string]interface{}, w io.Writer) error {
	writer := csv.NewWriter(w)
	writer.Comma = ';'
	writer.UseCRLF = true
	defer writer.Flush()

	if len(records) > 0 {
		headers := lo.Keys(records[0])
		if err := writer.Write(headers); err != nil {
			return err
		}
		for _, record := range records {
			var row []string
			for _, header := range headers {
				row = append(row, toString(record[header]))
			}
			if err := writer.Write(row); err != nil {
				return err
			}
		}
	}
	return nil
}

func toString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return ""
	}
}

func logrec(ctx context.Context) *gl.LogRec {
	logger, ok := ctx.Value(_ctxKeyLogger).(*gl.LogRec)
	if !ok {
		panic("logger not found in context! Use ctxWithLogger(ctx, logger) to set logger in context under key _ctxKeyLogger")
	}
	return logger
}

func ctxWithLog(ctx context.Context, l *gl.LogRec) context.Context {
	return context.WithValue(ctx, _ctxKeyLogger, l)
}

func Open(dsn string) gorm.Dialector {
	conn, err := sql.Open("godror", dsn)
	if err != nil {
		panic(fmt.Errorf("sql.Open(godror, dsn) - PANIC! Error: %v", err))
	}
	conn.SetConnMaxLifetime(time.Minute * 1)
	conn.SetMaxOpenConns(30)
	conn.SetMaxIdleConns(30)

	Ping(context.Background(), conn)

	return oracle.New(oracle.Config{
		DriverName: "godror",
		Conn:       conn,
	})
}

func Ping(ctx context.Context, pool *sql.DB) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.PingContext(ctx); err != nil {
		panic(fmt.Errorf("unable to connect to database: %v", err))
	}
}
