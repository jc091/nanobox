package commands

import (
	"github.com/spf13/cobra"

	"github.com/nanobox-io/nanobox/processor"
)

var (

	// DeployCmd ...
	DeployCmd = &cobra.Command{
		Use:   "deploy",
		Short: "Deploys your generated build package to a production app.",
		Long:  ``,
		Run:   deployFn,
	}
)

// deployFn ...
func deployFn(ccmd *cobra.Command, args []string) {
	processor.DefaultConfig.Meta["alias"] = "default"
	if len(args) == 1 {
		processor.DefaultConfig.Meta["alias"] = args[0]
	}

	//
	if err := processor.Run("deploy", processor.DefaultConfig); err != nil {

	}
}
