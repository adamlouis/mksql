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

	files := r.MultipartForm.File["files"]

	msg := []string{}

	for _, fh := range files {
		err = s.processFH(fh)
		if err != nil {
			msg = append(msg, err.Error())
		} else {
			msg = append(msg, "OK")
		}
	}

	fmt.Println(msg)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *srv) processFH(fh *multipart.FileHeader) error {
	uploadsDir := filepath.Join(s.opts.DataDir, "uploads")
	if err := os.MkdirAll(uploadsDir, os.ModePerm); err != nil {
		return err
	}

	// save the uploaded file to disk
	dstDir, err := os.MkdirTemp(uploadsDir, "")
	if err != nil {
		return err
	}

	dstPath := filepath.Join(dstDir, fh.Filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	_, err = io.Copy(dst, src)
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

		base := filepath.Base(fh.Filename)
		ext := filepath.Ext(base)
		name := base[:len(base)-len(ext)] + ".db"
		sqliteDst := filepath.Join(dbsDir, name)

		err := tosql.NewCSVToSQLer(tosql.CSVToSQLOpts{Strict: false}).ToSQL(sqliteDst, dstPath)
		if err != nil {
			fmt.Println(err)
		}
	}()

	return nil
	// incomingdst := "./uploads/" + fh.Filename

	// fincomingsrc, err := fh.Open()
	// if err != nil {
	// 	return err
	// }
	// defer fincomingsrc.Close()

	// fincomingdst, err := os.OpenFile(incomingdst, os.O_WRONLY|os.O_CREATE, 0666)
	// if err != nil {
	// 	return err
	// }
	// defer fincomingdst.Close()

	// io.Copy(fincomingdst, fincomingsrc)

	// sqlitedst := "./dbs/" + fh.Filename
	// csvdst := "./csvs/" + fh.Filename

	// err = processSQLite(incomingdst, sqlitedst)
	// if err == nil {
	// 	fmt.Println("IS A SQLITE")
	// 	return nil
	// }

	// if processCSV(incomingdst, csvdst) == nil {
	// 	fmt.Println("IS A CSV")
	// 	return tosql.NewCSVToSQLer(tosql.CSVToSQLOpts{Strict: true}).ToSQL(csvdst, strings.ReplaceAll(sqlitedst, ".csv", ".db"))
	// }

	// os.Remove(incomingdst)
	// fmt.Println("IS NONE")
	// return nil
}
