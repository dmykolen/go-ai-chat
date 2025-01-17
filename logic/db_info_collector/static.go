package dbinfocollector

const (
	nn = " NOT NULL"
)

const (
	RegExpDict               = ".*(DICT|TYPE).*"
	RegExpDict_CustomerModel = "CM_" + RegExpDict

	_ctxKeyLogger ctxKey = "logger"
)

const (
	getTableNamesQueryByRegExp = `
		SELECT TABLE_NAME
		FROM all_tables
		WHERE owner = 'TM_CIM' AND REGEXP_LIKE(TABLE_NAME, :pattern)
	`

	getTableNamesQuery = `
		SELECT TABLE_NAME
		FROM all_tables
		WHERE owner = 'TM_CIM' AND TABLE_NAME LIKE :pattern
	`
	getColumnsQuery = `
		SELECT COLUMN_NAME, DATA_TYPE, DATA_LENGTH, NULLABLE
		FROM all_tab_cols
		WHERE table_name = @tableName AND USER_GENERATED = 'YES'
	`
	getPrimaryKeysQuery = `
		SELECT cc.COLUMN_NAME, cc.POSITION
		FROM all_cons_columns cc
		JOIN all_constraints co ON cc.constraint_name = co.constraint_name
		WHERE cc.table_name = @tableName AND co.constraint_type = 'P'
	`
	getForeignKeysQuery = `
		SELECT cc.COLUMN_NAME, co.CONSTRAINT_NAME, pk.TABLE_NAME AS REFERENCED_TABLE_NAME, pk.COLUMN_NAME AS REFERENCED_COLUMN_NAME
		FROM all_cons_columns cc
		JOIN all_constraints co ON cc.constraint_name = co.constraint_name
		JOIN all_cons_columns pk ON co.r_constraint_name = pk.constraint_name
		WHERE cc.table_name = @tableName AND co.constraint_type = 'R'
	`
	_getUniqueConstraintsQuery = `
		SELECT cc.COLUMN_NAME, co.CONSTRAINT_NAME
		FROM all_cons_columns cc
		JOIN all_constraints co ON cc.constraint_name = co.constraint_name
		WHERE cc.table_name = @tableName AND co.constraint_type = 'U'
	`
	getUniqueConstraintsQuery = `
		SELECT co.CONSTRAINT_NAME, cc.COLUMN_NAME, cc.POSITION
		FROM all_cons_columns cc
		JOIN all_constraints co ON cc.constraint_name = co.constraint_name
		WHERE cc.table_name = @tableName AND co.constraint_type = 'U'
		ORDER BY co.CONSTRAINT_NAME, cc.POSITION
	`
	getCheckConstraintsQuery = `
		SELECT cc.COLUMN_NAME, co.SEARCH_CONDITION
		FROM all_cons_columns cc
		JOIN all_constraints co ON cc.constraint_name = co.constraint_name
		WHERE cc.table_name = @tableName AND co.constraint_type = 'C'
	`
)
