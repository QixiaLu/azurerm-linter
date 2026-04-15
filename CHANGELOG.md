# Changelog

## Unreleased

- Add `--output json` for machine-readable results with scope and summary metadata
- Add `--version` output using build info when available
- Centralize filtered-mode diagnostic decisions in the runner instead of analyzer-local line gating
- Normalize retained upstream rule and loader behavior to the new metadata-based filtered-mode model instead of restoring the previous analyzer-local filtering semantics
- Make diff filtering hunk-aware so structural diagnostics can survive deletion-only hunks when tied to changed evidence
- Add diagnostic evidence metadata so filtering can distinguish report location from the lines that justify keeping a diagnostic
- Reduce false negatives in `AZNR005` when registration literals are assigned to local variables before being returned
- Keep `AZNR005` whole-literal sorting validation so grouped registration blocks still report global ordering issues in filtered mode
- Reduce false negatives in `AZBP008` and `AZSD003` when schema values are provided through local variable-backed composite literals
- Restore `lintignore` support for `AZBP003`, `AZBP004`, `AZBP005`, and `AZRE001`
- Restore local untracked service files in local git filtered mode
- Deprecate `AZNR007` and relax it to skip interpolated names and excluded configuration resources

## v0.1.8 (2026-04-14)

- Include local untracked changes in diff analysis (#52)

## v0.1.7 (2026-04-09)

- Deprecate AZBP015 (#48)

## v0.1.6 (2026-04-09)

- Update rules for edge cases (#47)
- Add `--no-ext-diff` to `git diff` command to prevent external diff tools from interfering (#46)
- Update README for Go version consistency (#45)

## v0.1.5 (2026-03-19)

NEW RULES:

- AZBP012: check for unnecessary else blocks that can be avoided by setting a default
- AZBP013: check for chained nil checks that should be split into separate if statements
- AZBP014: check for empty `OperationOptions` literals when a `Default*` constructor exists
- AZBP015: check that `check.That().Key().HasValue()` is unnecessary when `ImportStep` is used
- AZNR007: check that resource names in test configurations start with `"acctest"`
- AZNR008: check for hardcoded resource IDs in test configurations

## v0.1.4 (2026-02-11)

- Deprecate AZNR003 as it's not enforced (#17)

## v0.1.3 (2026-02-10)

NEW RULES:

- AZBP009: check for variables that use the same name as an imported package
- AZBP010: check for variables that are declared and immediately returned
- AZBP011: check for `strings.EqualFold` usage in enum comparisons
- AZNR006: check that nil checks are performed inside `flatten*` methods

## v0.1.2 (2026-02-06)

NEW RULES:

- AZBP006: check for redundant `nil` assignments to pointer fields in struct literals
- AZBP007: check for string slices initialized using `make([]string, 0)` instead of `[]string{}`
- AZBP008: check for `ValidateFunc` uses `PossibleValuesFor*` instead of manual enum listing
- AZNR004: check for `flatten*` functions returning slices don't return `nil`
- AZNR005: check for registrations are sorted alphabetically
- AZSD003: check for redundant use of both `ExactlyOneOf` and `ConflictsWith`
- AZSD004: check for `computed` attributes should only have computed-only nested schema

## v0.1.1 (2026-01-28)

- Update AZBP005 to include test file validation (#14)

## v0.1.0 (2026-01-20)

Initial release.

RULES:

- AZBP001: check for all String arguments have `ValidateFunc`
- AZBP002: check for `Optional+Computed` fields follow conventions
- AZBP003: check for `pointer.ToEnum` to convert Enum type instead of explicitly type conversion
- AZBP004: check for zero-value initialization followed by nil check and pointer dereference that should use `pointer.From`
- AZBP005: check that Go source files have the correct licensing header
- AZNR001: check for Schema field ordering
- AZNR002: check for top-level updatable arguments are included in Update func
- AZNR003: check for `expand*`/`flatten*` functions are defined as receiver methods
- AZRE001: check for fixed error strings using `fmt.Errorf` instead of `errors.New`
- AZRN001: check for percentage properties use `_percentage` suffix instead of `_in_percent`
- AZSD001: check for `MaxItems:1` blocks with single property should be flattened
- AZSD002: check for `AtLeastOneOf` or `ExactlyOneOf` validation on TypeList fields with all optional nested fields
