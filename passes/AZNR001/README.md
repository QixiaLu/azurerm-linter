# AZNR001

The AZNR001 analyzer reports when schema fields are not ordered according to the provider's conventions.

Reference: [Guide: New Resource](https://github.com/hashicorp/terraform-provider-azurerm/blob/main/contributing/topics/guide-new-resource.md)

## Required Field Order

Schema fields must be ordered as follows:

1. **Special ID fields** - Specifically checked in order:
   - `name` (if present, must be first)
   - `resource_group_name` (if present, must come after `name`)
2. **Location field** - The `location` field (must come after the special ID fields)
3. **Required fields** - Sorted alphabetically
4. **Optional fields** - Sorted alphabetically  
5. **Computed fields** - Sorted alphabetically (in typed resources, these should be in the `Attributes()` method)

**Note:** This analyzer only checks the ordering of `name`, `resource_group_name`, and `location` fields. Other ID fields and required fields ordering in top level are not validated.

## Flagged Code

### Top-Level Schema Violations

```go
// Incorrect: location comes before name
func (r SomeResource) Arguments() map[string]*pluginsdk.Schema {
    return map[string]*pluginsdk.Schema{
        "location": commonschema.Location(),
        
        "name": {
            Type:     pluginsdk.TypeString,
            Required: true,
        },
    }
}
```

```go
// Incorrect: resource_group_name comes before name
func (r SomeResource) Arguments() map[string]*pluginsdk.Schema {
    return map[string]*pluginsdk.Schema{
        "resource_group_name": commonschema.ResourceGroupName(),
        
        "name": {
            Type:     pluginsdk.TypeString,
            Required: true,
        },

        "location": commonschema.Location(),
    }
}
```

### Nested Schema Violations

```go
// Incorrect: nested required fields not sorted alphabetically
"config": {
    Type:     pluginsdk.TypeList,
    Required: true,
    Elem: &pluginsdk.Resource{
        Schema: map[string]*pluginsdk.Schema{
            "sku_name": {           // Should come after "capacity"
                Type:     pluginsdk.TypeString,
                Required: true,
            },
            "capacity": {
                Type:     pluginsdk.TypeInt,
                Required: true,
            },
        },
    },
}
```

## Passing Code

### Top-Level Schema

```go
// Correct: name, then resource_group_name, then location
func (r SomeResource) Arguments() map[string]*pluginsdk.Schema {
    return map[string]*pluginsdk.Schema{
        "name": {
            Type:     pluginsdk.TypeString,
            Required: true,
        },

        "resource_group_name": commonschema.ResourceGroupName(),

        "location": commonschema.Location(),

        // Other required fields can be in any order (not checked)
        "sku_name": {
            Type:     pluginsdk.TypeString,
            Required: true,
        },

        "capacity": {
            Type:     pluginsdk.TypeInt,
            Required: true,
        },

        "description": {
            Type:     pluginsdk.TypeString,
            Optional: true,
        },

        "tags": commonschema.Tags(),
    }
}
```

### Nested Schema

```go
// Correct: nested fields sorted alphabetically within each category
"config": {
    Type:     pluginsdk.TypeList,
    Required: true,
    Elem: &pluginsdk.Resource{
        Schema: map[string]*pluginsdk.Schema{
            // Required fields - alphabetically sorted
            "capacity": {
                Type:     pluginsdk.TypeInt,
                Required: true,
            },
            "sku_name": {
                Type:     pluginsdk.TypeString,
                Required: true,
            },
            
            // Optional fields - alphabetically sorted
            "description": {
                Type:     pluginsdk.TypeString,
                Optional: true,
            },
            "tags": {
                Type:     pluginsdk.TypeMap,
                Optional: true,
            },
        },
    },
}
```
