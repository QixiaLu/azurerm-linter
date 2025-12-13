package AZC006_test

import (
	"testing"

	"github.com/qixialu/azurerm-linter/passes/AZC006"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, AZC006.Analyzer, "testdata/src/a")
}
