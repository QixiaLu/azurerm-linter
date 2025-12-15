# AZBP002

The AZBP002 analyzer reports when schema properties are marked as both `Optional` and `Computed` without proper documentation explaining why this pattern is necessary.

Reference: [Best Practices Guide](https://github.com/hashicorp/terraform-provider-azurerm/blob/main/contributing/topics/best-practices.md#setting-properties-to-optional--computed)

## Flagged Code

```go
"etag": {
    Type: pluginsdk.TypeString,
    Optional: true,
    Computed: true,
},
```

```go
"create_mode": {
    Type: pluginsdk.TypeString,
    Optional: true,
    Computed: true,  // Missing explanatory comment
},
```

## Passing Code

```go
"etag": {
    Type: pluginsdk.TypeString,
    Optional: true,
    // NOTE: O+C Azure generates a new value every time this resource is updated
    Computed: true,
},
```

```go
"create_mode": {
    Type: pluginsdk.TypeString,
    Optional: true,
    // NOTE: O+C Azure API defaults this to "Default" when not specified
    Computed: true,
},
```
