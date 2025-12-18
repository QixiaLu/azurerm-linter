# AZSD001

The AZSD001 analyzer reports when blocks with `MaxItems: 1` contain only a single nested property without proper justification. These should typically be flattened for better user experience.

## Flagged Code

```go
// Incorrect: MaxItems:1 with single property, no explanation
"config": {
    Type:     pluginsdk.TypeList,
    MaxItems: 1,
    Optional: true,
    Elem: &pluginsdk.Resource{
        Schema: map[string]*pluginsdk.Schema{
            "value": {
                Type:     pluginsdk.TypeString,
                Required: true,
            },
        },
    },
}
```

```go
// Incorrect: TypeSet with MaxItems:1 and single property
"settings": {
    Type:     pluginsdk.TypeSet,
    MaxItems: 1,
    Optional: true,
    Elem: &pluginsdk.Resource{
        Schema: map[string]*pluginsdk.Schema{
            "enabled": {
                Type:     pluginsdk.TypeBool,
                Required: true,
            },
        },
    },
}
```

## Passing Code

```go
// Correct: flattened to a single property
"config_value": {
    Type:     pluginsdk.TypeString,
    Optional: true,
}
```

```go
// Correct: MaxItems:1 with multiple properties
"config": {
    Type:     pluginsdk.TypeList,
    MaxItems: 1,
    Optional: true,
    Elem: &pluginsdk.Resource{
        Schema: map[string]*pluginsdk.Schema{
            "value": {
                Type:     pluginsdk.TypeString,
                Required: true,
            },
            "enabled": {
                Type:     pluginsdk.TypeBool,
                Required: true,
            },
        },
    },
}
```

```go
// ✅ Correct: single property with documented justification
"config": {
    Type:     pluginsdk.TypeList,
    MaxItems: 1,
    Optional: true,
    // NOTE: Additional properties will be added in future API versions
    Elem: &pluginsdk.Resource{
        Schema: map[string]*pluginsdk.Schema{
            "value": {
                Type:     pluginsdk.TypeString,
                Required: true,
            },
        },
    },
}
```
