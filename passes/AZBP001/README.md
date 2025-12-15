# AZBP001

The AZBP001 analyzer reports when String type schema fields (`Required` or `Optional`) do not have a `ValidateFunc` defined.

Reference: [guide new fields to resource](https://github.com/hashicorp/terraform-provider-azurerm/blob/main/contributing/topics/guide-new-fields-to-resource.md#schema)

## Flagged Code

```go
"name": {
    Type:     pluginsdk.TypeString,
    Required: true,
    // Missing ValidateFunc!
}
```

```go
"location": {
    Type:     pluginsdk.TypeString,
    Optional: true,
    // Missing ValidateFunc!
}
```

## Passing Code

```go
"name": {
    Type:         pluginsdk.TypeString,
    Required:     true,
    ValidateFunc: validation.StringIsNotEmpty,
}
```

```go
"location": {
    Type:         pluginsdk.TypeString,
    Optional:     true,
    ValidateFunc: validation.StringIsNotEmpty,
}
```

```go
"resource_id": {
    Type:     pluginsdk.TypeString,
    Computed: true,
},
```
