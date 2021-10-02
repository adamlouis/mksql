package tosql

type ToSQLer interface {
	ToSQL(dst, src string) error
}

type DBDefinition struct {
	TableName         string
	ColumnDefinitions []*ColumnDefinition
	ExpectedRowCount  uint64
}

type ColumnDefinition struct {
	Name string
	Type string
}
