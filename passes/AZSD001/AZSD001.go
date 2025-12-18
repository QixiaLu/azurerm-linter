package AZSD001

import (
	"go/ast"
	"go/token"

	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
	"github.com/qixialu/azurerm-linter/passes/changedlines"
	localschema "github.com/qixialu/azurerm-linter/passes/helpers/schema/localSchemaInfos"
	"github.com/qixialu/azurerm-linter/passes/util"
	"golang.org/x/tools/go/analysis"
)

const Doc = `check MaxItems:1 blocks with single property should be flattened

The AZSD001 analyzer checks that blocks with MaxItems: 1 containing only a single 
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

const analyzerName = "AZSD001"

var Analyzer = &analysis.Analyzer{
	Name:     analyzerName,
	Doc:      Doc,
	Run:      run,
	Requires: []*analysis.Analyzer{localschema.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	schemaInfoCache, ok := pass.ResultOf[localschema.Analyzer].(map[*ast.CompositeLit]*localschema.SchemaInfoWithName)
	if !ok {
		return nil, nil
	}

	// Build file comments map for all files
	fileCommentsMap := make(map[string][]*ast.CommentGroup)
	for _, f := range pass.Files {
		filename := pass.Fset.Position(f.Pos()).Filename
		fileCommentsMap[filename] = f.Comments
	}

	// Iterate over cached schema infos
	for schemaLit, cached := range schemaInfoCache {
		schemaInfo := cached.Info
		fieldName := cached.PropertyName

		// Check if MaxItems is 1
		if schemaInfo.Schema.MaxItems != 1 {
			continue
		}

		// Get Elem field
		elemKV := schemaInfo.Fields[schema.SchemaFieldElem]
		if elemKV == nil {
			continue
		}

		// Check if Elem is &schema.Resource{...}
		var resourceSchema *ast.CompositeLit
		if unary, ok := elemKV.Value.(*ast.UnaryExpr); ok && unary.Op == token.AND {
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
			filename := pass.Fset.Position(schemaLit.Pos()).Filename
			elemLine := pass.Fset.Position(elemKV.Value.Pos()).Line

			hasComment := false
			comments := fileCommentsMap[filename]
			for _, cg := range comments {
				for _, c := range cg.List {
					commentLine := pass.Fset.Position(c.Pos()).Line
					// Check if comment is on the same line as Elem (inline comment)
					if commentLine == elemLine {
						hasComment = true
						break
					}
				}
				if hasComment {
					break
				}
			}

			if !hasComment {
				pos := pass.Fset.Position(schemaLit.Pos())
				if changedlines.ShouldReport(pos.Filename, pos.Line) {
					pass.Reportf(schemaLit.Pos(), "%s: field %q has %s with only one nested property - consider %s or add inline comment explaining why (e.g., %s)\n",
						analyzerName, fieldName,
						util.IssueLine("MaxItems: 1"),
						util.FixedCode("flattening"),
						util.FixedCode("'// Additional properties will be added per service team confirmation'"))
				}
			}
		}
	}

	return nil, nil
}
