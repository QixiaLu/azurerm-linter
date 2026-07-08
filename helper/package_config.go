package helper

import "golang.org/x/tools/go/packages"

func NewPackageLoadConfig(dir string, tests bool) *packages.Config {
	return &packages.Config{
		Mode:       packages.LoadAllSyntax,
		Tests:      tests,
		Dir:        dir,
		BuildFlags: []string{"-buildvcs=false"},
	}
}
