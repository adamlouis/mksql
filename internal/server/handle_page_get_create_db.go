package server

import (
	"html/template"
	"net/http"
)

func (s *srv) HandleCreateDBPage(w http.ResponseWriter, r *http.Request) {
	defer Recover(w)

	// dbname := mux.Vars(r)["db"]
	// conn := fmt.Sprintf("file:%s?mode=ro", filepath.Join(s.opts.DataDir, "dbs", dbname))

	// fmt.Println(conn)

	// db, err := getDB(conn)
	// if err != nil {
	// 	_, _ = w.Write([]byte(err.Error()))
	// 	return
	// }

	// schema, err := sqliteutils.GetSchema(db)
	// if err != nil {
	// 	_, _ = w.Write([]byte(err.Error()))
	// 	return
	// }

	// tbls, err := sqliteutils.GetTables(db)
	// if err != nil {
	// 	_, _ = w.Write([]byte(err.Error()))
	// 	return
	// }
	// dq := ""
	// if len(tbls) > 0 {
	// 	dq = fmt.Sprintf("SELECT * FROM %s LIMIT 10", tbls[0].Name)
	// }

	// ss := make([]string, len(schema))
	// for i, s := range schema {
	// 	ss[i] = s.SQL
	// }
	// dbschema := strings.Join(ss, "\n")

	t := template.Must(template.New("dbs-create.go.html").ParseFiles(
		"internal/server/templates/dbs-create.go.html",
		"internal/server/templates/common.go.html",
	))

	_ = t.Execute(w, PageData{})
}
