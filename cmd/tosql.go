package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adamlouis/mksql/internal/tosql"
	"github.com/spf13/cobra"
)

var toSQLCmd = &cobra.Command{
	Use:   "tosql [SRC] [DST]",
	Short: `convert the provided source to a sqlite database`,
	Long:  `convert the provided source to a sqlite database`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("expected 2 positional arguments but received %d: %s", len(args), strings.Join(args, " "))
		}

		src, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		dst, err := filepath.Abs(args[1])
		if err != nil {
			return err
		}

		_ = os.Remove(dst)

		// for now use file extension to determine how to parse content
		// later, accept explicit arg or do "intelligently"
		ext := strings.ToLower(filepath.Ext(src))
		switch ext {
		case ".csv":
			start := time.Now()
			defer func() {
				fmt.Println("elapsed:", time.Since(start))
			}()
			return tosql.NewCSVToSQLer(tosql.CSVToSQLOpts{Strict: true}).ToSQL(dst, src)
		case ".tsv":
			start := time.Now()
			defer func() {
				fmt.Println("elapsed:", time.Since(start))
			}()
			return tosql.NewCSVToSQLer(tosql.CSVToSQLOpts{Strict: true, Comma: '\t'}).ToSQL(dst, src)
		}

		return fmt.Errorf("unsupported src %s", src)

	},
}

func init() {
	rootCmd.AddCommand(toSQLCmd)
}
