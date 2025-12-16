package AZBP002_test

import (
	"testing"

	"github.com/qixialu/azurerm-linter/passes/AZBP002"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAZBP002(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, AZBP002.Analyzer, "testdata/src/a")
}
