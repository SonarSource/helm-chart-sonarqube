package tests

import (
	"os"
	"testing"

	"github.com/AlaudaDevops/bdd"
	"github.com/AlaudaDevops/bdd/steps"

	// register built-in config generators
	_ "github.com/AlaudaDevops/bdd/steps/kubernetes/generators"
)

func TestMain(m *testing.M) {
	bdd.New().
		WithOption(bdd.WithFeaturePaths("./features")).
		WithSteps(steps.BuiltinSteps...).
		Run()

	exitVal := m.Run()
	os.Exit(exitVal)
}
