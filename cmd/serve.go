package cmd

import (
	"github.com/adamlouis/mksql/internal/server"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: `run the web service & UI`,
	Long:  `run the web service & UI`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return server.Serve(server.ServerOpts{Port: 9876})
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
