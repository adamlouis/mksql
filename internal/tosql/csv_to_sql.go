package tosql

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type CSVToSQLOpts struct {
	Strict bool
	Comma  rune
}

func NewCSVToSQLer(opts CSVToSQLOpts) ToSQLer {
	return &csvtosql{
		opts: opts,
	}
}

type csvtosql struct {
	opts CSVToSQLOpts
}

func (t *csvtosql) ToSQL(dst, src string) error {
	if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		return err
	}
	// get table definitions
	fmt.Println("1) planning table schema...")
	def, err := t.getDefinition(src)
	if err != nil {
		return err
	}

	// create db
	fmt.Println("2) creating database...")
	db, err := t.createDB(dst)
	if err != nil {
		return err
	}
	defer db.Close()

	// create table
	fmt.Println("3) creating table...")
	if err := t.createTable(db, def); err != nil {
		return err
	}

	// insert rows
	fmt.Println("4) inserting rows...")
	return t.insertCSVRows(src, db, def)
}

func (t *csvtosql) createDB(dst string) (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite3", fmt.Sprintf("file:%s", dst))
	if err != nil {
		return nil, err
	}
	if _, err := db.Conn(context.Background()); err != nil {
		return nil, err
	}
	return db, nil
}

func (t *csvtosql) createTable(db *sqlx.DB, def *DBDefinition) error {
	cs := make([]string, len(def.ColumnDefinitions))
	for i, cd := range def.ColumnDefinitions {
		cs[i] = "\t" + cd.Name + " " + cd.Type
		if t.opts.Strict {
			cs[i] = cs[i] + fmt.Sprintf(" CHECK(typeof(%s) = '%s' OR %s IS NULL)", cd.Name, strings.ToLower(cd.Type), cd.Name) // TODO: dangerous string fmt, review
		}
	}

	createTableQ := fmt.Sprintf(
		"CREATE TABLE %s(\n%s\n)",
		def.TableName,
		strings.Join(cs, ",\n"),
	) // TODO: dangerous string fmt, review

	fmt.Println(createTableQ)
	_, err := db.Exec(createTableQ)
	if err != nil {
		_ = db.Close()
		return err
	}

	return nil
}

func (t *csvtosql) insertCSVRows(src string, db *sqlx.DB, def *DBDefinition) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	csvr := csv.NewReader(f)
	if t.opts.Comma != 0 {
		csvr.Comma = t.opts.Comma
	}

	coldefs := make([]string, len(def.ColumnDefinitions))
	placeholders := make([]string, len(def.ColumnDefinitions))
	for i, cd := range def.ColumnDefinitions {
		coldefs[i] = cd.Name
		placeholders[i] = "?"
	}

	insertQ := fmt.Sprintf(
		"INSERT INTO %s(%s) VALUES (%s)",
		def.TableName,
		strings.Join(coldefs, ","),
		strings.Join(placeholders, ","),
	) // TODO: dangerous string fmt, review

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	smt, err := tx.Prepare(insertQ)
	if err != nil {
		return err
	}
	first := true

	i := uint64(0)
	pct := pctprnt{milestones: []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}}
	for {
		line, err := csvr.Read()
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		if first {
			first = false
			continue
		}

		args, err := toSQLiteArgs(line, def.ColumnDefinitions)
		if err != nil {
			return err
		}

		_, err = smt.Exec(args...)
		if err != nil {
			return fmt.Errorf("error inserting args:\n%v\ninto columns:\n%v\nerror: %w", args, coldefs, err)
		}

		i++
		pct.Update("inserted: %.0f%%\n", float64(i)/float64(def.ExpectedRowCount)*100)
	}
	return tx.Commit()
}

type pctprnt struct {
	milestones []float64
	idx        int
	pct        float64
}

func (p *pctprnt) Update(s string, pct float64) {
	if p.idx >= len(p.milestones) {
		return
	}

	p.pct = pct

	for p.pct >= p.milestones[p.idx] {
		pct := p.milestones[p.idx]
		fmt.Printf(s, pct)
		p.idx = p.idx + 1
		if p.idx >= len(p.milestones) {
			return
		}
	}
}

var (
	sqlreg = regexp.MustCompile("[^0-9a-zA-Z_]")
)

func sanitize(s string) string {
	sout := string(sqlreg.ReplaceAll([]byte(s), []byte("_")))
	return sout
}

func (t *csvtosql) getDefinition(src string) (*DBDefinition, error) {
	f, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var coldefs []*ColumnDefinition
	csvr := csv.NewReader(f)
	if t.opts.Comma != 0 {
		csvr.Comma = t.opts.Comma
	}

	first := true
	i := uint64(0)
	count := uint64(0)
	for {
		line, err := csvr.Read()
		if err != nil {
			break
		}

		i++
		count++
		if i == 500000 {
			p := message.NewPrinter(language.English)
			p.Printf("processed %d rows\n", count)
			i = 0
		}

		if first {
			coldefs = make([]*ColumnDefinition, len(line))
			for i, v := range line {
				coldefs[i] = &ColumnDefinition{
					Name: sanitize(v), // TODO: review
					Type: "INTEGER",   // start as an INTEGER, then downgrade
				}
			}
			first = false
		} else {
			if len(coldefs) != len(line) {
				return nil, fmt.Errorf("head has %d columns but row has %d", len(coldefs), len(line))
			}

			for i, cd := range coldefs {
				if cd.Type == "TEXT" {
					continue
				}

				if cd.Type == "INTEGER" {
					if _, err := toSQLiteInteger(line[i]); err != nil {
						cd.Type = "REAL" // not an INTEGER .. downgrade to REAL
					}
				}

				if cd.Type == "REAL" {
					if _, err := toSQLiteReal(line[i]); err != nil {
						cd.Type = "TEXT" // not a REAL .. downgrade to TEXT
					}
				}
			}
		}
	}

	return &DBDefinition{
		TableName:         "data",
		ColumnDefinitions: coldefs,
		ExpectedRowCount:  count - 1, // -1 for the column headers
	}, nil
}

func toSQLiteArgs(s []string, coldefs []*ColumnDefinition) ([]interface{}, error) {
	ifcs := make([]interface{}, len(s))
	for i := range s {
		switch t := coldefs[i].Type; t {
		case "INTEGER":
			v, err := toSQLiteInteger(s[i])
			if err != nil {
				return nil, err
			}
			ifcs[i] = v
		case "REAL":
			v, err := toSQLiteReal(s[i])
			if err != nil {
				return nil, err
			}
			ifcs[i] = v
		case "TEXT":
			ifcs[i] = toSQLiteText(s[i])
		default:
			return nil, fmt.Errorf("unexpected column type: %s", t)
		}
	}
	return ifcs, nil
}
func toSQLiteInteger(v string) (interface{}, error) {
	if v == "" {
		return nil, nil
	}
	if strings.ToLower(v) == "true" {
		return true, nil
	}
	if strings.ToLower(v) == "false" {
		return false, nil
	}
	if i, err := strconv.Atoi(v); err == nil {
		return i, nil
	}
	return nil, fmt.Errorf("%s is not a sqlite int", v)
}
func toSQLiteReal(v string) (interface{}, error) {
	if v == "" {
		return nil, nil
	}
	if strings.ToLower(v) == "true" {
		return true, nil
	}
	if strings.ToLower(v) == "false" {
		return false, nil
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f, nil
	}
	return nil, fmt.Errorf("%s is not a sqlite real", v)
}
func toSQLiteText(v string) interface{} {
	if v == "" {
		return nil
	}
	return v
}
