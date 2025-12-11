package AZC002_test

import (
	"testing"

	"github.com/qixialu/azurerm-linter/passes/AZC002"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAZC002(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, AZC002.Analyzer, "testdata/src/a")
}