package server

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/adamlouis/mksql/internal/tosql"
)

func (s *srv) HandlePostUploadPage(w http.ResponseWriter, r *http.Request) {
	defer Recover(w)

	// ParseMultipartForm parses a request body as multipart/form-data
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	filename := r.Form.Get("name")
	filefmt := r.Form.Get("fmt")

	f, _, err := r.FormFile("file")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// files := r.MultipartForm.File["file"]

	// fmt.Println(files)

	msg := []string{}
	// for _, fh := range files {
	err = s.processFH(f, filename, filefmt)
	if err != nil {
		msg = append(msg, err.Error())
	} else {
		msg = append(msg, "OK")
	}
	// }

	fmt.Println(msg)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *srv) processFH(f multipart.File, filename, filefmt string) error {
	if filefmt != ".csv" {
		return fmt.Errorf("unsupported format %s", filefmt)
	}

	uploadsDir := filepath.Join(s.opts.DataDir, "uploads")
	if err := os.MkdirAll(uploadsDir, os.ModePerm); err != nil {
		return err
	}

	dstDir, err := os.MkdirTemp(uploadsDir, "")
	if err != nil {
		return err
	}

	dstPath := filepath.Join(dstDir, filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, f)
	if err != nil {
		return err
	}

	// TODO: queue

	defer func() {
		dbsDir := filepath.Join(s.opts.DataDir, "dbs")
		if err := os.MkdirAll(dbsDir, os.ModePerm); err != nil {
			fmt.Println(err)
			return
		}

		base := filepath.Base(filename)
		ext := filepath.Ext(base)
		name := base[:len(base)-len(ext)] + ".db"
		sqliteDst := filepath.Join(dbsDir, name)

		err := tosql.NewCSVToSQLer(tosql.CSVToSQLOpts{Strict: false}).ToSQL(sqliteDst, dstPath)
		if err != nil {
			fmt.Println(err)
		}
		os.RemoveAll(dstDir)
	}()

	return nil
}
