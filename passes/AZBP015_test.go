package passes_test

import (
	"testing"

	"github.com/qixialu/azurerm-linter/passes"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAZBP015(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, passes.AZBP015Analyzer, "testdata/src/azbp015")
}
