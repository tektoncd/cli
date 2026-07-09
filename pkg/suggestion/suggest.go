// Copyright © 2020 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package suggestion

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// SubcommandsRequiredWithSuggestions will ensure we have a subcommand provided by the user and augments it with
// suggestion for commands, alias and help on root command.
func SubcommandsRequiredWithSuggestions(cmd *cobra.Command, args []string) error {
	requireMsg := "unknown command \"%s\" for \"%s\""
	typedName := ""
	// This will be triggered if cobra didn't find any subcommands.
	// Find some suggestions.
	var suggestions []string

	if len(args) != 0 && !cmd.DisableSuggestions {
		typedName += args[0]
		if cmd.SuggestionsMinimumDistance <= 0 {
			cmd.SuggestionsMinimumDistance = 2
		}
		// subcommand suggestions
		suggestions = cmd.SuggestionsFor(args[0])

		// subcommand alias suggestions (with distance, not exact)
		for _, c := range cmd.Commands() {
			if c.IsAvailableCommand() {
				candidate := suggestsByPrefixOrLd(typedName, c.Name(), cmd.SuggestionsMinimumDistance)
				if candidate == "" {
					continue
				}
				_, found := Find(suggestions, candidate)
				if !found {
					suggestions = append(suggestions, candidate)
				}
			}
		}

		// help for root command
		if !cmd.HasParent() {
			candidate := suggestsByPrefixOrLd(typedName, "help", cmd.SuggestionsMinimumDistance)
			if candidate != "" {
				suggestions = append(suggestions, candidate)
			}
		}
	}

	var suggestionsMsg string
	if len(suggestions) > 0 {
		suggestionsMsg += "\nDid you mean this?\n"
		for _, s := range suggestions {
			suggestionsMsg += fmt.Sprintf("\t%v\n", s)
		}
	}

	if suggestionsMsg != "" {
		requireMsg = fmt.Sprintf("%s\n%s", requireMsg, suggestionsMsg)
		return fmt.Errorf(requireMsg, typedName, cmd.CommandPath())
	}

	if typedName == "" {
		return cmd.Help()
	}
	return fmt.Errorf("command %s %s doesn't exist. Run %s help for available commands", cmd.Name(), typedName, cmd.Root().Name())
}

// suggestsByPrefixOrLd suggests a command by levenshtein distance or by prefix.
// It returns an empty string if nothing was found
func suggestsByPrefixOrLd(typedName, candidate string, minDistance int) string {
	levenshteinVariable := levenshteinDistance(typedName, candidate, true)
	suggestByLevenshtein := levenshteinVariable <= minDistance
	suggestByPrefix := strings.HasPrefix(strings.ToLower(candidate), strings.ToLower(typedName))
	if !suggestByLevenshtein && !suggestByPrefix {
		return ""
	}
	return candidate
}

// ld compares two strings and returns the levenshtein distance between them.
func levenshteinDistance(s, t string, ignoreCase bool) int {
	if ignoreCase {
		s = strings.ToLower(s)
		t = strings.ToLower(t)
	}
	d := make([][]int, len(s)+1)
	for i := range d {
		d[i] = make([]int, len(t)+1)
	}
	for i := range d {
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for j := 1; j <= len(t); j++ {
		for i := 1; i <= len(s); i++ {
			if s[i-1] == t[j-1] {
				d[i][j] = d[i-1][j-1]
			} else {
				minCost := d[i-1][j]
				if d[i][j-1] < minCost {
					minCost = d[i][j-1]
				}
				if d[i-1][j-1] < minCost {
					minCost = d[i-1][j-1]
				}
				d[i][j] = minCost + 1
			}
		}

	}
	return d[len(s)][len(t)]
}

// Find takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func Find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}
