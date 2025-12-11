package AZC005_test

import (
	"testing"

	"github.com/qixialu/azurerm-linter/passes/AZC005"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, AZC005.Analyzer, "testdata/src/a")
}
