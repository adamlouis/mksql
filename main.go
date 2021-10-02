package main

import (
	"fmt"
	"os"

	"github.com/adamlouis/mksql/cmd"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
