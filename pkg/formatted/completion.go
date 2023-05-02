package formatted

import (
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// GetObjectsWithKubectl return completions with kubectl, we are doing this with
// kubectl since we have caching and without it completion is way too slow
func GetObjectsWithKubectl(obj string) []string {
	out, err := exec.Command("kubectl", "get", obj, "-o=jsonpath={range .items[*]}{.metadata.name} {end}").Output()
	if err != nil {
		return nil
	}
	return strings.Fields(string(out))
}

// BaseCompletion return a completion for a kubernetes object using Kubectl
func BaseCompletion(target string, args []string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return GetObjectsWithKubectl(target), cobra.ShellCompDirectiveNoFileComp
}

// ParentCompletion do completion of command to the Parent
func ParentCompletion(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	return BaseCompletion(cmd.Parent().Name(), args)
}
