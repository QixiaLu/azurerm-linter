package AZC001_test

import (
	"testing"

	"github.com/qixialu/azurerm-linter/passes/AZC001"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), AZC001.Analyzer, "a")
}
