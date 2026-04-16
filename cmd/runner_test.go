package cmd

import (
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/qixialu/azurerm-linter/loader"
	"github.com/qixialu/azurerm-linter/reporting"
)

func resetChangesForTest(t *testing.T) {
	t.Helper()
	if _, err := loader.LoadChanges(loader.LoaderOptions{NoFilter: true}); err != nil {
		t.Fatalf("LoadChanges() cleanup error = %v", err)
	}
}

func TestShouldKeepDiagnosticUsesMetadataBackedFiltering(t *testing.T) {
	reporting.Reset()
	diffPath := filepath.Join(t.TempDir(), "deletion_only.diff")
	diff := `diff --git a/internal/services/cdn/registration.go b/internal/services/cdn/registration.go
index 1111111..2222222 100644
--- a/internal/services/cdn/registration.go
+++ b/internal/services/cdn/registration.go
@@ -37,4 +37,3 @@ func (r Registration) SupportedResources() map[string]*pluginsdk.Resource {
-//lintignore:AZNR005 temporary exemption
 	return map[string]*pluginsdk.Resource{
 		"azurerm_managed_disk":     nil,
 		"azurerm_availability_set": nil,
`

	if err := os.WriteFile(diffPath, []byte(diff), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := loader.LoadChanges(loader.LoaderOptions{DiffFile: diffPath}); err != nil {
		t.Fatalf("LoadChanges() error = %v", err)
	}
	t.Cleanup(func() {
		resetChangesForTest(t)
		reporting.Reset()
	})

	file := filepath.Join("repo", "internal", "services", "cdn", "registration.go")
	message := "AZNR005: registrations should be sorted alphabetically\n"
	reporting.Record(reporting.DiagnosticMeta{
		PkgPath:       "github.com/qixialu/azurerm-linter/passes",
		Message:       message,
		ReportFile:    file,
		ReportLine:    37,
		ReportColumn:  1,
		EvidenceFile:  file,
		EvidenceLines: []int{37, 38, 39},
		MatchMode:     reporting.MatchModeSameHunk,
	})

	if !shouldKeepDiagnostic("github.com/qixialu/azurerm-linter/passes", token.Position{Filename: file, Line: 37, Column: 1}, message) {
		t.Fatalf("shouldKeepDiagnostic() = false, want true for same-hunk evidence")
	}

	reporting.Record(reporting.DiagnosticMeta{
		PkgPath:       "github.com/qixialu/azurerm-linter/passes",
		Message:       "AZNR005: unrelated\n",
		ReportFile:    file,
		ReportLine:    37,
		ReportColumn:  1,
		EvidenceFile:  file,
		EvidenceLines: []int{1},
		MatchMode:     reporting.MatchModeSameHunk,
	})

	if shouldKeepDiagnostic("github.com/qixialu/azurerm-linter/passes", token.Position{Filename: file, Line: 37, Column: 1}, "AZNR005: unrelated\n") {
		t.Fatalf("shouldKeepDiagnostic() = true, want false for unrelated evidence")
	}
}

func TestShouldKeepDiagnosticDefaultsToTrueWithoutMetadata(t *testing.T) {
	if !shouldKeepDiagnostic("github.com/qixialu/azurerm-linter/passes", token.Position{Filename: "internal/services/cdn/registration.go", Line: 1, Column: 1}, "message") {
		t.Fatalf("shouldKeepDiagnostic() = false, want true when metadata is absent")
	}
}

func TestShouldKeepDiagnosticUsesNewFileMetadata(t *testing.T) {
	reporting.Reset()
	diffPath := filepath.Join(t.TempDir(), "new_file.diff")
	diff := `diff --git a/internal/services/cdn/new_resource.go b/internal/services/cdn/new_resource.go
new file mode 100644
index 0000000..2222222
--- /dev/null
+++ b/internal/services/cdn/new_resource.go
@@ -0,0 +1,3 @@
+package cdn
+
+func resource() {}
`

	if err := os.WriteFile(diffPath, []byte(diff), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := loader.LoadChanges(loader.LoaderOptions{DiffFile: diffPath}); err != nil {
		t.Fatalf("LoadChanges() error = %v", err)
	}
	t.Cleanup(func() {
		resetChangesForTest(t)
		reporting.Reset()
	})

	file := filepath.Join("repo", "internal", "services", "cdn", "new_resource.go")
	message := "AZNR001: schema fields are out of order\n"
	reporting.Record(reporting.DiagnosticMeta{
		PkgPath:       "github.com/qixialu/azurerm-linter/passes",
		Message:       message,
		ReportFile:    file,
		ReportLine:    2,
		ReportColumn:  1,
		EvidenceFile:  file,
		EvidenceLines: []int{2},
		MatchMode:     reporting.MatchModeNewFile,
	})

	if !shouldKeepDiagnostic("github.com/qixialu/azurerm-linter/passes", token.Position{Filename: file, Line: 2, Column: 1}, message) {
		t.Fatalf("shouldKeepDiagnostic() = false, want true for new-file metadata")
	}

	otherFile := filepath.Join("repo", "internal", "services", "cdn", "existing_resource.go")
	otherMessage := "AZNR001: unchanged file\n"
	reporting.Record(reporting.DiagnosticMeta{
		PkgPath:       "github.com/qixialu/azurerm-linter/passes",
		Message:       otherMessage,
		ReportFile:    otherFile,
		ReportLine:    2,
		ReportColumn:  1,
		EvidenceFile:  otherFile,
		EvidenceLines: []int{2},
		MatchMode:     reporting.MatchModeNewFile,
	})

	if shouldKeepDiagnostic("github.com/qixialu/azurerm-linter/passes", token.Position{Filename: otherFile, Line: 2, Column: 1}, otherMessage) {
		t.Fatalf("shouldKeepDiagnostic() = true, want false for non-new file metadata")
	}
}

func TestShouldKeepDiagnosticUsesExactAddedEvidenceLine(t *testing.T) {
	reporting.Reset()
	diffPath := filepath.Join(t.TempDir(), "added_line.diff")
	diff := `diff --git a/internal/services/cdn/registration.go b/internal/services/cdn/registration.go
index 1111111..2222222 100644
--- a/internal/services/cdn/registration.go
+++ b/internal/services/cdn/registration.go
@@ -20,1 +20,1 @@
-	return resources
+	return registration.Resources()
`

	if err := os.WriteFile(diffPath, []byte(diff), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := loader.LoadChanges(loader.LoaderOptions{DiffFile: diffPath}); err != nil {
		t.Fatalf("LoadChanges() error = %v", err)
	}
	t.Cleanup(func() {
		resetChangesForTest(t)
		reporting.Reset()
	})

	file := filepath.Join("repo", "internal", "services", "cdn", "registration.go")
	message := "AZNR002: evidence on added line\n"
	reporting.Record(reporting.DiagnosticMeta{
		PkgPath:       "github.com/qixialu/azurerm-linter/passes",
		Message:       message,
		ReportFile:    file,
		ReportLine:    80,
		ReportColumn:  1,
		EvidenceFile:  file,
		EvidenceLines: []int{19, 20, 21},
		MatchMode:     reporting.MatchModeExactAdded,
	})

	if !shouldKeepDiagnostic("github.com/qixialu/azurerm-linter/passes", token.Position{Filename: file, Line: 80, Column: 1}, message) {
		t.Fatalf("shouldKeepDiagnostic() = false, want true for exact-added evidence")
	}

	otherMessage := "AZNR002: unrelated evidence\n"
	reporting.Record(reporting.DiagnosticMeta{
		PkgPath:       "github.com/qixialu/azurerm-linter/passes",
		Message:       otherMessage,
		ReportFile:    file,
		ReportLine:    81,
		ReportColumn:  1,
		EvidenceFile:  file,
		EvidenceLines: []int{1},
		MatchMode:     reporting.MatchModeExactAdded,
	})

	if shouldKeepDiagnostic("github.com/qixialu/azurerm-linter/passes", token.Position{Filename: file, Line: 81, Column: 1}, otherMessage) {
		t.Fatalf("shouldKeepDiagnostic() = true, want false for non-added evidence")
	}
}

func TestShouldKeepDiagnosticUsesCrossFileEvidenceMetadata(t *testing.T) {
	reporting.Reset()
	diffPath := filepath.Join(t.TempDir(), "schema_only.diff")
	diff := `diff --git a/internal/services/containers/schema.go b/internal/services/containers/schema.go
index 1111111..2222222 100644
--- a/internal/services/containers/schema.go
+++ b/internal/services/containers/schema.go
@@ -44,0 +45,1 @@
+\t\"name\": schema.StringAttribute{Optional: true},
`

	if err := os.WriteFile(diffPath, []byte(diff), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := loader.LoadChanges(loader.LoaderOptions{DiffFile: diffPath}); err != nil {
		t.Fatalf("LoadChanges() error = %v", err)
	}
	t.Cleanup(func() {
		resetChangesForTest(t)
		reporting.Reset()
	})

	reportFile := filepath.Join("repo", "internal", "services", "containers", "resource.go")
	evidenceFile := filepath.Join("repo", "internal", "services", "containers", "schema.go")
	message := "AZNR002: updatable property `name` is not handled in Update function\n"
	reporting.Record(reporting.DiagnosticMeta{
		PkgPath:       "github.com/qixialu/azurerm-linter/passes",
		Message:       message,
		ReportFile:    reportFile,
		ReportLine:    180,
		ReportColumn:  1,
		EvidenceFile:  evidenceFile,
		EvidenceLines: []int{45},
		MatchMode:     reporting.MatchModeExactAdded,
	})

	if !shouldKeepDiagnostic("github.com/qixialu/azurerm-linter/passes", token.Position{Filename: reportFile, Line: 180, Column: 1}, message) {
		t.Fatalf("shouldKeepDiagnostic() = false, want true when evidence file changed even if report file did not")
	}

	otherMessage := "AZNR002: unrelated schema property\n"
	reporting.Record(reporting.DiagnosticMeta{
		PkgPath:       "github.com/qixialu/azurerm-linter/passes",
		Message:       otherMessage,
		ReportFile:    reportFile,
		ReportLine:    181,
		ReportColumn:  1,
		EvidenceFile:  evidenceFile,
		EvidenceLines: []int{12},
		MatchMode:     reporting.MatchModeExactAdded,
	})

	if shouldKeepDiagnostic("github.com/qixialu/azurerm-linter/passes", token.Position{Filename: reportFile, Line: 181, Column: 1}, otherMessage) {
		t.Fatalf("shouldKeepDiagnostic() = true, want false when only the report file differs from the evidence file")
	}
}

func TestShouldKeepDiagnosticUsesStructuralEvidenceLinesAwayFromReportPosition(t *testing.T) {
	reporting.Reset()
	diffPath := filepath.Join(t.TempDir(), "enum_values.diff")
	diff := `diff --git a/internal/services/network/validate.go b/internal/services/network/validate.go
index 1111111..2222222 100644
--- a/internal/services/network/validate.go
+++ b/internal/services/network/validate.go
@@ -54,0 +55,3 @@
+	string(network.RuleTypeAllow),
+	string(network.RuleTypeDeny),
+	string(network.RuleTypeAudit),
`

	if err := os.WriteFile(diffPath, []byte(diff), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := loader.LoadChanges(loader.LoaderOptions{DiffFile: diffPath}); err != nil {
		t.Fatalf("LoadChanges() error = %v", err)
	}
	t.Cleanup(func() {
		resetChangesForTest(t)
		reporting.Reset()
	})

	file := filepath.Join("repo", "internal", "services", "network", "validate.go")
	message := "AZBP008: use network.PossibleValuesForRuleType() instead of manually listing enum values\n"
	reporting.Record(reporting.DiagnosticMeta{
		PkgPath:       "github.com/qixialu/azurerm-linter/passes",
		Message:       message,
		ReportFile:    file,
		ReportLine:    92,
		ReportColumn:  1,
		EvidenceFile:  file,
		EvidenceLines: []int{55, 56, 57},
		MatchMode:     reporting.MatchModeExactAdded,
	})

	if !shouldKeepDiagnostic("github.com/qixialu/azurerm-linter/passes", token.Position{Filename: file, Line: 92, Column: 1}, message) {
		t.Fatalf("shouldKeepDiagnostic() = false, want true when structural evidence lines changed away from report position")
	}

	otherMessage := "AZBP008: unrelated enum listing\n"
	reporting.Record(reporting.DiagnosticMeta{
		PkgPath:       "github.com/qixialu/azurerm-linter/passes",
		Message:       otherMessage,
		ReportFile:    file,
		ReportLine:    93,
		ReportColumn:  1,
		EvidenceFile:  file,
		EvidenceLines: []int{54},
		MatchMode:     reporting.MatchModeExactAdded,
	})

	if shouldKeepDiagnostic("github.com/qixialu/azurerm-linter/passes", token.Position{Filename: file, Line: 93, Column: 1}, otherMessage) {
		t.Fatalf("shouldKeepDiagnostic() = true, want false when structural evidence lines were not added")
	}
}

func TestShouldKeepDiagnosticFallsBackToReportFileWhenEvidenceFileMissing(t *testing.T) {
	reporting.Reset()
	diffPath := filepath.Join(t.TempDir(), "single_line.diff")
	diff := `diff --git a/internal/services/storage/errors.go b/internal/services/storage/errors.go
index 1111111..2222222 100644
--- a/internal/services/storage/errors.go
+++ b/internal/services/storage/errors.go
@@ -11,1 +11,1 @@
-	return fmt.Errorf("bad request")
+	return errors.New("bad request")
`

	if err := os.WriteFile(diffPath, []byte(diff), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := loader.LoadChanges(loader.LoaderOptions{DiffFile: diffPath}); err != nil {
		t.Fatalf("LoadChanges() error = %v", err)
	}
	t.Cleanup(func() {
		resetChangesForTest(t)
		reporting.Reset()
	})

	file := filepath.Join("repo", "internal", "services", "storage", "errors.go")
	message := "AZRE001: fixed error strings should use errors.New() instead of fmt.Errorf()\n"
	reporting.Record(reporting.DiagnosticMeta{
		PkgPath:      "github.com/qixialu/azurerm-linter/passes",
		Message:      message,
		ReportFile:   file,
		ReportLine:   11,
		ReportColumn: 1,
		MatchMode:    reporting.MatchModeExactAdded,
	})

	if !shouldKeepDiagnostic("github.com/qixialu/azurerm-linter/passes", token.Position{Filename: file, Line: 11, Column: 1}, message) {
		t.Fatalf("shouldKeepDiagnostic() = false, want true when evidence file falls back to the report file")
	}
}

func TestShouldKeepDiagnosticDropsAZBP005LineOneDiagnosticForDeletionOnlyDiff(t *testing.T) {
	reporting.Reset()
	diffPath := filepath.Join(t.TempDir(), "deletion_only.diff")
	diff := `diff --git a/internal/services/cdnazbp005/registration.go b/internal/services/cdnazbp005/registration.go
index 1111111..2222222 100644
--- a/internal/services/cdnazbp005/registration.go
+++ b/internal/services/cdnazbp005/registration.go
@@ -12,1 +11,0 @@
-//lintignore:AZNR005 temporary exemption
`

	if err := os.WriteFile(diffPath, []byte(diff), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := loader.LoadChanges(loader.LoaderOptions{DiffFile: diffPath}); err != nil {
		t.Fatalf("LoadChanges() error = %v", err)
	}
	t.Cleanup(func() {
		resetChangesForTest(t)
		reporting.Reset()
	})

	file := filepath.Join("repo", "internal", "services", "cdnazbp005", "registration.go")
	message := "AZBP005: missing license header. Add at the beginning:\n// Copyright IBM Corp. 2014, 2025\n// SPDX-License-Identifier: MPL-2.0\n"
	reporting.Record(reporting.DiagnosticMeta{
		PkgPath:       "github.com/qixialu/azurerm-linter/passes",
		Message:       message,
		ReportFile:    file,
		ReportLine:    1,
		ReportColumn:  1,
		EvidenceFile:  file,
		EvidenceLines: []int{1},
		MatchMode:     reporting.MatchModeExactAdded,
	})

	if shouldKeepDiagnostic("github.com/qixialu/azurerm-linter/passes", token.Position{Filename: file, Line: 1, Column: 1}, message) {
		t.Fatalf("shouldKeepDiagnostic() = true, want false for unrelated line-1 header issue")
	}
}
