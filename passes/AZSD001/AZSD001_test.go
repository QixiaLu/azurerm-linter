package AZSD001_test

import (
	"testing"

	"github.com/qixialu/azurerm-linter/passes/AZSD001"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, AZSD001.Analyzer, "testdata/src/a")
}
