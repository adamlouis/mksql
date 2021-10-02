package cmd

import (
	"os"
	"path/filepath"

	"github.com/adamlouis/mksql/internal/server"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: `run the web service & UI`,
	Long:  `run the web service & UI`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir, err := filepath.Abs("./data")
		if err != nil {
			return err
		}

		if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
			return err
		}

		return server.NewServer(server.ServerOpts{
			Port:    9876,
			DataDir: dataDir,
		}).Serve()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
