package dbinfocollector

import (
	"context"
	"fmt"
	"strings"

	"github.com/samber/lo"
)

// All formatters should implement Formatter interface
var (
	_ Formatter = (*DbFormatterClaude)(nil)
	_ Formatter = (*DbFormatterOAI)(nil)
)

type DbFormatterClaude struct{}

func (f *DbFormatterClaude) FormatTables(ctx context.Context, tables []TableInfo) string {
	var buf strings.Builder

	for _, t := range tables {
		buf.WriteString(f.FormatTableInfo(ctx, t))
		buf.WriteByte('\n')
	}
	return buf.String()
}

func (f *DbFormatterClaude) FormatTableInfo(ctx context.Context, tableInfo TableInfo) string {
	resolveNullable := func(nullable string) (res string) {
		if nullable == "N" {
			res = nn
		}
		return
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Table: %s (%d rows)\n", tableInfo.TableName, tableInfo.NumRows))
	sb.WriteString("Columns:\n")
	for _, column := range tableInfo.Columns {
		sb.WriteString(fmt.Sprintf(" - %s %s(%d)%s\n", column.ColumnName, column.DataType, column.DataLength, resolveNullable(column.Nullable)))
		// sb.WriteString(fmt.Sprintf(" - %s (%s(%d), %s)\n", column.ColumnName, column.DataType, column.DataLength, column.Nullable))
	}
	sb.WriteString("Constraints:\n")
	if len(tableInfo.PrimaryKeys) > 0 {
		sb.WriteString(" - Primary Keys:\n")
		for _, pk := range tableInfo.PrimaryKeys {
			sb.WriteString(fmt.Sprintf("   - %s (Position: %d)\n", pk.ColumnName, pk.Position))
		}
	}
	if len(tableInfo.ForeignKeys) > 0 {
		sb.WriteString(" - Foreign Keys:\n")
		for _, fk := range tableInfo.ForeignKeys {
			sb.WriteString(fmt.Sprintf("   - %s (Constraint: %s, Referenced Table: %s, Referenced Column: %s)\n", fk.ColumnName, fk.ConstraintName, fk.ReferencedTableName, fk.ReferencedColumnName))
		}
	}
	if len(tableInfo.UniqueConstraints) > 0 {
		sb.WriteString(" - Unique Constraints:\n")
		var currentConstraint string
		for _, uc := range tableInfo.UniqueConstraints {
			if uc.ConstraintName != currentConstraint {
				sb.WriteString(fmt.Sprintf("   - %s:\n", uc.ConstraintName))
				currentConstraint = uc.ConstraintName
			}
			sb.WriteString(fmt.Sprintf("     - %s (Position: %d)\n", uc.ColumnName, uc.Position))
		}
	}
	if len(tableInfo.CheckConstraints) > 0 {
		if _, ok := lo.Find(tableInfo.CheckConstraints, func(cc CheckConstraint) bool { return !strings.HasSuffix(cc.SearchCondition, nn) }); ok {
			sb.WriteString(" - Check Constraints:\n")
		}
		for _, cc := range tableInfo.CheckConstraints {
			if !strings.HasSuffix(cc.SearchCondition, nn) {
				sb.WriteString(fmt.Sprintf("   - %s (Condition: %s)\n", cc.ColumnName, cc.SearchCondition))
			}
		}
	}
	return sb.String()
}

type DbFormatterOAI struct{}

func (f *DbFormatterOAI) FormatTables(ctx context.Context, tables []TableInfo) string {
	var buf strings.Builder

	for _, t := range tables {
		buf.WriteString(f.FormatTableInfo(ctx, t))
		buf.WriteByte('\n')
	}
	return buf.String()
}

func (f *DbFormatterOAI) FormatTableInfo(ctx context.Context, tableInfo TableInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s (%d rows)\n", tableInfo.TableName, tableInfo.NumRows))
	sb.WriteString("Columns:\n")
	for _, column := range tableInfo.Columns {
		nullable := ""
		if column.Nullable == "N" {
			nullable = "NOT NULL"
		}
		sb.WriteString(fmt.Sprintf(" - %s %s(%d) %s\n", column.ColumnName, column.DataType, column.DataLength, nullable))
	}

	sb.WriteString("Constraints:\n")
	if len(tableInfo.PrimaryKeys) > 0 {
		sb.WriteString("PrimaryKey:\n")
		columnNames := make([]string, len(tableInfo.PrimaryKeys))
		for i, pk := range tableInfo.PrimaryKeys {
			columnNames[i] = pk.ColumnName
		}
		sb.WriteString(fmt.Sprintf(" - %s\n", strings.Join(columnNames, ", ")))
	}

	if len(tableInfo.ForeignKeys) > 0 {
		sb.WriteString("ForeignKeys:\n")
		for _, fk := range tableInfo.ForeignKeys {
			sb.WriteString(fmt.Sprintf(" - %s: %s -> %s.%s\n", fk.ConstraintName, fk.ColumnName, fk.ReferencedTableName, fk.ReferencedColumnName))
		}
	}

	if len(tableInfo.UniqueConstraints) > 0 {
		sb.WriteString("UniqueConstraints:\n")
		for _, uc := range tableInfo.UniqueConstraints {
			sb.WriteString(fmt.Sprintf(" - %s: %s\n", uc.ConstraintName, uc.ColumnName))
		}
	}

	if len(tableInfo.CheckConstraints) > 0 {
		sb.WriteString("CheckConstraints:\n")
		for _, cc := range tableInfo.CheckConstraints {
			sb.WriteString(fmt.Sprintf(" - %s: %s\n", cc.ColumnName, cc.SearchCondition))
		}
	}

	return sb.String()
}
