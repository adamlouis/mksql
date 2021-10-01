package server

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/adamlouis/mksql/internal/jsonlog"
	"github.com/gorilla/mux"
)

type ServerOpts struct {
	Port int
}

func Serve(opts ServerOpts) error {
	r := mux.NewRouter()
	r.HandleFunc("/", HandleHome)
	r.Use(loggerMiddleware)
	jsonlog.Log("name", "SERVER_START", "port", opts.Port)
	return http.ListenAndServe(fmt.Sprintf(":%d", opts.Port), r)
}

func HandleHome(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.New("home").Parse("<html><body>hello</body></html>"))
	_ = t.Execute(w, struct{}{})
}

func loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		next.ServeHTTP(w, r)
		jsonlog.Log(
			"name", "REQUEST",
			"method", r.Method,
			"duration_ms", time.Since(now)/time.Millisecond,
			"path", r.URL.Path,
			"time", time.Now().Format(time.RFC3339),
		)
	})
}
