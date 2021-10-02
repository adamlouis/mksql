package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "mksql",
	Short: `make & serve sqlite databases`,
	Long:  `make & serve sqlite databases`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
}
