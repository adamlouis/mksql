package tosql

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
)

type JSONCatToSQLOpts struct {
	Strict bool
}

func NewJSONCatToSQLer(opts JSONCatToSQLOpts) ToSQLer {
	return &jsoncattosql{
		opts: opts,
	}
}

type jsoncattosql struct {
	opts JSONCatToSQLOpts
}

func (t *jsoncattosql) ToSQL(dst, src string) error {
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
	return t.insertJSONRows(src, db, def)
}

// rm dupe function
func (t *jsoncattosql) createDB(dst string) (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite3", fmt.Sprintf("file:%s", dst))
	if err != nil {
		return nil, err
	}
	if _, err := db.Conn(context.Background()); err != nil {
		return nil, err
	}
	return db, nil
}

// rm dupe function
func (t *jsoncattosql) createTable(db *sqlx.DB, def *DBDefinition) error {
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

func (t *jsoncattosql) insertJSONRows(src string, db *sqlx.DB, def *DBDefinition) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)

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

	i := uint64(0)
	pct := pctprnt{milestones: []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}}
	for {
		doc := map[string]interface{}{}
		err := dec.Decode(&doc)
		if err == io.EOF {
			// all done
			break
		}
		if err != nil {
			return err
		}

		args, err := jtoSQLiteArgs(doc, def.ColumnDefinitions)
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

// func (t *csvtosql) createDB(dst string) (*sqlx.DB, error) {
// 	db, err := sqlx.Open("sqlite3", fmt.Sprintf("file:%s", dst))
// 	if err != nil {
// 		return nil, err
// 	}
// 	if _, err := db.Conn(context.Background()); err != nil {
// 		return nil, err
// 	}
// 	return db, nil
// }

// func (t *csvtosql) createTable(db *sqlx.DB, def *DBDefinition) error {
// 	cs := make([]string, len(def.ColumnDefinitions))
// 	for i, cd := range def.ColumnDefinitions {
// 		cs[i] = "\t" + cd.Name + " " + cd.Type
// 		if t.opts.Strict {
// 			cs[i] = cs[i] + fmt.Sprintf(" CHECK(typeof(%s) = '%s' OR %s IS NULL)", cd.Name, strings.ToLower(cd.Type), cd.Name) // TODO: dangerous string fmt, review
// 		}
// 	}

// 	createTableQ := fmt.Sprintf(
// 		"CREATE TABLE %s(\n%s\n)",
// 		def.TableName,
// 		strings.Join(cs, ",\n"),
// 	) // TODO: dangerous string fmt, review

// 	fmt.Println(createTableQ)
// 	_, err := db.Exec(createTableQ)
// 	if err != nil {
// 		_ = db.Close()
// 		return err
// 	}

// 	return nil
// }

// func (t *csvtosql) insertCSVRows(src string, db *sqlx.DB, def *DBDefinition) error {
// 	f, err := os.Open(src)
// 	if err != nil {
// 		return err
// 	}
// 	defer f.Close()

// 	csvr := csv.NewReader(f)
// 	if t.opts.Comma != 0 {
// 		csvr.Comma = t.opts.Comma
// 	}

// 	coldefs := make([]string, len(def.ColumnDefinitions))
// 	placeholders := make([]string, len(def.ColumnDefinitions))
// 	for i, cd := range def.ColumnDefinitions {
// 		coldefs[i] = cd.Name
// 		placeholders[i] = "?"
// 	}

// 	insertQ := fmt.Sprintf(
// 		"INSERT INTO %s(%s) VALUES (%s)",
// 		def.TableName,
// 		strings.Join(coldefs, ","),
// 		strings.Join(placeholders, ","),
// 	) // TODO: dangerous string fmt, review

// 	tx, err := db.Begin()
// 	if err != nil {
// 		return err
// 	}
// 	smt, err := tx.Prepare(insertQ)
// 	if err != nil {
// 		return err
// 	}
// 	first := true

// 	i := uint64(0)
// 	pct := pctprnt{milestones: []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}}
// 	for {
// 		line, err := csvr.Read()
// 		if err != nil {
// 			if err != io.EOF {
// 				return err
// 			}
// 			break
// 		}

// 		if first {
// 			first = false
// 			continue
// 		}

// 		args, err := toSQLiteArgs(line, def.ColumnDefinitions)
// 		if err != nil {
// 			return err
// 		}

// 		_, err = smt.Exec(args...)
// 		if err != nil {
// 			return fmt.Errorf("error inserting args:\n%v\ninto columns:\n%v\nerror: %w", args, coldefs, err)
// 		}

// 		i++
// 		pct.Update("inserted: %.0f%%\n", float64(i)/float64(def.ExpectedRowCount)*100)
// 	}
// 	return tx.Commit()
// }

// type pctprnt struct {
// 	milestones []float64
// 	idx        int
// 	pct        float64
// }

// func (p *pctprnt) Update(s string, pct float64) {
// 	if p.idx >= len(p.milestones) {
// 		return
// 	}

// 	p.pct = pct

// 	for p.pct >= p.milestones[p.idx] {
// 		pct := p.milestones[p.idx]
// 		fmt.Printf(s, pct)
// 		p.idx = p.idx + 1
// 		if p.idx >= len(p.milestones) {
// 			return
// 		}
// 	}
// }

// var (
// 	sqlreg = regexp.MustCompile("[^0-9a-zA-Z_]")
// )

// func sanitize(s string) string {
// 	sout := string(sqlreg.ReplaceAll([]byte(s), []byte("_")))
// 	return sout
// }

func (t *jsoncattosql) getDefinition(src string) (*DBDefinition, error) {
	f, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)

	count := uint64(0)
	colnames := map[string]string{}
	for {
		doc := map[string]interface{}{}
		err := dec.Decode(&doc)
		if err == io.EOF {
			// all done
			break
		}
		count++
		if err != nil {
			return nil, err
		}
		for k, v := range doc {

			prevType, ok := colnames[k]
			if !ok || prevType == "" {
				prevType = "INTEGER"    // start everything as INTEGER
				colnames[k] = "INTEGER" // start everything as INTEGER
			}

			if prevType == "TEXT" {
				continue
			}

			if prevType == "INTEGER" {
				if _, err := itoSQLiteInteger(v); err != nil {
					prevType = "REAL"    // not an INTEGER .. downgrade to REAL
					colnames[k] = "REAL" // not an INTEGER .. downgrade to REAL
				}
			}

			if prevType == "REAL" {
				if _, err := itoSQLiteReal(v); err != nil {
					if k == "Latitude" {
						fmt.Println(err, k, v)
					}
					prevType = "TEXT"    // not a REAL .. downgrade to TEXT
					colnames[k] = "TEXT" // not a REAL .. downgrade to TEXT
				}
			}
		}
	}

	coldefs := make([]*ColumnDefinition, len(colnames))
	i := 0
	for k, v := range colnames {
		coldefs[i] = &ColumnDefinition{
			Name: k,
			Type: v,
		}
		i++
	}

	sort.Slice(coldefs, func(i, j int) bool {
		return coldefs[i].Name < coldefs[j].Name
	})

	return &DBDefinition{
		TableName:         "data",
		ColumnDefinitions: coldefs,
		ExpectedRowCount:  count,
	}, nil
}

