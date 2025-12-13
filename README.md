# Azurerm-linter

A custom linter for the [terraform-provider-azurerm](https://github.com/hashicorp/terraform-provider-azurerm) codebase that enforces best practices and coding standards for Azure Terraform resources and data sources.

## Features

This linter includes several analyzers that check for common issues:

- **AZC001**: Correct error func usage - Checks that pure string error should use `Errors.New()` instead of `fmt.Errorf()`
- **AZC002**: String validation - Checks that TypeString fields have ValidateFunc or ValidateDiagFunc
- **AZC003**: Optional+Computed comments - Validates that O+C fields have proper comment format
- **AZC004**: MaxItems:1 flattening - Suggests flattening blocks with MaxItems:1 and single nested property

## Installation

### Prerequisites

- Go 1.24 or later
- Git

### Build from Source

```bash
git clone https://github.com/QixiaLu/azurerm-linter.git
cd azurerm-linter
go build
```

This will create an `azurerm-linter.exe` executable (on Windows) or `azurerm-linter` (on Linux/macOS).

## Usage

### Basic Usage

Go to Azurerm repo directory:
```bash
cd ./path/to/terraform-provider-azurerm/internal/services/...
```

Under this directory, run linter against all services

```bash
./path/to/azurerm-linter ./internal/...
```

### Run Specific Analyzers

Go to Azurerm repo directory:
```bash
cd ./path/to/terraform-provider-azurerm/internal/services/...
```

Run only specific checks using the analyzer name as a flag:

```bash
# Run only schema ordering check
./path/to/azurerm-linter -AZC001 ./internal/services/compute/...

# Run only string validation check
./path/to/azurerm-linter -AZC002 ./internal/services/...

# Run multiple specific checks
./path/to/azurerm-linter -AZC001 -AZC003 ./internal/...
```
