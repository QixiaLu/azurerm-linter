package passes

import (
	"go/ast"
	"go/token"

	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
	"github.com/bflad/tfproviderlint/passes/commentignore"
	"github.com/qixialu/azurerm-linter/helper"
	"github.com/qixialu/azurerm-linter/loader"
	localschema "github.com/qixialu/azurerm-linter/passes/schema"
	"golang.org/x/tools/go/analysis"
)

const AZSD004Doc = `check that computed attributes should only contain computed-only nested schemas

The AZSD004 analyzer checks that schema fields marked as Computed should not
declare ValidateFunc, and their nested schemas should also be computed-only
(no Required/Optional fields).

Example violations:
  "computed_field": {
      Type:         schema.TypeString,
      Computed:     true,
      ValidateFunc: validation.StringIsNotEmpty, // Invalid: computed fields don't need validation
  }

  "computed_list": {
      Type:     schema.TypeList,
      Computed: true,
      Elem: &schema.Resource{
          Schema: map[string]*schema.Schema{
              "property": {
                  Type:     schema.TypeString,
                  Required: true, // Invalid: nested schemas in computed attributes should be computed-only
              },
          },
      },
  }

Valid usage:
  "computed_field": {
      Type:     schema.TypeString,
      Computed: true,
  }

  "computed_list": {
      Type:     schema.TypeList,
      Computed: true,
      Elem: &schema.Resource{
          Schema: map[string]*schema.Schema{
              "property": {
                  Type:     schema.TypeString,
                  Computed: true, // Correct: nested schemas should be computed-only
              },
          },
      },
  }`

const azsd004Name = "AZSD004"

var AZSD004Analyzer = &analysis.Analyzer{
	Name:     azsd004Name,
	Doc:      AZSD004Doc,
	Run:      runAZSD004,
	Requires: []*analysis.Analyzer{localschema.LocalAnalyzer, commentignore.Analyzer},
}

func runAZSD004(pass *analysis.Pass) (interface{}, error) {
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

		if ignorer.ShouldIgnore(azsd004Name, schemaLit) {
			continue
		}

		if !schemaInfo.Schema.Computed || schemaInfo.Schema.Optional || schemaInfo.Schema.Required {
			continue
		}

		checkSchemaForViolations(pass, schemaInfo)

		if schemaInfo.DeclaresField(schema.SchemaFieldElem) {
			checkElemChildren(pass, schemaInfo)
		}
	}

	return nil, nil
}

// checkSchemaForViolations checks a single schema for ValidateFunc and Required/Optional violations
func checkSchemaForViolations(pass *analysis.Pass, schemaInfo *schema.SchemaInfo) {
	if schemaInfo == nil {
		return
	}

	var pos token.Position
	
	if schemaInfo.DeclaresField(schema.SchemaFieldValidateFunc) {
		validateKV := schemaInfo.Fields[schema.SchemaFieldValidateFunc]
		if validateKV != nil {
			pos = pass.Fset.Position(validateKV.Pos())
		} else {
			return
		}
	} else if schemaInfo.Schema.Required || schemaInfo.Schema.Optional {
		pos = pass.Fset.Position(schemaInfo.AstCompositeLit.Pos())
	} else {
		return
	}

	if loader.ShouldReport(pos.Filename, pos.Line) {
		pass.Reportf(schemaInfo.AstCompositeLit.Pos(), "%s: %s\n",
			azsd004Name, helper.FixedCode("computed attributes should only contain computed-only nested schemas"))
	}
}

// checkElemChildren checks nested schemas in Elem fields
func checkElemChildren(pass *analysis.Pass, schemaInfo *schema.SchemaInfo) {
	elemKV := schemaInfo.Fields[schema.SchemaFieldElem]
	if elemKV == nil {
		return
	}

	compLit := helper.GetResourceSchemaFromElem(elemKV)
	if compLit == nil {
		return
	}

	if helper.IsSchemaSchema(pass.TypesInfo, compLit) {
		childSchemaInfo := schema.NewSchemaInfo(compLit, pass.TypesInfo)
		checkSchemaForViolations(pass, childSchemaInfo)
	} else {
		nestedSchemaMap := helper.GetNestedSchemaMap(compLit)
		if nestedSchemaMap != nil {
			for _, elt := range nestedSchemaMap.Elts {
				if kv, ok := elt.(*ast.KeyValueExpr); ok {
					if schemaLit, ok := kv.Value.(*ast.CompositeLit); ok {
						childSchemaInfo := schema.NewSchemaInfo(schemaLit, pass.TypesInfo)
						checkSchemaForViolations(pass, childSchemaInfo)
					}
				}
			}
		}
	}
}
