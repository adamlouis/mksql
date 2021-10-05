package server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/adamlouis/mksql/internal/sqliteutils"
	"github.com/gorilla/mux"
)

var (
	dbPageCache = map[string][]byte{} // todo: handle non-ro mode
	dbPageLock  sync.RWMutex
)

func (s *srv) HandleGetDBPage(w http.ResponseWriter, r *http.Request) {
	defer Recover(w)

	dbname := mux.Vars(r)["db"]
	cachekey := dbname

	dbPageLock.RLock()
	if b, ok := dbPageCache[cachekey]; ok {
		defer dbPageLock.RUnlock()
		w.Write(b)
		return
	}
	dbPageLock.RUnlock()

	dbx, err := s.getRODB(dbname)
	if err != nil {
		SendError(w, err)
		return
	}
	defer dbx.Close()

	schema, err := sqliteutils.GetSchema(dbx)
	if err != nil {
		SendError(w, err)
		return
	}

	tbls, err := sqliteutils.GetTables(dbx)
	if err != nil {
		SendError(w, err)
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

	buf := new(bytes.Buffer)
	err = t.Execute(buf, PageData{
		DBName:   dbname,
		DBSchema: dbschema,
		DefaultQ: dq,
	})
	if err != nil {
		SendError(w, err)
		return
	}

	executed, err := ioutil.ReadAll(buf)
	if err != nil {
		SendError(w, err)
		return
	}

	dbPageLock.Lock()
	dbPageCache[cachekey] = executed
	dbPageLock.Unlock()
	w.Write(executed)
}
