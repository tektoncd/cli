package test

import (
	"testing"

	_ "github.com/tektoncd/plumbing/scripts"
)

func TestDummy(t *testing.T) {
	t.Log("This is required to make sure we get tektoncd/plumbing in the repository, folder vendor")
}
