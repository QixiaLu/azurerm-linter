package passes

import (
	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
	"github.com/bflad/tfproviderlint/passes/commentignore"
	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	localschema "github.com/qixialu/azurerm-linter/passes/schema"
	"github.com/qixialu/azurerm-linter/reporting"
	"golang.org/x/tools/go/analysis"
)

const AZNR010Doc = `check that schema fields do not have redundant Description

The AZNR010 analyzer reports cases where schema fields declare a Description
field. In the AzureRM provider, descriptions are generated from documentation
and should not be specified inline in schema definitions.

Example violations:
  "name": {
      Type:        schema.TypeString,
      Required:    true,
      Description: "The name of the resource.",
  }

Valid usage:
  "name": {
      Type:     schema.TypeString,
      Required: true,
  }`

const aznr010Name = "AZNR010"

var AZNR010Analyzer = &analysis.Analyzer{
	Name:     aznr010Name,
	Doc:      AZNR010Doc,
	Run:      runAZNR010,
	Requires: []*analysis.Analyzer{localschema.LocalAnalyzer, commentignore.Analyzer},
}

func runAZNR010(pass *analysis.Pass) (interface{}, error) {
	ignorer, ok := pass.ResultOf[commentignore.Analyzer].(*commentignore.Ignorer)
	if !ok {
		return nil, nil
	}

	schemaInfoList, ok := pass.ResultOf[localschema.LocalAnalyzer].(localschema.LocalSchemaInfoList)
	if !ok {
		return nil, nil
	}

	for _, cached := range schemaInfoList {
		schemaInfo := cached.Info
		schemaLit := schemaInfo.AstCompositeLit

		if ignorer.ShouldIgnore(aznr010Name, schemaInfo.AstCompositeLit) {
			continue
		}

		// Check if description exists
		hasDescription := schemaInfo.DeclaresField(schema.SchemaFieldDescription)

		if hasDescription {
			pos := pass.Fset.Position(schemaLit.Pos())
			if !loader.IsFileChanged(pos.Filename) {
				continue
			}
			reporting.Reportf(pass, reporting.ReportOptions{
				Rule: aznr010Name,
				ReportPos: schemaLit.Pos(),
				EvidenceFile: pos.Filename,
				EvidenceLines: []int{pos.Line},
				MatchMode: reporting.MatchModeExactAdded,
			}, "%s: %s is redundant\n", aznr010Name, helper.ItalicCode("Description"))
		}
	}

	return nil, nil
}
