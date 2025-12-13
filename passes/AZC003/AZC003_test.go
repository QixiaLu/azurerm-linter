package AZC003_test

import (
	"testing"

	"github.com/qixialu/azurerm-linter/passes/AZC003"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAZC003(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, AZC003.Analyzer, "testdata/src/a")
}