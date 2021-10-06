package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"crawshaw.io/sqlite"
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
	conn, err := sqlite.OpenConn(os.Args[1], sqlite.SQLITE_OPEN_READONLY|sqlite.SQLITE_OPEN_SHAREDCACHE)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	for i := 0; i < n; i++ {
		r, err := runQ(conn)
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

func runQ(conn *sqlite.Conn) ([]string, error) {

	stmt, err := conn.Prepare("select by FROM data WHERE by IS NOT NULL GROUP BY by ORDER BY SUM(score) LIMIT 10")
	// stmt, err := conn.Prepare("SELECT by FROM data WHERE by IS NOT NULL ORDER BY id LIMIT 10")
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, 10)
	for {
		if hasRow, err := stmt.Step(); err != nil {
			return nil, err
		} else if !hasRow {
			break
		}
		// by := stmt.GetText("by")
		result = append(result, "")
	}
	return result, nil
}
