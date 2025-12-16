package AZBP002

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/bflad/tfproviderlint/helper/terraformtype/helper/schema"
	"github.com/qixialu/azurerm-linter/passes/changedlines"
	"github.com/qixialu/azurerm-linter/passes/helpers/schemafields"
	"github.com/qixialu/azurerm-linter/passes/util"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `check Optional+Computed fields follow conventions

The AZBP002 analyzer checks that fields marked as both Optional and Computed:
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

const analyzerName = "AZBP002"

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

	// Get the shared inspector
	inspector, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, nil
	}

	// Pre-filter: only look at CompositeLit nodes
	nodeFilter := []ast.Node{
		(*ast.CompositeLit)(nil),
	}

	// Build file comments map for all files
	fileCommentsMap := make(map[string][]*ast.CommentGroup)
	for _, f := range pass.Files {
		filename := pass.Fset.Position(f.Pos()).Filename
		fileCommentsMap[filename] = f.Comments
	}

	inspector.Preorder(nodeFilter, func(n ast.Node) {
		comp, ok := n.(*ast.CompositeLit)
		if !ok {
			return
		}

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

			// Use tfproviderlint's SchemaInfo to analyze the schema
			schemaInfo := schema.NewSchemaInfo(schemaLit, pass.TypesInfo)

			// Only check fields that are both Optional and Computed
			if !schemaInfo.Schema.Optional || !schemaInfo.Schema.Computed {
				continue
			}

			// Get positions of Optional and Computed fields
			optionalKV := schemaInfo.Fields[schema.SchemaFieldOptional]
			computedKV := schemaInfo.Fields[schema.SchemaFieldComputed]

			if optionalKV == nil || computedKV == nil {
				continue
			}

			optionalPos := optionalKV.Pos()
			computedPos := computedKV.Pos()

			// Check order: Optional should come before Computed
			if optionalPos > computedPos {
				pos := pass.Fset.Position(kv.Pos())
				if changedlines.ShouldReport(pos.Filename, pos.Line) {
					pass.Reportf(kv.Pos(), "%s: field %q has %s and %s in wrong order (%s must come before %s)",
						analyzerName, fieldName,
						util.FixedCode("Optional"), util.IssueLine("Computed"),
						util.FixedCode("Optional"), util.IssueLine("Computed"))
				}
				continue
			}

			// Check for NOTE: O+C comment between Optional and Computed
			hasOCComment := false
			optionalLine := pass.Fset.Position(optionalPos).Line
			computedLine := pass.Fset.Position(computedPos).Line

			// Look for comments between Optional and Computed lines
			comments := fileCommentsMap[filename]
			for _, cg := range comments {
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
				pos := pass.Fset.Position(kv.Pos())
				if changedlines.ShouldReport(pos.Filename, pos.Line) {
					pass.Reportf(kv.Pos(), "%s: field %q is Optional+Computed but missing required comment. Add %s between Optional and Computed\n",
						analyzerName, fieldName, util.FixedCode("'// NOTE: O+C - <explanation>'"))
				}
			}
		}
	})

	return nil, nil
}
