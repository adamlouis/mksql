package server

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"github.com/adamlouis/mksql/internal/sqliteutils"
)

func (s *srv) HandleGetHomePage(w http.ResponseWriter, r *http.Request) {
	defer Recover(w)

	dataDir := filepath.Join(s.opts.DataDir, "dbs")

	d, _ := os.ReadDir(dataDir)

	dbs := []DB{}
	for _, fd := range d {
		if !fd.IsDir() {
			info, err := fd.Info()
			if err != nil {
				_, _ = w.Write([]byte(err.Error()))
				return
			}
			dbs = append(dbs, DB{Name: fd.Name(), Size: toByteSize(info.Size())})
		}
	}

	dq := ""
	if len(dbs) > 0 {
		conn := fmt.Sprintf("file:%s?mode=ro", filepath.Join(dataDir, dbs[0].Name))
		db, err := getDB(conn)
		if err != nil {
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		tbls, err := sqliteutils.GetTables(db)
		if err != nil {
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		if len(tbls) > 0 {
			dq = fmt.Sprintf("SELECT * FROM %s LIMIT 10", tbls[0].Name)
		}

	}

	t := template.Must(template.New("home.go.html").ParseFiles("internal/server/templates/home.go.html", "internal/server/templates/common.go.html"))

	_ = t.Execute(w, PageData{
		DBs:      dbs,
		DefaultQ: dq,
	})
}
