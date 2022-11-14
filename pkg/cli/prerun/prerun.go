package prerun

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/flags"
)

func PersistentPreRunE(p cli.Params) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := WarnExperimental(cmd); err != nil {
			return err
		}
		return flags.InitParams(p, cmd)
	}
}

func WarnExperimental(cmd *cobra.Command) error {
	if IsExperimental(cmd) {
		fmt.Fprintf(cmd.OutOrStderr(), "*Warning*: This is an experimental command, it's usage and behavior can change in the next release(s)\n")
	}
	return nil
}

func IsExperimental(cmd *cobra.Command) bool {
	if _, ok := cmd.Annotations["experimental"]; ok {
		return true
	}
	var experimental bool
	cmd.VisitParents(func(cmd *cobra.Command) {
		if _, ok := cmd.Annotations["experimental"]; ok {
			experimental = true
		}
	})
	return experimental
}
