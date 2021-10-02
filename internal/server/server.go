package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/adamlouis/mksql/internal/jsonlog"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
)

type Server interface {
	Serve() error
}

func NewServer(opts ServerOpts) Server {
	return &srv{
		opts: opts,
	}
}

type srv struct {
	opts ServerOpts
}

type ServerOpts struct {
	Port    int
	DataDir string
}

func (s *srv) Serve() error {
	r := mux.NewRouter()

	if err := os.MkdirAll(s.opts.DataDir, os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(s.opts.DataDir, "dbs"), os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(s.opts.DataDir, "uploads"), os.ModePerm); err != nil {
		return err
	}

	// web
	r.HandleFunc("/", s.HandleGetHomePage).Methods(http.MethodGet)
	r.HandleFunc("/dbs/{db}", s.HandleGetDBPage).Methods(http.MethodGet)
	r.HandleFunc("/dbs:create", s.HandleCreateDBPage).Methods(http.MethodGet)
	r.HandleFunc("/upload", s.HandlePostUploadPage).Methods(http.MethodPost)
	// api
	r.HandleFunc("/q", s.HandleGetQ).Methods(http.MethodGet)
	// static
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./internal/server/static/"))))

	r.Use(loggerMiddleware)
	jsonlog.Log("name", "SERVER_START", "port", s.opts.Port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.opts.Port), r)
}

type DB struct {
	Name string
	Size string
}

type HTMLData struct {
	Columns []string      `json:"columns"`
	Rows    []interface{} `json:"rows"`
}

type PageData struct {
	DBs      []DB
	DBName   string
	DBSchema string
	DefaultQ string
}

var (
	dbs = map[string]*sqlx.DB{}
)

func Recover(w http.ResponseWriter) {
	if r := recover(); r != nil {
		_, _ = w.Write([]byte(fmt.Sprintf("recovered from panic: %v", r)))
	}
}

// TODO: conn pooling
func getDB(conn string) (*sqlx.DB, error) {
	if cached, ok := dbs[conn]; ok {
		fmt.Println("cached db")
		return cached, nil
	}

	dbx, err := sqlx.Open("sqlite3", conn)
	// if err == nil {
	// 	dbs[conn] = dbx
	// }

	return dbx, err
}

func toByteSize(b int64) string {
	if b > 1e9 {
		return fmt.Sprintf("%.2f GB", float64(b)/1e9)
	}
	if b > 1e6 {
		return fmt.Sprintf("%d MB", b/1e6)
	}
	if b > 1e3 {
		return fmt.Sprintf("%d KB", b/1e3)
	}
	return fmt.Sprintf("%d B", b)
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
