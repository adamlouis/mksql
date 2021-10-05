package server

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
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

	t := template.Must(template.New("home.go.html").ParseFiles("internal/server/templates/home.go.html", "internal/server/templates/common.go.html"))

	_ = t.Execute(w, PageData{
		DBs: dbs,
	})
}
