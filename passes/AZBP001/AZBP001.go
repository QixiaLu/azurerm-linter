package AZBP001

import (
	"go/ast"
	"strings"

	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
	"github.com/qixialu/azurerm-linter/passes/changedlines"
	"github.com/qixialu/azurerm-linter/passes/helpers/schemafields"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
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
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Skip migration packages
	if strings.Contains(pass.Pkg.Path(), "/migration") {
		return nil, nil
	}

	// Get the shared inspector - this reuses the AST traversal
	inspector, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, nil
	}

	// Pre-filter: only look at CompositeLit nodes
	nodeFilter := []ast.Node{
		(*ast.CompositeLit)(nil),
	}

	inspector.Preorder(nodeFilter, func(n ast.Node) {
		comp, ok := n.(*ast.CompositeLit)
		if !ok {
			return
		}

		// Get filename for this node
		filename := pass.Fset.Position(comp.Pos()).Filename

		if !changedlines.IsFileChanged(filename) {
			return
		}

		if strings.HasSuffix(filename, "_test.go") {
			return
		}

		// Check if it's a schema map
		if !schemafields.IsSchemaMap(comp) {
			return
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
					pass.Reportf(kv.Pos(), "%s: string argument %q must have ValidateFunc",
						analyzerName, propertyName)
				}
			}
		}
	})

	return nil, nil
}
