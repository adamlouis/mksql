package server

import (
	"database/sql"
	"embed"
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/adamlouis/mksql/internal/jsonlog"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"
)

//go:embed templates
var templatesFS embed.FS

//go:embed static
var staticFS embed.FS

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

var (
	//go:embed static/favicon.ico
	faviconBytes []byte
)

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

	sql.Register("sqlite3_with_limits", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			conn.SetLimit(sqlite3.SQLITE_LIMIT_ATTACHED, 0)
			conn.SetLimit(sqlite3.SQLITE_LIMIT_TRIGGER_DEPTH, 1)
			conn.SetLimit(sqlite3.SQLITE_LIMIT_COLUMN, 100)
			return nil
		},
	})

	// web home
	r.HandleFunc("/", s.HandleGetHomePage).Methods(http.MethodGet)
	// dbs
	r.HandleFunc("/dbs/{db}", s.HandleGetDBPage).Methods(http.MethodGet)
	r.HandleFunc("/dbs:create", s.HandleCreateDBPage).Methods(http.MethodGet)
	r.HandleFunc("/dbs:create", s.HandlePostUploadPage).Methods(http.MethodPost)

	// query
	r.HandleFunc("/q", s.HandleGetQ).Methods(http.MethodGet) // 1) consider "/dbs/{db}:q" ? 2) consider http.MethodGet

	// static
	r.HandleFunc("/favicon.ico", s.HandleFavicon).Methods(http.MethodGet)
	if os.Getenv("MKSQL_MODE") == "DEVELOPMENT" {
		r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./internal/server/static/"))))
	} else {
		// r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
		r.PathPrefix("/static/").Handler(http.FileServer(http.FS(staticFS)))
	}

	r.Use(loggerMiddleware)
	r.Use(getACLMiddleware()...)

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
func SendError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte("error: " + err.Error()))
}

func (s *srv) HandleFavicon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/x-icon")
	_, _ = w.Write(faviconBytes)
}

var dbnameRegex = regexp.MustCompile("^[a-zA-Z]+.db$")

// TODO: better connection pooling ... this is for read-only demo
func (s *srv) getRODB(dbname string) (*sqlx.DB, error) {
	if !dbnameRegex.Match([]byte(dbname)) {
		return nil, fmt.Errorf("invalid db name")
	}

	dataDir := filepath.Join(s.opts.DataDir, "dbs")
	conn := fmt.Sprintf("file:%s?mode=ro&_query_only=true", filepath.Join(dataDir, dbname))

	dbx, err := sqlx.Open("sqlite3_with_limits", conn)
	if err == nil {
		// dbs[conn] = dbx
	}

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

// in development, load templates from local filesystem
// in production, use embeded FS
//
// in development, this allows template files to update without re-building
// in production, this allows for a self-contained executable
func newTemplate(name string, patterns []string) *template.Template {
	if os.Getenv("MKSQL_MODE") == "DEVELOPMENT" {
		resolved := make([]string, len(patterns))
		for i := range patterns {
			resolved[i] = fmt.Sprintf("internal/server/%s", patterns[i])
		}
		return template.Must(template.New(name).ParseFiles(resolved...))
	}
	return template.Must(template.New(name).ParseFS(templatesFS, patterns...))
}
