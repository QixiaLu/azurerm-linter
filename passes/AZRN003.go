package passes

import (
	"fmt"
	"strings"

	"github.com/bflad/tfproviderlint/passes/commentignore"
	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	localschema "github.com/qixialu/azurerm-linter/passes/schema"
	"github.com/qixialu/azurerm-linter/reporting"
	"golang.org/x/tools/go/analysis"
)

const AZRN003Doc = `check that property names use non-abbreviated forms

The AZRN003 analyzer reports when property names contain common abbreviations
instead of using the full word form as required by the naming conventions.

Reference: https://github.com/hashicorp/terraform-provider-azurerm/blob/main/contributing/topics/reference-naming.md

Example violations:
  "min_size_gb": {...}       // should be "minimum_size_in_gb"
  "max_retry_count": {...}   // should be "maximum_retry_count"
  "vm_name": {...}           // should be "virtual_machine_name"

Valid usage:
  "minimum_size_in_gb": {...}
  "maximum_retry_count": {...}
  "virtual_machine_name": {...}`

const azrn003Name = "AZRN003"

var AZRN003Analyzer = &analysis.Analyzer{
	Name: azrn003Name,
	Doc:  AZRN003Doc,
	Run:  runAZRN003,
	Requires: []*analysis.Analyzer{
		localschema.LocalAnalyzer,
		commentignore.Analyzer,
	},
}

// abbreviations maps abbreviated segments to their full forms.
// Each key is matched as a complete segment between underscores in property names.
var abbreviations = map[string]string{
	"min":  "minimum",
	"max":  "maximum",
	"vm":   "virtual_machine",
	"vmss": "virtual_machine_scale_set",
	"vnet": "virtual_network",
	"rg":   "resource_group",
}

// suffixAbbreviations maps abbreviated suffixes (including underscore prefix) to their full forms.
// These are checked at the end of property names.
var suffixAbbreviations = map[string]string{
	"_gb": "_in_gb",
	"_mb": "_in_mb",
}

func runAZRN003(pass *analysis.Pass) (interface{}, error) {
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

		if ignorer.ShouldIgnore(azrn003Name, schemaLit) {
			continue
		}

		abbreviation, expansion := findAbbreviation(fieldName)
		if abbreviation == "" {
			continue
		}

		pos := pass.Fset.Position(schemaLit.Pos())
		if !loader.IsFileChanged(pos.Filename) {
			continue
		}

		reporting.Reportf(pass, reporting.ReportOptions{
			Rule:          azrn003Name,
			ReportPos:     schemaLit.Pos(),
			EvidenceFile:  pos.Filename,
			EvidenceLines: []int{pos.Line},
			MatchMode:     reporting.MatchModeExactAdded,
		}, "%s: field %q contains abbreviation %s, use non-abbreviated form %s\n",
			azrn003Name, fieldName,
			helper.IssueLine(fmt.Sprintf("'%s'", abbreviation)),
			helper.FixedCode(expansion))
	}

	return nil, nil
}

// findAbbreviation checks if the field name contains any known abbreviation.
// Returns the matched abbreviation and its expansion, or empty strings if none found.
func findAbbreviation(fieldName string) (string, string) {
	for suffix, expansion := range suffixAbbreviations {
		if strings.HasSuffix(fieldName, suffix) {
			// Make sure it's not already in the expanded form
			if !strings.HasSuffix(fieldName, expansion) {
				return suffix, expansion
			}
		}
	}

	// Check segment-based abbreviations
	segments := strings.Split(fieldName, "_")
	for _, segment := range segments {
		if expansion, ok := abbreviations[segment]; ok {
			return segment, expansion
		}
	}

	return "", ""
}
