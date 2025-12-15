package AZBP001

import (
	"go/ast"
	"strings"

	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
	"github.com/qixialu/azurerm-linter/passes/changedlines"
	"github.com/qixialu/azurerm-linter/passes/util"
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
	Name: analyzerName,
	Doc:  Doc,
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Skip migration packages
	if strings.Contains(pass.Pkg.Path(), "/migration") {
		return nil, nil
	}

	for _, f := range pass.Files {
		filename := pass.Fset.Position(f.Pos()).Filename

		if !changedlines.IsFileChanged(filename) {
			continue
		}

		if strings.HasSuffix(filename, "_test.go") {
			continue
		}

		ast.Inspect(f, func(n ast.Node) bool {
			// Look for composite literals that might be schema maps
			comp, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}

			// Check if it's a map literal (map[string]*schema.Schema or map[string]*pluginsdk.Schema)
			mapType, ok := comp.Type.(*ast.MapType)
			if !ok {
				return true
			}

			// Check if key is string
			if ident, ok := mapType.Key.(*ast.Ident); !ok || ident.Name != "string" {
				return true
			}

			// Check if value is *schema.Schema or *pluginsdk.Schema
			starExpr, ok := mapType.Value.(*ast.StarExpr)
			if !ok {
				return true
			}

			selExpr, ok := starExpr.X.(*ast.SelectorExpr)
			if !ok || selExpr.Sel.Name != "Schema" {
				return true
			}

			// Iterate through each schema field
			for _, elt := range comp.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}

				// Get field name
				key, ok := kv.Key.(*ast.BasicLit)
				if !ok {
					continue
				}
				propertyName := strings.Trim(key.Value, `"`)

				// Check if the value is a schema composite literal
				schemaLit, ok := kv.Value.(*ast.CompositeLit)
				if !ok {
					continue
				}

				// Use tfproviderlint's SchemaInfo to analyze the schema
				schemaInfo := schema.NewSchemaInfo(schemaLit, pass.TypesInfo)
				if !schemaInfo.IsType(schema.SchemaValueTypeString) {
					continue
				}
				// Skip if it's computed-only (no Required or Optional)
				if schemaInfo.Schema.Computed && !schemaInfo.Schema.Required && !schemaInfo.Schema.Optional {
					continue
				}

				// Check if validation exists (ValidateFunc)
				hasValidation := schemaInfo.DeclaresField(schema.SchemaFieldValidateFunc)

				// Report if no validation
				if !hasValidation {
					pos := pass.Fset.Position(kv.Pos())
					if changedlines.ShouldReport(pos.Filename, pos.Line) {
						pass.Reportf(kv.Pos(), "%s: string argument %q must have %s\n", analyzerName, propertyName, util.FixedCode("ValidateFunc"))
					}
				}
			}

			return true
		})
	}

	return nil, nil
}
