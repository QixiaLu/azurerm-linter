package AZC003

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/bflad/tfproviderlint/passes/commentignore"
	"golang.org/x/tools/go/analysis"
)

const Doc = `check Optional+Computed fields follow conventions

The AZC003 analyzer checks that fields marked as both Optional and Computed:
1. Have properties in sequence: Optional, Comment, Computed
2. Have a comment starting with "// NOTE: O+C " explaining why

Example violation:
  "field": {
      Type:     schema.TypeString,
      Optional: true,
      Computed: true,  // Missing NOTE: O+C comment
  }

Valid usage:
  "field": {
      Type:     schema.TypeString,
      Optional: true,
      // NOTE: O+C - field can be set by user or computed from API when not provided
      Computed: true,
  }`

const analyzerName = "AZC003"

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
		if strings.HasSuffix(filename, "_test.go") {
			continue
		}

		ast.Inspect(f, func(n ast.Node) bool {
			comp, ok := n.(*ast.CompositeLit)
			if !ok || ignorer.ShouldIgnore(analyzerName, comp) {
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

				// Track Optional and Computed positions
				var optionalPos token.Pos
				var computedPos token.Pos
				hasOptional := false
				hasComputed := false

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
					case "Optional":
						if id, ok := fieldKV.Value.(*ast.Ident); ok && id.Name == "true" {
							hasOptional = true
							optionalPos = fieldKV.Pos()
						}
					case "Computed":
						if id, ok := fieldKV.Value.(*ast.Ident); ok && id.Name == "true" {
							hasComputed = true
							computedPos = fieldKV.Pos()
						}
					}
				}

				if !hasOptional || !hasComputed {
					continue
				}

				// Check order: Optional should come before Computed
				if optionalPos > computedPos {
					pass.Reportf(kv.Pos(), "%s: field %q has Optional and Computed in wrong order (Optional must come before Computed)", analyzerName, fieldName)
					continue
				}

				// Check for NOTE: O+C comment between Optional and Computed
				hasOCComment := false
				optionalLine := pass.Fset.Position(optionalPos).Line
				computedLine := pass.Fset.Position(computedPos).Line

				// Look for comments between Optional and Computed lines
				for _, cg := range f.Comments {
					for _, c := range cg.List {
						commentLine := pass.Fset.Position(c.Pos()).Line
						if commentLine > optionalLine && commentLine < computedLine {
							if strings.Contains(c.Text, "NOTE: O+C") {
								hasOCComment = true
								break
							}
						}
					}
					if hasOCComment {
						break
					}
				}

				if !hasOCComment {
					pass.Reportf(kv.Pos(), "%s: field %q is Optional+Computed but missing required comment. Add '// NOTE: O+C - <explanation>' between Optional and Computed", analyzerName, fieldName)
				}
			}

			return true
		})
	}

	return nil, nil
}