// func toSQLiteArgs(s []string, coldefs []*ColumnDefinition) ([]interface{}, error) {
// 	ifcs := make([]interface{}, len(s))
// 	for i := range s {
// 		switch t := coldefs[i].Type; t {
// 		case "INTEGER":
// 			v, err := toSQLiteInteger(s[i])
// 			if err != nil {
// 				return nil, err
// 			}
// 			ifcs[i] = v
// 		case "REAL":
// 			v, err := toSQLiteReal(s[i])
// 			if err != nil {
// 				return nil, err
// 			}
// 			ifcs[i] = v
// 		case "TEXT":
// 			ifcs[i] = toSQLiteText(s[i])
// 		default:
// 			return nil, fmt.Errorf("unexpected column type: %s", t)
// 		}
// 	}
// 	return ifcs, nil
// }

func jtoSQLiteArgs(doc map[string]interface{}, coldefs []*ColumnDefinition) ([]interface{}, error) {
	ifcs := make([]interface{}, len(coldefs))
	for i, cd := range coldefs {
		switch cd.Type {
		case "INTEGER":
			v, err := itoSQLiteInteger(doc[cd.Name])
			if err != nil {
				return nil, err
			}
			ifcs[i] = v
		case "REAL":
			v, err := itoSQLiteReal(doc[cd.Name])
			if err != nil {
				return nil, err
			}
			ifcs[i] = v
		case "TEXT":
			ifcs[i] = itoSQLiteText(doc[cd.Name])
		default:
			return nil, fmt.Errorf("unexpected column type: %s", cd.Type)
		}
	}
	return ifcs, nil
}
func itoSQLiteInteger(i interface{}) (interface{}, error) {
	if i == nil {
		return nil, nil
	}
	return toSQLiteInteger(fmt.Sprintf("%v", i))
}
func itoSQLiteReal(i interface{}) (interface{}, error) {
	if i == nil {
		return nil, nil
	}
	return toSQLiteReal(fmt.Sprintf("%v", i))
}
func itoSQLiteText(i interface{}) interface{} {
	if i == nil {
		return nil
	}

	switch v := i.(type) {
	case []interface{}:
	case map[string]interface{}:
		b, _ := json.Marshal(v)
		return string(b)
	}
	return fmt.Sprintf("%v", i)
}
