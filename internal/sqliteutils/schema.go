package sqliteutils

import (
	"github.com/jmoiron/sqlx"
)

type Schema struct {
	Type     string `db:"type"`
	Name     string `db:"name"`
	TblName  string `db:"tbl_name"`
	RootPage int64  `db:"rootpage"`
	SQL      string `db:"sql"`
}

func GetSchema(db *sqlx.DB) ([]Schema, error) {
	result := []Schema{}
	if err := db.Select(&result, "SELECT * from sqlite_master ORDER BY rootpage"); err != nil {
		return nil, err
	}
	return result, nil
}

func GetTables(db *sqlx.DB) ([]Schema, error) {
	result := []Schema{}
	if err := db.Select(&result, "SELECT * from sqlite_master WHERE type = 'table' ORDER BY rootpage"); err != nil {
		return nil, err
	}
	return result, nil
}

func GetTable(db *sqlx.DB, name string) (*Schema, error) {
	row := db.QueryRowx("SELECT * from sqlite_master WHERE type = 'table' AND name = ?", name)
	result := &Schema{}
	if err := row.StructScan(result); err != nil {
		return nil, err
	}
	return result, nil
}
