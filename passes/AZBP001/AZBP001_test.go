package AZBP001_test

import (
	"testing"

	"github.com/qixialu/azurerm-linter/passes/AZBP001"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAZBP001(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, AZBP001.Analyzer, "testdata/src/a")
}
