package loader

import (
	"path/filepath"
	"testing"

	"github.com/qixialu/azurerm-linter/reporting"
)

func TestChangeSetShouldKeepDiagnosticExactAddedTracksOnlyAddedLines(t *testing.T) {
	cs := NewChangeSet()

	diff := `diff --git a/internal/services/cdn/registration.go b/internal/services/cdn/registration.go
index 1111111..2222222 100644
--- a/internal/services/cdn/registration.go
+++ b/internal/services/cdn/registration.go
@@ -37,1 +37,0 @@
-//lintignore:AZNR005 temporary exemption
`

	if err := cs.parseDiffOutput(diff); err != nil {
		t.Fatalf("parseDiffOutput() error = %v", err)
	}

	file := filepath.Join("repo", "internal", "services", "cdn", "registration.go")

	if !cs.IsFileChanged(file) {
		t.Fatalf("expected file to be marked changed")
	}

	if got := cs.getTotalChangedLines(); got != 0 {
		t.Fatalf("getTotalChangedLines() = %d, want 0", got)
	}

	if cs.ShouldKeepDiagnostic(reporting.DiagnosticMeta{
		ReportFile:    file,
		EvidenceFile:  file,
		EvidenceLines: []int{37},
		MatchMode:     reporting.MatchModeExactAdded,
	}) {
		t.Fatalf("ShouldKeepDiagnostic() = true, want false for deletion-only line")
	}
	if cs.ShouldKeepDiagnostic(reporting.DiagnosticMeta{
		ReportFile:    file,
		EvidenceFile:  file,
		EvidenceLines: []int{1},
		MatchMode:     reporting.MatchModeExactAdded,
	}) {
		t.Fatalf("ShouldKeepDiagnostic() = true, want false for unrelated line")
	}
	if cs.ShouldKeepDiagnostic(reporting.DiagnosticMeta{
		ReportFile:    filepath.Join("repo", "internal", "services", "dns", "registration.go"),
		EvidenceFile:  filepath.Join("repo", "internal", "services", "dns", "registration.go"),
		EvidenceLines: []int{37},
		MatchMode:     reporting.MatchModeExactAdded,
	}) {
		t.Fatalf("ShouldKeepDiagnostic() = true for unchanged file, want false")
	}
}

func TestChangeSetShouldKeepDiagnosticExactAddedRemainsLineScoped(t *testing.T) {
	cs := NewChangeSet()
	file := "internal/services/cdn/registration.go"
	cs.changedFiles[file] = true
	cs.changedLines[file] = map[int]bool{42: true}

	fullPath := filepath.Join("repo", "internal", "services", "cdn", "registration.go")

	if !cs.ShouldKeepDiagnostic(reporting.DiagnosticMeta{
		ReportFile:    fullPath,
		EvidenceFile:  fullPath,
		EvidenceLines: []int{42},
		MatchMode:     reporting.MatchModeExactAdded,
	}) {
		t.Fatalf("ShouldKeepDiagnostic() = false, want true for exact added line")
	}
	if cs.ShouldKeepDiagnostic(reporting.DiagnosticMeta{
		ReportFile:    fullPath,
		EvidenceFile:  fullPath,
		EvidenceLines: []int{41},
		MatchMode:     reporting.MatchModeExactAdded,
	}) {
		t.Fatalf("ShouldKeepDiagnostic() = true, want false for unchanged line")
	}
}

func TestChangeSetShouldKeepDiagnosticMatchesDeletionHunkContext(t *testing.T) {
	cs := NewChangeSet()

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

	if err := cs.parseDiffOutput(diff); err != nil {
		t.Fatalf("parseDiffOutput() error = %v", err)
	}

	file := filepath.Join("repo", "internal", "services", "cdn", "registration.go")

	if cs.ShouldKeepDiagnostic(reporting.DiagnosticMeta{
		ReportFile:    file,
		EvidenceFile:  file,
		EvidenceLines: []int{37},
		MatchMode:     reporting.MatchModeExactAdded,
	}) {
		t.Fatalf("ShouldKeepDiagnostic() = true, want false for deletion-only exact-added evidence")
	}
	meta := reporting.DiagnosticMeta{
		ReportFile:    file,
		EvidenceFile:  file,
		EvidenceLines: []int{37, 38, 39},
		MatchMode:     reporting.MatchModeSameHunk,
	}
	if !cs.ShouldKeepDiagnostic(meta) {
		t.Fatalf("ShouldKeepDiagnostic() = false, want true for same-hunk evidence")
	}

	unrelated := reporting.DiagnosticMeta{
		ReportFile:    file,
		EvidenceFile:  file,
		EvidenceLines: []int{1},
		MatchMode:     reporting.MatchModeSameHunk,
	}
	if cs.ShouldKeepDiagnostic(unrelated) {
		t.Fatalf("ShouldKeepDiagnostic() = true, want false for unrelated evidence line")
	}
}
