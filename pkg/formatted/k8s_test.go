package formatted

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fatih/color"
)

func TestColor(t *testing.T) {
	greenSuccess := color.New(color.FgHiGreen).Sprint("Succeeded")
	cs := ColorStatus("Succeeded")
	if cs != greenSuccess {
		t.Errorf("%s != %s", cs, greenSuccess)
	}
}

func TestAutoStepName(t *testing.T) {
	firstRound := AutoStepName("")
	assert.Equal(t, firstRound, "unnamed-0")

	secondRound := AutoStepName("named")
	assert.Equal(t, secondRound, "named")

	thirdRound := AutoStepName("")
	assert.Equal(t, thirdRound, "unnamed-2")
}
