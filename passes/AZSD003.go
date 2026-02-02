package passes

import (
	"go/ast"
	"go/token"
	"strconv"

	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
	"github.com/bflad/tfproviderlint/passes/commentignore"
	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	localschema "github.com/qixialu/azurerm-linter/passes/schema"
	"golang.org/x/tools/go/analysis"
)

const AZSD003Doc = `check for redundant use of ConflictsWith when ExactlyOneOf already covers the same fields

The AZSD003 analyzer checks that when both ExactlyOneOf and ConflictsWith are used,
the ConflictsWith values are not already covered by ExactlyOneOf. If a field is in
ExactlyOneOf, adding it to ConflictsWith is redundant because ExactlyOneOf already
implies mutual exclusivity.

Example violation:
  "field_a": {
      Type:          pluginsdk.TypeString,
      Optional:      true,
      ExactlyOneOf:  []string{"field_a", "field_b"},
      ConflictsWith: []string{"field_b"},  // Redundant - field_b is already in ExactlyOneOf
  }

Valid usage (ConflictsWith has different fields than ExactlyOneOf):
  "pipeline": {
      Type:          pluginsdk.TypeList,
      Optional:      true,
      ExactlyOneOf:  []string{"pipeline", "pipeline_name"},
      ConflictsWith: []string{"pipeline_parameters"},  // OK - different field
  }

Valid usage (ExactlyOneOf only):
  "field_a": {
      Type:         pluginsdk.TypeString,
      Optional:     true,
      ExactlyOneOf: []string{"field_a", "field_b"},
  }`

const azsd003Name = "AZSD003"

var AZSD003Analyzer = &analysis.Analyzer{
	Name: azsd003Name,
	Doc:  AZSD003Doc,
	Run:  runAZSD003,
	Requires: []*analysis.Analyzer{
		localschema.LocalAnalyzer,
		commentignore.Analyzer,
	},
}

func runAZSD003(pass *analysis.Pass) (interface{}, error) {
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

		if ignorer.ShouldIgnore(azsd003Name, schemaLit) {
			continue
		}

		// Check if both ExactlyOneOf and ConflictsWith are present
		exactlyOneOfKV := schemaInfo.Fields[schema.SchemaFieldExactlyOneOf]
		conflictsWithKV := schemaInfo.Fields[schema.SchemaFieldConflictsWith]

		if exactlyOneOfKV == nil || conflictsWithKV == nil {
			continue
		}

		// Extract string values from ExactlyOneOf
		exactlyOneOfValues := extractStringSliceValues(exactlyOneOfKV.Value)
		if len(exactlyOneOfValues) == 0 {
			continue
		}

		// Extract string values from ConflictsWith
		conflictsWithValues := extractStringSliceValues(conflictsWithKV.Value)
		if len(conflictsWithValues) == 0 {
			continue
		}

		// Check for overlap - find ConflictsWith values that are also in ExactlyOneOf
		exactlyOneOfSet := make(map[string]bool)
		for _, v := range exactlyOneOfValues {
			exactlyOneOfSet[v] = true
		}

		var redundantFields []string
		for _, v := range conflictsWithValues {
			if exactlyOneOfSet[v] {
				redundantFields = append(redundantFields, v)
			}
		}

		// Only report if there's overlap
		if len(redundantFields) > 0 {
			pos := pass.Fset.Position(schemaLit.Pos())
			if loader.ShouldReport(pos.Filename, pos.Line) {
				pass.Reportf(schemaLit.Pos(), "%s: ConflictsWith contains %s which is redundant - already covered by ExactlyOneOf",
					azsd003Name,
					helper.IssueLine(formatFieldList(redundantFields)))
			}
		}
	}

	return nil, nil
}

// extractStringSliceValues extracts string values from a composite literal like []string{"a", "b"}
func extractStringSliceValues(expr ast.Expr) []string {
	var values []string

	compositeLit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return values
	}

	for _, elt := range compositeLit.Elts {
		if lit, ok := elt.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if unquoted, err := strconv.Unquote(lit.Value); err == nil {
				values = append(values, unquoted)
			}
		}
	}

	return values
}

// formatFieldList formats a list of field names for display
func formatFieldList(fields []string) string {
	if len(fields) == 1 {
		return fields[0]
	}
	result := ""
	for i, f := range fields {
		if i > 0 {
			result += ", "
		}
		result += f
	}
	return result
}
