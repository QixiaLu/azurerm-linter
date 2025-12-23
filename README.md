# azurerm-linter

The azurerm-linter tool is an AzureRM Provider code linting tool, specifically tailored for checking if the code is consistent with rules defined in `/contributing`.

## Lint Checks

For additional information about each check, see the documentation in passes's directory (e.g., `passes/doc.go`).

### Azure Best Practice Checks

| Check | Description |
|-------|-------------|
| AZBP001 | check for all String arguments have `ValidationFunc` |
| AZBP002 | check for `Optional+Computed` fields follow conventions |
| AZBP003 | check for `pointer.ToEnum` to convert Enum type instead of explicitly type conversion.

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

## Installation

### Prerequisites

- Go 1.24 or later
- Git

### Build from Source

```bash
git clone https://github.com/QixiaLu/azurerm-linter.git
cd azurerm-linter
go build -o <path/to/terraform-provider-azurerm>
```

This will create an `azurerm-linter.exe` executable (on Windows) or `azurerm-linter` (on Linux/macOS).

## Usage

### Basic Usage

Run linter against all packages:
(For better performance, please narrow down to a single service)

```bash
cd ./path/to/terraform-provider-azurerm
./azurerm-linter -use-git-repo=false ./internal/services/...
```

### Check Only Changed Lines

The linter can analyze only the lines that have been modified.

#### Check Local Git Changes

Check only the lines changed in your current branch compared to the target branch:

```bash
# Auto-detect remote and branch (defaults to upstream/main or origin/main)
./azurerm-linter ./internal/services/policy/...

# Specify remote explicitly
./azurerm-linter -remote=origin ./internal/services/policy/...

# Specify both remote and branch
./azurerm-linter -remote=upstream -branch=main ./internal/services/policy/...

# Specify only the branch (remote will be auto-detected)
./azurerm-linter -branch=main ./internal/services/policy/...
```

#### Check GitHub Pull Request

Check only the lines changed in a specific pull request:

```bash
# Use GitHub PR number
./azurerm-linter -pr-number=1234 -use-github-api=true ./internal/services/policy/...

# Specify repository if not using default
./azurerm-linter -pr-number=1234 ./internal/services/policy/...
```

#### Check from Diff File

Check lines from a git diff file:

```bash
# Generate and use a diff file
git diff main > changes.patch
./azurerm-linter -diff-file=changes.patch ./internal/services/policy/...
```

**Note**: When using any of the changed-line detection modes (`-remote`, `-pr`, `-diff-file`), the linter will only report issues on lines that were modified, making it easier to focus on reviewing new changes without noise from existing code.
