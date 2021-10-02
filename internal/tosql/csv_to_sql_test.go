package tosql

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/adamlouis/mksql/internal/sqliteutils"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func TestCSVToSQL(t *testing.T) {
	fds, err := os.ReadDir("testdata")
	if err != nil {
		t.Error(err)
	}

	for _, fd := range fds {
		if !fd.IsDir() && strings.HasSuffix(fd.Name(), ".csv") && !strings.HasSuffix(fd.Name(), ".types.csv") {
			t.Run(fd.Name(), func(t *testing.T) {
				tmp, err := os.CreateTemp(os.TempDir(), "*.db")
				if err != nil {
					t.Error(err)
				}

				src := "testdata/" + fd.Name()
				dst := tmp.Name()
				cts := NewCSVToSQLer(CSVToSQLOpts{Strict: true})
				err = cts.ToSQL(dst, src)

				expectErr := strings.HasSuffix(src, "_err.csv")

				if expectErr {
					if err == nil {
						t.Fatalf("expected an error for %s", fd.Name())
					}

					return
				}

				if err != nil {
					t.Fatal(err)
				}

				expectedTypes, err := getTypesFromCSV(src)
				if err != nil {
					t.Fatal(err)
				}

				db, err := sqlx.Open("sqlite3", dst)
				if err != nil {
					t.Fatal(err)
				}

				schema, err := sqliteutils.GetTable(db, "data")
				if err != nil {
					t.Fatal(err)
				}

				actualtypes, err := getTypesFromSQL(schema.SQL)
				if err != nil {
					t.Fatal(err)
				}

				if !sliceEq(expectedTypes, actualtypes) {
					t.Fatal(fmt.Errorf("types do not match: %v %v", expectedTypes, actualtypes))
				}
			})
		}
	}
}

func getTypesFromCSV(csvpath string) ([]string, error) {
	f, err := os.Open(strings.Replace(csvpath, ".csv", ".types.csv", 1))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	csvr := csv.NewReader(f)

	header, err := csvr.Read()
	if err != nil {
		return nil, err
	}

	return header, nil
}

func getTypesFromSQL(q string) ([]string, error) {
	begin := strings.Index(q, "(")
	if begin == -1 {
		return nil, fmt.Errorf("no open paren")
	}

	defs := strings.Split(q[begin+1:], ",\n")
	result := make([]string, len(defs))
	for i, k := range defs {
		parts := strings.Split(strings.TrimSpace(k), " ")
		if len(parts) < 2 {
			return nil, fmt.Errorf("expected >= 2 parts")
		}
		result[i] = parts[1]
	}
	return result, nil
}

func sliceEq(l1, l2 []string) bool {
	if len(l1) != len(l2) {
		return false
	}
	for i := range l1 {
		if l1[i] != l2[i] {
			return false
		}
	}
	return true
}
