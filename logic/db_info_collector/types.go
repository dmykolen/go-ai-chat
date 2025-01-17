package dbinfocollector

import (
	"context"

	"gorm.io/gorm"
)

type ctxKey string
type filter = func(*gorm.DB) *gorm.DB

type DBSchemaInfoProvider interface {
	Formatter
	PrepareDBInfoForLLM(ctx context.Context, tblPattern ...string) (res string)
	GetTablesInfoByPattern(ctx context.Context, tableNamePattern string) (tables []TableInfo, err error)
	GetTableInfo(ctx context.Context, tableName string) (tbl TableInfo, err error)
	SetFormatter(formatter Formatter)
	Debug() (tx *gorm.DB)
	G() *gorm.DB
}

type Formatter interface {
	FormatTableInfo(ctx context.Context, tableInfo TableInfo) string
	FormatTables(ctx context.Context, tables []TableInfo) string
}

type (
	ColumnInfo struct {
		ColumnName string `gorm:"column:COLUMN_NAME" json:"column_name"`
		DataType   string `gorm:"column:DATA_TYPE" json:"data_type"`
		DataLength int    `gorm:"column:DATA_LENGTH" json:"data_length"`
		Nullable   string `gorm:"column:NULLABLE" json:"nullable"`
	}
	PrimaryKey struct {
		ColumnName string `gorm:"column:COLUMN_NAME" json:"column_name"`
		Position   int    `gorm:"column:POSITION" json:"position"`
	}
	ForeignKey struct {
		ColumnName           string `gorm:"column:COLUMN_NAME" json:"column_name"`
		ReferencedTableName  string `gorm:"column:REFERENCED_TABLE_NAME" json:"referenced_table_name"`
		ReferencedColumnName string `gorm:"column:REFERENCED_COLUMN_NAME" json:"referenced_column_name"`
		ConstraintName       string `gorm:"column:CONSTRAINT_NAME" json:"constraint_name"`
	}
	UniqueConstraint struct {
		ConstraintName string `gorm:"column:CONSTRAINT_NAME" json:"constraint_name"`
		ColumnName     string `gorm:"column:COLUMN_NAME" json:"column_name"`
		Position       int    `gorm:"column:POSITION" json:"position"`
	}
	CheckConstraint struct {
		ColumnName      string `gorm:"column:COLUMN_NAME" json:"column_name"`
		SearchCondition string `gorm:"column:SEARCH_CONDITION" json:"search_condition"`
	}
	TableInfo struct {
		TableName         string             `gorm:"column:TABLE_NAME" json:"table_name"`
		NumRows           int64              `gorm:"column:NUM_ROWS" json:"num_rows"`
		Columns           []ColumnInfo       `gorm:"-" json:"columns"`
		PrimaryKeys       []PrimaryKey       `gorm:"-" json:"primary_keys,omitempty"`
		ForeignKeys       []ForeignKey       `gorm:"-" json:"foreign_keys,omitempty"`
		UniqueConstraints []UniqueConstraint `gorm:"-" json:"unique_constraints,omitempty"`
		CheckConstraints  []CheckConstraint  `gorm:"-" json:"check_constraints,omitempty"`
	}
)
