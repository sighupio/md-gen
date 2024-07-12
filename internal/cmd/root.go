package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "md-gen",
		Short:         "Generates a markdown file from a json schema file",
		Long:          "Generates a markdown file from a json schema file",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			logrus.SetLevel(logrus.DebugLevel)
		},
	}

	rootCmd.AddCommand(NewGenCmd())

	return rootCmd
}
