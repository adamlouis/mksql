package server

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

func (s *srv) HandleGetQ(w http.ResponseWriter, r *http.Request) {
	dbname := r.URL.Query().Get("db")
	q := r.URL.Query().Get("q")
	qfmt := r.URL.Query().Get("fmt")

	defer Recover(w)

	conn := fmt.Sprintf("file:%s?mode=ro", filepath.Join(s.opts.DataDir, "dbs", dbname))

	dbx, err := getDB(conn)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	rows, err := dbx.Queryx(q)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	col, err := rows.Columns()
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	switch qfmt {
	case "html":

		result := &HTMLData{
			Columns: col,
		}

		for rows.Next() {
			r, err := rows.SliceScan()
			if err != nil {
				_, _ = w.Write([]byte(err.Error()))
				return
			}
			result.Rows = append(result.Rows, r)
		}

		template := template.Must(template.New("t").Parse(`<table><thead><tr>{{ range $c := .Columns }}<td>{{$c}}</td>{{ end}}</tr></thead><tbody>{{ range $r := .Rows }}<tr>{{ range $c := $r }}<td>{{$c}}</td>{{ end}}</tr>{{ end}}</tbody></table>`))
		_ = template.Execute(w, result)

	case "json/obj":
		result := []map[string]interface{}{}
		for rows.Next() {
			m := map[string]interface{}{}
			err := rows.MapScan(m)
			if err != nil {
				_, _ = w.Write([]byte(err.Error()))
				return
			}
			result = append(result, m)
		}

		b, err := json.Marshal(result)
		if err != nil {
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		_, _ = w.Write(b)
	case "json/arr":
		result := [][]interface{}{}
		result = append(result, toI(col))

		for rows.Next() {
			r, err := rows.SliceScan()
			if err != nil {
				_, _ = w.Write([]byte(err.Error()))
				return
			}
			result = append(result, r)
		}

		b, err := json.Marshal(result)
		if err != nil {
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		_, _ = w.Write(b)
	case "col":
		tbw := tabwriter.NewWriter(w, 4, 4, 4, ' ', 0)
		fmt.Fprintln(tbw, strings.Join(col, "\t")+"\t")
		for rows.Next() {
			r, err := rows.SliceScan()
			if err != nil {
				_, _ = w.Write([]byte(err.Error()))
				return
			}
			fmt.Fprintln(tbw, strings.Join(toS(r), "\t")+"\t")
		}
		tbw.Flush()
	case "csv":
		csvw := csv.NewWriter(w)
		if err := csvw.Write(col); err != nil {
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		for rows.Next() {
			r, err := rows.SliceScan()
			if err == nil {
				if err := csvw.Write(toS(r)); err != nil {
					_, _ = w.Write([]byte(err.Error()))
					return
				}
			}
		}
		csvw.Flush()
	default:

	}
}

func toS(ifcs []interface{}) []string {
	s := make([]string, len(ifcs))
	for i, ifc := range ifcs {
		s[i] = fmt.Sprintf("%v", ifc)
	}
	return s
}

func toI(s []string) []interface{} {
	ifcs := make([]interface{}, len(s))
	for i := range s {
		ifcs[i] = s[i]
	}
	return ifcs
}

func (s *srv) HandlePostQ(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.New("home").Parse("<html><body>hello</body></html>"))
	_ = t.Execute(w, struct{}{})
}
