package passes_test

import (
	"testing"

	"github.com/qixialu/azurerm-linter/passes"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAZSD005(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, passes.AZSD005Analyzer, "testdata/src/azsd005")
}
