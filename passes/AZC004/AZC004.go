package AZC004

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
	"github.com/qixialu/azurerm-linter/passes/changedlines"
)

const Doc = `check MaxItems:1 blocks with single property should be flattened

The AZC004 analyzer checks that blocks with MaxItems: 1 containing only a single 
nested property should be flattened unless there's a comment explaining why.

Example violation:
  "config": {
      Type:     schema.TypeList,
      MaxItems: 1,
      Elem: &schema.Resource{
          Schema: map[string]*schema.Schema{
              "value": {...},  // Only one property - should be flattened
          },
      },
  }

Valid usage (flattened):
  "config_value": {...}

Valid usage (with explanation):
  "config": {
      Type:     schema.TypeList,
      MaxItems: 1,
      // Additional properties will be added per service team confirmation
      Elem: &schema.Resource{
          Schema: map[string]*schema.Schema{
              "value": {...},
          },
      },
  }`

const analyzerName = "AZC004"

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
		filePos := pass.Fset.Position(f.Pos())
		filename := filePos.Filename

		if strings.HasSuffix(filename, "_test.go") {
			continue
		}

		ast.Inspect(f, func(n ast.Node) bool {
			comp, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}

			// Check if this is a map[string]*schema.Schema
			mapType, ok := comp.Type.(*ast.MapType)
			if !ok {
				return true
			}
			if ident, ok := mapType.Key.(*ast.Ident); !ok || ident.Name != "string" {
				return true
			}
			starExpr, ok := mapType.Value.(*ast.StarExpr)
			if !ok {
				return true
			}
			selExpr, ok := starExpr.X.(*ast.SelectorExpr)
			if !ok || selExpr.Sel.Name != "Schema" {
				return true
			}

			// Check each field in the schema map
			for _, elt := range comp.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}

				keyLit, ok := kv.Key.(*ast.BasicLit)
				if !ok || keyLit.Kind != token.STRING {
					continue
				}
				fieldName := strings.Trim(keyLit.Value, `"`)

				// Only check inline schema definitions
				schemaLit, ok := kv.Value.(*ast.CompositeLit)
				if !ok {
					continue
				}

				// Look for MaxItems: 1 and Elem with nested schema
				hasMaxItems1 := false
				var elemValue ast.Expr

				for _, fld := range schemaLit.Elts {
					fieldKV, ok := fld.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					ident, ok := fieldKV.Key.(*ast.Ident)
					if !ok {
						continue
					}

					switch ident.Name {
					case "MaxItems":
						if lit, ok := fieldKV.Value.(*ast.BasicLit); ok && lit.Value == "1" {
							hasMaxItems1 = true
						}
					case "Elem":
						elemValue = fieldKV.Value
					}
				}

				// Only check if MaxItems: 1
				if !hasMaxItems1 || elemValue == nil {
					continue
				}

				// Check if Elem is &schema.Resource{...}
				var resourceSchema *ast.CompositeLit

				// Handle &schema.Resource{...}
				if unary, ok := elemValue.(*ast.UnaryExpr); ok && unary.Op == token.AND {
					if compLit, ok := unary.X.(*ast.CompositeLit); ok {
						resourceSchema = compLit
					}
				}

				if resourceSchema == nil {
					continue
				}

				// Find the Schema field in the Resource
				var nestedSchemaMap *ast.CompositeLit
				for _, fld := range resourceSchema.Elts {
					fieldKV, ok := fld.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					if ident, ok := fieldKV.Key.(*ast.Ident); ok && ident.Name == "Schema" {
						if compLit, ok := fieldKV.Value.(*ast.CompositeLit); ok {
							nestedSchemaMap = compLit
						}
						break
					}
				}

				if nestedSchemaMap == nil {
					continue
				}

				// Count properties in the nested schema
				propertyCount := 0
				for _, elt := range nestedSchemaMap.Elts {
					if _, ok := elt.(*ast.KeyValueExpr); ok {
						propertyCount++
					}
				}

				// If only one property, check for any explanatory comment
				if propertyCount == 1 {
					hasComment := false
					elemLine := pass.Fset.Position(elemValue.Pos()).Line
					fieldStartLine := pass.Fset.Position(kv.Pos()).Line

					// Look for any comments between field start and Elem or within a few lines after Elem
					for _, cg := range f.Comments {
						for _, c := range cg.List {
							commentLine := pass.Fset.Position(c.Pos()).Line

							// Check if comment is between field start and Elem or within a few lines after Elem
							if commentLine >= fieldStartLine && commentLine <= elemLine+2 {
								hasComment = true
								break
							}
						}
						if hasComment {
							break
						}
					}

					if !hasComment {
						pos := pass.Fset.Position(kv.Pos())
						// Only report if this line is in the changed lines (or filter is disabled)
						if changedlines.ShouldReport(pos.Filename, pos.Line) {
							pass.Reportf(kv.Pos(), "%s: field %q has MaxItems: 1 with only one nested property - consider flattening or add inline comment explaining why (e.g., '// Additional properties will be added per service team confirmation')", analyzerName, fieldName)
						}
					}
				}
			}

			return true
		})
	}

	return nil, nil
}
