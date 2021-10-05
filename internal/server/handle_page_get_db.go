package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/adamlouis/mksql/internal/sqliteutils"
	"github.com/gorilla/mux"
)

func (s *srv) HandleGetDBPage(w http.ResponseWriter, r *http.Request) {
	defer Recover(w)

	dbname := mux.Vars(r)["db"]

	db, err := s.getRODB(dbname)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	schema, err := sqliteutils.GetSchema(db)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	tbls, err := sqliteutils.GetTables(db)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	dq := ""
	if len(tbls) > 0 {
		dq = fmt.Sprintf("SELECT * FROM %s LIMIT 10", tbls[0].Name)
	}

	ss := make([]string, len(schema))
	for i, s := range schema {
		ss[i] = s.SQL
	}
	dbschema := strings.Join(ss, "\n")

	t := newTemplate(
		"db.go.html",
		[]string{
			"templates/db.go.html",
			"templates/common.go.html",
		},
	)

	_ = t.Execute(w, PageData{
		DBName:   dbname,
		DBSchema: dbschema,
		DefaultQ: dq,
	})

}
