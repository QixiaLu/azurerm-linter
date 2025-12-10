# Azurerm-linter

A custom linter for the [terraform-provider-azurerm](https://github.com/hashicorp/terraform-provider-azurerm) codebase that enforces best practices and coding standards for Azure Terraform resources and data sources.

## Features

This linter includes several analyzers that check for common issues:

- **AZC001**: Schema field ordering - Ensures schema fields follow the correct order (ID fields → location → Required → Optional → Computed)
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

Run all analyzers on a directory:

```bash
./azurerm-linter ./path/to/terraform-provider-azurerm/internal/services/...
```

### Run Specific Analyzers

Run only specific checks using the analyzer name as a flag:

```bash
# Run only schema ordering check
./azurerm-linter -AZC001 ./internal/services/compute/...

# Run only string validation check
./azurerm-linter -AZC002 ./internal/services/...

# Run multiple specific checks
./azurerm-linter -AZC001 -AZC003 ./internal/...
```

### Common Examples

```bash
# Check all services
./azurerm-linter ./internal/services/...

# Check a specific service
./azurerm-linter ./internal/services/compute/...

# Save results to a file
./azurerm-linter ./internal/... > lint-results.txt 2>&1
