# azurerm-linter

The azurerm-linter tool is an AzureRM Provider code linting tool, specifically tailored for checking if the code is consistent with rules defined in `/contributing`.

## Lint Checks

For additional information about each check, see the documentation in passes's directory (e.g., `passes/doc.go`).

### Azure Best Practice Checks

| Check | Description |
|-------|-------------|
| AZBP001 | check for all String arguments have `ValidateFunc` |
| AZBP002 | check for `Optional+Computed` fields follow conventions |
| AZBP003 | check for `pointer.ToEnum` to convert Enum type instead of explicitly type conversion |
| AZBP004 | check for zero-value initialization followed by nil check and pointer dereference that should use `pointer.From` |
| AZBP005 | check that Go source files have the correct licensing header |
| AZBP006 | check for redundant `nil` assignments to pointer fields in struct literals |
| AZBP007 | check for string slices initialized using `make([]string, 0)` instead of `[]string{}` |
| AZBP008 | check for `ValidateFunc` uses `PossibleValuesFor*` instead of manual enum listing |
| AZBP009 | check for variable uses the same name as the imported package |
| AZBP010 | check for variables that are declared and immediately returned |
| AZBP011 | check for `strings.EqualFold` usage in enum comparisons |

### Azure New Resource Checks

| Check | Description | Comments |
|-------|-------------|----------|
| AZNR001 | check for Schema field ordering | When git filter is on, this analyzer only run on newly created resources/data sources |
| AZNR002 | check for top-level updatable arguments are included in Update func | This analyzer currently only runs on typed resource |
| AZNR003 | check for `expand*`/`flatten*` functions are defined as receiver methods |This analyzer currently only runs on typed resource/data source |
| AZNR004 | check for `flatten*` functions returning slices don't return `nil` |
| AZNR005 | check for registrations are sorted alphabetically |

### Azure Naming Rule Checks

| Check | Description |
|-------|-------------|
| AZRN001 | check for percentage properties use `_percentage` suffix instead of `_in_percent` |

### Azure Reference Error Checks

| Check | Description |
|-------|-------------|
| AZRE001 | check for fixed error strings using `fmt.Errorf` instead of `errors.New` |

### Azure Schema Design Checks

| Check | Description |
|-------|-------------|
| AZSD001 | check for `MaxItems:1` blocks with single property should be flattened |
| AZSD002 | check for `AtLeastOneOf` or `ExactlyOneOf` validation on TypeList fields with all optional nested fields |
| AZSD003 | check for redundant use of both `ExactlyOneOf` and `ConflictsWith` |
| AZSD004 | check for `computed` attributes should only have computed-only nested schema |

## Installation

### Prerequisites

This tool must be compiled with the **same Go version** required by `terraform-provider-azurerm`. Check the Go version in `terraform-provider-azurerm/go.mod`.

**Windows users:** Enable long paths to avoid "Filename too long" errors when using `--pr`:
```bash
git config --global core.longpaths true
```

### Build

```bash
go install github.com/qixialu/azurerm-linter@latest
```

This will install the binary to your `$GOPATH/bin` (or `$HOME/go/bin` by default).

## Usage

### Quick Start

```bash
# Run in terraform-provider-azurerm directory
cd /path/to/terraform-provider-azurerm

# Check your local branch changes (auto-detect changed lines and packages)
azurerm-linter

# Check specific PR (fetch PR branch and create worktree in tmp)
azurerm-linter --pr=12345

# Check from diff file
azurerm-linter --diff=changes.txt

# Check specific packages
azurerm-linter ./internal/services/compute/...

# Check all lines in all packages (no filtering)
azurerm-linter --no-filter ./internal/services/...
```

### Common Options

```bash
--pr=<number>      # Check GitHub PR
--remote=<name>    # Specify git remote (origin/upstream)
--base=<branch>    # Specify base branch
--diff=<file>      # Read diff from file
--no-filter        # Analyze all lines (not just changes)
--list             # List all available checks
--help             # Show help
```

**Note**: By default, only changed lines are analyzed. Use `--no-filter` to check everything.

### Output

The tool prints results directly to **standard output (console/terminal)**:

