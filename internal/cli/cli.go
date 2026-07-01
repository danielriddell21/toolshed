package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/danielriddell21/toolshed/internal/buildinfo"
)

func Execute(root *cobra.Command) {
	root.Version = buildinfo.Version
	root.SilenceUsage = true
	root.AddCommand(completionCmd(root.Name()))
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", root.Name(), err)
		os.Exit(1)
	}
}
