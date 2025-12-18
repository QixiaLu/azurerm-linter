# AZRN001

The AZRN001 analyzer reports when percentage properties use `_in_percent` suffix instead of the standardized `_percentage` suffix.

## Flagged Code

```go
// Incorrect: using _in_percent suffix
"cpu_threshold_in_percent": {
    Type:     pluginsdk.TypeInt,
    Required: true,
}
```

```go
// Incorrect: using _in_percent in nested schema
"scaling_policy": {
    Type:     pluginsdk.TypeList,
    Optional: true,
    Elem: &pluginsdk.Resource{
        Schema: map[string]*pluginsdk.Schema{
            "target_utilization_in_percent": {
                Type:     pluginsdk.TypeInt,
                Required: true,
            },
        },
    },
}
```

## Passing Code

```go
// Correct: using _percentage suffix
"cpu_threshold_percentage": {
    Type:     pluginsdk.TypeInt,
    Required: true,
}
```

```go
// Correct: using _percentage in nested schema
"scaling_policy": {
    Type:     pluginsdk.TypeList,
    Optional: true,
    Elem: &pluginsdk.Resource{
        Schema: map[string]*pluginsdk.Schema{
            "target_utilization_percentage": {
                Type:     pluginsdk.TypeInt,
                Required: true,
            },
        },
    },
}
```