**If issues are found:**
- Each issue is printed with file path, line number, and check ID
- Summary: `Found X issue(s)`
- Exit code: 1

**If no issues are found:**
- Message: `✓ Analysis completed successfully with no issues found`
- Exit code: 0

**If errors occur (e.g., build failures, missing dependencies):**
- Error message with details
- Exit code: 2

#### Example output (with issues)

```bash
azurerm-linter
2026/01/05 10:39:01 Using local git diff mode
2026/01/05 10:39:01 Current branch: lint_test
2026/01/05 10:39:02 Merge-base: 0aac888
2026/01/05 10:39:03 ✓ Found 9 changed files with 1553 changed lines
2026/01/05 10:39:03 Changed lines filter: tracking 9 files with 1553 changed lines
2026/01/05 10:39:03 Auto-detected 1 changed packages:
2026/01/05 10:39:03   ./internal/services/policy
2026/01/05 10:39:03 Loading packages...
2026/01/05 10:40:36 Running analysis...
C:\Users\**\Repos\terraform-provider-azurerm\internal\services\policy\management_group_policy_definition_resource.go:55:19: AZBP001: string argument "display_name" must have ValidateFunc

C:\Users\**\Repos\terraform-provider-azurerm\internal\services\policy\management_group_policy_definition_resource.go:94:18: AZBP002: field "policy_rule" is Optional+Computed but missing required comment. Add '// NOTE: O+C - <explanation>' between Optional and Computed

C:\Users\**\Repos\terraform-provider-azurerm\internal\services\policy\management_group_policy_definition_resource.go:162:19: AZBP003: use `pointer.ToEnum` to convert Enum type instead of explicitly type conversion.

C:\Users\**\Repos\terraform-provider-azurerm\internal\services\policy\management_group_policy_definition_resource.go:309:24: AZBP003: use `pointer.ToEnum` to convert Enum type instead of explicitly type conversion.

C:\Users\**\Repos\terraform-provider-azurerm\internal\services\policy\policy_definition_resource.go:126:17: AZBP003: use `pointer.ToEnum` to convert Enum type instead of explicitly type conversion.

C:\Users\**\Repos\terraform-provider-azurerm\internal\services\policy\policy_definition_resource.go:567:19: AZBP003: use `pointer.ToEnum` to convert Enum type instead of explicitly type conversion.

C:\Users\**\Repos\terraform-provider-azurerm\internal\services\policy\management_group_policy_definition_resource.go:233:6: AZBP004: can simplify with `pointer.From()` since variable is initialized to zero value

C:\Users\**\Repos\terraform-provider-azurerm\internal\services\policy\management_group_policy_definition_resource.go:408:14: AZRE001: fixed error strings should use errors.New() instead of fmt.Errorf()

C:\Users\**\Repos\terraform-provider-azurerm\internal\services\policy\management_group_policy_definition_resource.go:40:9: AZNR001: schema fields are not in the correct order
Expected order:
  name, management_group_id, display_name, mode, policy_type, description, metadata, parameters, policy_rule
Actual order:
  name, management_group_id, display_name, mode, policy_type, metadata, description, policy_rule, parameters

2026/01/05 10:40:40 Found 9 issue(s)
```

## Limitations

Schema-related checks (e.g., AZNR002, AZSD001, AZSD002) analyze schemas defined as `map[string]*pluginsdk.Schema` or `map[string]*schema.Schema` composite literals returned from functions. This includes:
- Direct returns: `return &map[string]*pluginsdk.Schema{...}`
- Variable returns: `output := map[string]*pluginsdk.Schema{...}; return output` (captures initial `:=` definition only, ignoring subsequent `=` modifications)
- Inline schema definitions: `return &pluginsdk.Schema{...}`
- Cross-package function calls: Only `commonschema` package is currently supported (e.g., `commonschema.ResourceGroupName()`)
- Same-package helper functions returning schemas

Schemas defined in other ways (nested blocks) are excluded to reduce false positives from runtime modifications (e.g., conditional properties based on feature flags) that cannot be determined through static analysis.

For detailed limitations of each analyzer, refer to the documentation in the respective analyzer files (e.g., `passes/AZNR002.go`).
