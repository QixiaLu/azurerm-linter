package AZC004_test

import (
	"testing"

	"github.com/qixialu/azurerm-linter/passes/AZC004"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, AZC004.Analyzer, "testdata/src/a")
}
