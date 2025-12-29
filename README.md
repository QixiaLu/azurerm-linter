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

### Azure New Resource Checks

| Check | Description |
|-------|-------------|
| AZNR001 | check for Schema field ordering |

### Azure Naming Rule Checks

| Check | Description |
|-------|-------------|
| AZRN001 | check for percentage properties use `_percentage` suffix instead of `_in_percent` |

### Azure Resource Error Checks

| Check | Description |
|-------|-------------|
| AZRE001 | check for fixed error strings using `fmt.Errorf` instead of `errors.New` |

### Azure Schema Design Checks

| Check | Description |
|-------|-------------|
| AZSD001 | check for `MaxItems:1` blocks with single property should be flattened |
| AZSD002 | check for `AtLeastOneOf` validation on TypeList fields with all optional nested fields |

## Installation

```bash
git clone https://github.com/QixiaLu/azurerm-linter.git
cd azurerm-linter
go build -o <path/to/terraform-provider-azurerm>
```

## Usage

### Quick Start

```bash
# Run in terraform-provider-azurerm directory
cd /path/to/terraform-provider-azurerm

# Check your local branch changes (auto-detect changed lines and packages)
./azurerm-linter.exe

# Check specific PR (fetch PR branch and create worktree in tmp)
./azurerm-linter.exe --pr=12345

# Check from diff file
./azurerm-linter.exe --diff=changes.txt

# Check specific packages
./azurerm-linter.exe ./internal/services/compute/...

# Check all lines in all packages (no filtering)
./azurerm-linter.exe --no-filter ./internal/services/...
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
