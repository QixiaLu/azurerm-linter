package passes

import (
	"strings"

	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
	"github.com/bflad/tfproviderlint/passes/commentignore"
	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	localschema "github.com/qixialu/azurerm-linter/passes/schema"
	"github.com/qixialu/azurerm-linter/reporting"
	"golang.org/x/tools/go/analysis"
)

const AZRN002Doc = `check that Boolean property names do not start with 'is_'

The AZRN002 analyzer reports when property names begin with 'is_' prefix,
which is considered a redundant verb in Terraform schema naming.

Example violations:
  "is_enabled": {...}     // should be "enabled"
  "is_active": {...}      // should be "active"

Valid usage:
  "enabled": {...}
  "active": {...}`

const azrn002Name = "AZRN002"

var AZRN002Analyzer = &analysis.Analyzer{
	Name: azrn002Name,
	Doc:  AZRN002Doc,
	Run:  runAZRN002,
	Requires: []*analysis.Analyzer{
		localschema.LocalAnalyzer,
		commentignore.Analyzer,
	},
}

func runAZRN002(pass *analysis.Pass) (interface{}, error) {
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
		fieldName := cached.PropertyName

		if ignorer.ShouldIgnore(azrn002Name, schemaLit) {
			continue
		}

		if !schemaInfo.IsType(schema.SchemaValueTypeBool) {
			continue
		}

		if strings.HasPrefix(fieldName, "is_") {
			suggestedName := strings.ReplaceAll(fieldName, "is_", "")
			pos := pass.Fset.Position(schemaLit.Pos())

			if !loader.IsFileChanged(pos.Filename) {
				continue
			}

			reporting.Reportf(pass, reporting.ReportOptions{
				Rule:          azrn002Name,
				ReportPos:     schemaLit.Pos(),
				EvidenceFile:  pos.Filename,
				EvidenceLines: []int{pos.Line},
				MatchMode:     reporting.MatchModeExactAdded,
			}, "%s: field %q has redundant verbs %s (suggested: %s)\n",
				azrn002Name, fieldName,
				helper.IssueLine("'is_'"),
				helper.FixedCode(suggestedName))
		}
	}

	return nil, nil
}
