package AZC002

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/bflad/tfproviderlint/passes/commentignore"
	"golang.org/x/tools/go/analysis"
)

const Doc = `check that all String arguments have validation

The AZC002 analyzer reports cases where String type schema fields 
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

const analyzerName = "AZC002"

var Analyzer = &analysis.Analyzer{
	Name: analyzerName,
	Doc:  Doc,
	Requires: []*analysis.Analyzer{
		commentignore.Analyzer,
	},
	Run: run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	ignorer := pass.ResultOf[commentignore.Analyzer].(*commentignore.Ignorer)

	for _, f := range pass.Files {
		filePos := pass.Fset.Position(f.Pos())
		filename := filePos.Filename

		// Only check resource and data source files
		if !strings.HasSuffix(filename, "_resource.go") && !strings.HasSuffix(filename, "_data_source.go") {
			continue
		}

		// Skip test files
		if strings.HasSuffix(filename, "_test.go") {
			continue
		}

		ast.Inspect(f, func(n ast.Node) bool {
			// Look for composite literals that might be schema definitions
			comp, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}

			if ignorer.ShouldIgnore(analyzerName, comp) {
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

			for _, elt := range comp.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}

				key, ok := kv.Key.(*ast.BasicLit)
				if !ok || key.Kind != token.STRING {
					continue
				}

				propertyName := strings.Trim(key.Value, `"`)

				// Check if the value is a composite literal (schema definition)
				schemaLit, ok := kv.Value.(*ast.CompositeLit)
				if !ok {
					// TODO: Might need to include schema which defined in another file
					continue
				}

				// Analyze the schema definition
				isTypeString := false
				hasValidation := false
				isComputedOnly := false
				hasOptional := false

				for _, field := range schemaLit.Elts {
					kvField, ok := field.(*ast.KeyValueExpr)
					if !ok {
						continue
					}

					fieldName, ok := kvField.Key.(*ast.Ident)
					if !ok {
						continue
					}

					switch fieldName.Name {
					case "Type":
						// Check if Type is TypeString
						if sel, ok := kvField.Value.(*ast.SelectorExpr); ok {
							if sel.Sel.Name == "TypeString" {
								isTypeString = true
							}
						}
					case "ValidateFunc":
						hasValidation = true
					case "Computed":
						// Check if Computed is true
						if ident, ok := kvField.Value.(*ast.Ident); ok && ident.Name == "true" {
							isComputedOnly = true
						}
					case "Optional":
						// Check if Optional is true
						if ident, ok := kvField.Value.(*ast.Ident); ok && ident.Name == "true" {
							hasOptional = true
						}
					}
				}

				// Only check TypeString fields
				if !isTypeString {
					continue
				}

				// Skip if it's computed-only (Computed: true without Optional or Required)
				if isComputedOnly && !hasOptional {
					continue
				}

				// Report if no validation
				if !hasValidation {
					pass.Reportf(kv.Pos(), "%s: string argument %q must have ValidateFunc", analyzerName, propertyName)
				}
			}

			return true
		})
	}

	return nil, nil
}
