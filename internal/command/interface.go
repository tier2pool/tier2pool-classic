package command

import "github.com/spf13/cobra"

type Interface interface {
	Initialize(cmd *cobra.Command) error
	Run(cmd *cobra.Command, args []string) error
}
