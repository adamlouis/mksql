package server

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/adamlouis/mksql/internal/jsonlog"
)

var fmtHTMLTemplate = template.Must(template.New("t").Parse(`<table><thead><tr>{{ range $c := .Columns }}<td>{{$c}}</td>{{ end}}</tr></thead><tbody>{{ range $r := .Rows }}<tr>{{ range $c := $r }}<td>{{$c}}</td>{{ end}}</tr>{{ end}}</tbody></table>`))

var (
	_limitResponseSize int = 2e6
	_limitQueryDuraton     = time.Second * 10
)

func (s *srv) HandleGetQ(wrw http.ResponseWriter, r *http.Request) {
	metw := NewMeteredResponseWriter(wrw, _limitResponseSize) // cap response size at 2 MB
	dbname := r.URL.Query().Get("db")
	q := r.URL.Query().Get("q")
	qfmt := r.URL.Query().Get("fmt")

	defer Recover(wrw)

	if strings.TrimSpace(q) == "" {
		SendError(wrw, fmt.Errorf("empty query"))
		return
	}

	jsonlog.Log("name", "QUERY", "db", dbname, "q", q, "fmt", qfmt)

	dbx, err := s.getRODB(dbname)
	if err != nil {
		SendError(wrw, err)
		return
	}

	timed, cancel := context.WithTimeout(r.Context(), _limitQueryDuraton)
	defer cancel()
	rows, err := dbx.QueryxContext(timed, q)

	if err != nil {
		SendError(wrw, err)
		return
	}

	col, err := rows.Columns()
	if err != nil {
		SendError(wrw, err)
		return
	}

	if len(col) == 0 {
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
				SendError(wrw, err)
				return
			}
			result.Rows = append(result.Rows, r)
		}
		if err := fmtHTMLTemplate.Execute(metw, result); err != nil {
			SendError(wrw, err)
			return
		}
	case "json/obj":
		result := []map[string]interface{}{}
		for rows.Next() {
			m := map[string]interface{}{}
			err := rows.MapScan(m)
			if err != nil {
				SendError(wrw, err)
				return
			}
			result = append(result, m)
		}

		b, err := json.Marshal(result)
		if err != nil {
			SendError(wrw, err)
			return
		}

		if _, err = metw.Write(b); err != nil {
			SendError(wrw, err)
			return
		}
	case "json/arr":
		result := [][]interface{}{}
		result = append(result, toI(col))

		for rows.Next() {
			r, err := rows.SliceScan()
			if err != nil {
				SendError(wrw, err)
				return
			}
			result = append(result, r)
		}

		b, err := json.Marshal(result)
		if err != nil {
			SendError(wrw, err)
			return
		}

		if _, err = metw.Write(b); err != nil {
			SendError(wrw, err)
			return
		}
	case "col":
		tbw := tabwriter.NewWriter(metw, 4, 4, 4, ' ', 0)
		_, err = fmt.Fprintln(tbw, strings.Join(col, "\t")+"\t")
		if err != nil {
			SendError(wrw, err)
			return
		}
		for rows.Next() {
			qr, err := rows.SliceScan()
			if err != nil {
				SendError(wrw, err)
				return
			}
			_, err = fmt.Fprintln(tbw, strings.Join(toS(qr), "\t")+"\t")
			if err != nil {
				SendError(wrw, err)
				return
			}
		}
		if err := tbw.Flush(); err != nil {
			SendError(wrw, err)
			return
		}
	case "csv":
		csvw := csv.NewWriter(metw)
		if err := csvw.Write(col); err != nil {
			SendError(wrw, err)
			return
		}
		for rows.Next() {
			r, err := rows.SliceScan()
			if err == nil {
				if err := csvw.Write(toS(r)); err != nil {
					SendError(wrw, err)
					return
				}
			}
		}
		csvw.Flush()
	default:
		SendError(wrw, fmt.Errorf("unrecognized response fmt %s", qfmt))
		return
	}

	if err := timed.Err(); err != nil {
		SendError(wrw, err)
		return
	}
	metw.Flush()
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

// MeteredResponseWriter errors if requested to write more than the provided limit
type MeteredResponseWriter interface {
	io.Writer
	http.Flusher
}

type meteredwriter struct {
	w      http.ResponseWriter
	buffer []byte
	end    int
	err    error
}

func NewMeteredResponseWriter(w http.ResponseWriter, limit int) MeteredResponseWriter {
	return &meteredwriter{
		w:      w,
		buffer: make([]byte, limit),
		end:    0,
		err:    nil,
	}
}

func (w *meteredwriter) Write(b []byte) (int, error) {
	// if we errored previously, keep sending it
	if w.err != nil {
		return 0, w.err
	}
	if w.end+len(b) > len(w.buffer) {
		w.err = fmt.Errorf("exceeded max response size of %s", toByteSize(int64(len(w.buffer))))
		return 0, w.err
	}
	copy(w.buffer[w.end:w.end+len(b)], b)
	w.end = w.end + len(b)
	return len(b), nil
}

func (w *meteredwriter) Flush() {
	// if we errored previously, send it on flush
	if w.err != nil {
		_, _ = w.w.Write([]byte(w.err.Error()))
		return
	}
	_, _ = w.w.Write(w.buffer[:w.end])
}
