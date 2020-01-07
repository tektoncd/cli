package names

import (
	"fmt"
	"strings"
)

func QuotedList(names []string) string {
	quoted := make([]string, len(names))
	for i := range names {
		quoted[i] = fmt.Sprintf("%q", names[i])
	}
	return strings.Join(quoted, ", ")
}
