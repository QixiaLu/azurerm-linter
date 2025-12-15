package AZRN001_test

import (
	"testing"

	"github.com/qixialu/azurerm-linter/passes/AZRN001"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, AZRN001.Analyzer, "testdata/src/a")
}
