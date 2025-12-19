package AZBP001

import (
	"go/ast"

	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
	"github.com/qixialu/azurerm-linter/loader"
	localschema "github.com/qixialu/azurerm-linter/passes/schema"
	"golang.org/x/tools/go/analysis"
)

const Doc = `check that all String arguments have validation

The AZBP001 analyzer reports cases where String type schema fields 
(Required or Optional) do not have a ValidateFunc.

Example violations:
  "name": {
      Type:     schema.TypeString,
      Required: true,
      // Missing ValidateFunc!
  }
  
Valid usage:
  "name": {
      Type:         schema.TypeString,
      Required:     true,
      ValidateFunc: validation.StringIsNotEmpty,
  }
  
  "description": {
      Type:     schema.TypeString,
      Computed: true,  // OK - computed-only fields don't need validation
  }`

const analyzerName = "AZBP001"

var Analyzer = &analysis.Analyzer{
	Name:     analyzerName,
	Doc:      Doc,
	Run:      run,
	Requires: []*analysis.Analyzer{localschema.LocalAnalyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	schemaInfoCache, ok := pass.ResultOf[localschema.LocalAnalyzer].(map[*ast.CompositeLit]*localschema.LocalSchemaInfoWithName)
	if !ok {
		return nil, nil
	}

	for schemaLit, cached := range schemaInfoCache {
		schemaInfo := cached.Info
		propertyName := cached.PropertyName

		// Type check: only check String fields
		if !schemaInfo.IsType(schema.SchemaValueTypeString) {
			continue
		}

		// Skip computed-only fields (no Required or Optional)
		if schemaInfo.Schema.Computed && !schemaInfo.Schema.Required && !schemaInfo.Schema.Optional {
			continue
		}

		// Check if validation exists
		hasValidation := schemaInfo.DeclaresField(schema.SchemaFieldValidateFunc)

		if !hasValidation {
			pos := pass.Fset.Position(schemaLit.Pos())
			if loader.ShouldReport(pos.Filename, pos.Line) {
				pass.Reportf(schemaLit.Pos(), "%s: string argument %q must have ValidateFunc\n",
					analyzerName, propertyName)
			}
		}
	}

	return nil, nil
}
