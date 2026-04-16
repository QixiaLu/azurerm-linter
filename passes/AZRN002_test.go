package passes_test

import (
	"testing"

	"github.com/qixialu/azurerm-linter/passes"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAZRN002(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, passes.AZRN002Analyzer, "testdata/src/azrn002")
}
