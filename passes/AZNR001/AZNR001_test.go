package AZNR001_test

import (
	"testing"

	"github.com/qixialu/azurerm-linter/passes/AZNR001"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, AZNR001.Analyzer, "testdata/src/a")
}
