package AZRE001_test

import (
	"testing"

	"github.com/qixialu/azurerm-linter/passes/AZRE001"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), AZRE001.Analyzer, "a")
}
