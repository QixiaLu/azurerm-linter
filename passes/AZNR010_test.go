package passes_test

import (
	"testing"

	"github.com/qixialu/azurerm-linter/passes"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAZNR010(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, passes.AZNR010Analyzer, "testdata/src/aznr010")
}
