package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var toSQLCmd = &cobra.Command{
	Use:   "tosql",
	Short: `convert the provided source to a sqlite database`,
	Long:  `convert the provided source to a sqlite database`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("TODO")
	},
}

func init() {
	rootCmd.AddCommand(toSQLCmd)
}
