package e2e

import (
	"log"
	"os"
	"testing"

	"github.com/tektoncd/cli/test/environment"
)

var TestEnv = &environment.TknRunner{}

func init() {
	var err error
	TestEnv, err = environment.NewTknRunner()
	if err != nil {
		log.Fatalf("Error while creating Env %+v", err)
	}
}

func TestMain(m *testing.M) {
	log.Println("Running main e2e Test suite ")
	v := m.Run()
	os.Exit(v)
}
