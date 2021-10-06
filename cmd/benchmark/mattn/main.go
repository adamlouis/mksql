package main

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if len(os.Args) != 2 {
		panic("unexpected args")
	}
	start := time.Now()
	n := 1
	m := 3
	var wg sync.WaitGroup
	wg.Add(n * m)
	for i := 0; i < m; i++ {
		go runN(n, wg.Done)
	}
	wg.Wait()
	fmt.Println(time.Since(start))
}

func runN(n int, done func()) {
	db, err := sql.Open("sqlite3", os.Args[1]+"?mode=ro")
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(3)
	defer db.Close()
	for i := 0; i < n; i++ {
		r, err := runQ(db)
		if err != nil {
			panic(err)
		}
		if len(r) != 10 {
			panic("not 10")
		}
		fmt.Println(i)
		done()
	}
}

func runQ(db *sql.DB) ([]string, error) {
	rows, err := db.Query("select by FROM data WHERE by IS NOT NULL GROUP BY by ORDER BY SUM(score) LIMIT 10")
	// rows, err := db.Query("SELECT by FROM data WHERE by IS NOT NULL ORDER BY id LIMIT 10")
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, 10)
	for rows.Next() {
		// s := ""
		// err := rows.Scan(&s)
		// if err != nil {
		// 	return nil, err
		// }
		result = append(result, "")
	}
	return result, nil
}
