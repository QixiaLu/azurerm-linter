package loader

import (
	"path/filepath"
	"testing"
)

func TestChangeSetShouldReportFallsBackToChangedFileWhenNoLinesTracked(t *testing.T) {
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

	if !cs.ShouldReport(file, 37) {
		t.Fatalf("ShouldReport(%q, 37) = false, want true", file)
	}
	if !cs.ShouldReport(file, 1) {
		t.Fatalf("ShouldReport(%q, 1) = false, want true for changed file fallback", file)
	}
	if cs.ShouldReport(filepath.Join("repo", "internal", "services", "dns", "registration.go"), 37) {
		t.Fatalf("ShouldReport() = true for unchanged file, want false")
	}
}

func TestChangeSetShouldReportRemainsLineScopedWhenLinesTracked(t *testing.T) {
	cs := NewChangeSet()
	file := "internal/services/cdn/registration.go"
	cs.changedFiles[file] = true
	cs.changedLines[file] = map[int]bool{42: true}

	fullPath := filepath.Join("repo", "internal", "services", "cdn", "registration.go")

	if !cs.ShouldReport(fullPath, 42) {
		t.Fatalf("ShouldReport(%q, 42) = false, want true", fullPath)
	}
	if cs.ShouldReport(fullPath, 41) {
		t.Fatalf("ShouldReport(%q, 41) = true, want false", fullPath)
	}
}

func TestChangeSetShouldReportFallsBackWhenFileHasDeletionOnlyHunk(t *testing.T) {
	cs := NewChangeSet()

	diff := `diff --git a/internal/services/cdn/registration.go b/internal/services/cdn/registration.go
index 1111111..2222222 100644
--- a/internal/services/cdn/registration.go
+++ b/internal/services/cdn/registration.go
@@ -37,1 +37,0 @@
-//lintignore:AZNR005 temporary exemption
@@ -55,1 +55,1 @@
-	return resources
+	return registration.Resources()
`

	if err := cs.parseDiffOutput(diff); err != nil {
		t.Fatalf("parseDiffOutput() error = %v", err)
	}

	file := filepath.Join("repo", "internal", "services", "cdn", "registration.go")

	if !cs.ShouldReport(file, 37) {
		t.Fatalf("ShouldReport(%q, 37) = false, want true for deletion-only hunk fallback", file)
	}
	if !cs.ShouldReport(file, 55) {
		t.Fatalf("ShouldReport(%q, 55) = false, want true for tracked changed line", file)
	}
	if !cs.fileFallback["internal/services/cdn/registration.go"] {
		t.Fatalf("fileFallback not recorded for deletion-only hunk")
	}
}
