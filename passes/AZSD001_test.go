package passes_test

import (
	"testing"

	"github.com/qixialu/azurerm-linter/passes"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAZSD001(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, passes.AZSD001Analyzer, "testdata/src/azsd001")
}
