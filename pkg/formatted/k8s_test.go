package formatted

import (
	"testing"

	"github.com/fatih/color"
)

func TestColor(t *testing.T) {
	greenSuccess := color.New(color.FgHiGreen).Sprint("Succeeded")
	cs := ColorStatus("Succeeded")
	if cs != greenSuccess {
		t.Errorf("%s != %s", cs, greenSuccess)
	}
}
