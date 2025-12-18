# AZRE001

The AZRE001 analyzer reports when fixed error strings (without format placeholders) use `fmt.Errorf()` instead of `errors.New()`.

## Flagged Code

```go
// Incorrect: using fmt.Errorf for a fixed string
if err := client.Delete(ctx, id); err != nil {
    return fmt.Errorf("deleting resource")
}
```

```go
// Incorrect: no placeholders in the format string
if resp.StatusCode != 200 {
    return fmt.Errorf("unexpected status code")
}
```

## Passing Code

```go
// Correct: using errors.New for fixed strings
if err := client.Delete(ctx, id); err != nil {
    return errors.New("deleting resource")
}
```

```go
// Correct: using fmt.Errorf with format placeholders
if err := client.Delete(ctx, id); err != nil {
    return fmt.Errorf("deleting %s: %+v", id, err)
}
```

```go
// Correct: using fmt.Errorf with multiple values
if resp.StatusCode != 200 {
    return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}
```
