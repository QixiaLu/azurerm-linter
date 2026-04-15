package passes_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qixialu/azurerm-linter/loader"
	"github.com/qixialu/azurerm-linter/passes"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAZNR005(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, passes.AZNR005Analyzer, "testdata/src/aznr005")
}

func TestAZNR005DeletionOnlyDiffStillReportsInFilteredMode(t *testing.T) {
	testdata := analysistest.TestData()
	diffPath := filepath.Join(t.TempDir(), "deletion_only.diff")
	diff := `diff --git a/internal/services/cdn/registration.go b/internal/services/cdn/registration.go
index 1111111..2222222 100644
--- a/internal/services/cdn/registration.go
+++ b/internal/services/cdn/registration.go
@@ -10,4 +10,3 @@ func (r Registration) SupportedResources() map[string]*pluginsdk.Resource {
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
		_, _ = loader.LoadChanges(loader.LoaderOptions{NoFilter: true})
	})

	analysistest.Run(t, testdata, passes.AZNR005Analyzer, "testdata/src/internal/services/cdn")
}

func TestAZNR005DeletionOnlyHunkStillReportsWhenOtherLinesChanged(t *testing.T) {
	testdata := analysistest.TestData()
	diffPath := filepath.Join(t.TempDir(), "deletion_only_with_other_changes.diff")
	diff := `diff --git a/internal/services/cdn/registration.go b/internal/services/cdn/registration.go
index 1111111..2222222 100644
--- a/internal/services/cdn/registration.go
+++ b/internal/services/cdn/registration.go
@@ -10,4 +10,3 @@ func (r Registration) SupportedResources() map[string]*pluginsdk.Resource {
-//lintignore:AZNR005 temporary exemption
 	return map[string]*pluginsdk.Resource{
 		"azurerm_managed_disk":     nil,
 		"azurerm_availability_set": nil,
@@ -15,3 +14,3 @@ func (r Registration) Resources() []sdk.Resource {
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
		_, _ = loader.LoadChanges(loader.LoaderOptions{NoFilter: true})
	})

	analysistest.Run(t, testdata, passes.AZNR005Analyzer, "testdata/src/internal/services/cdn")
}
