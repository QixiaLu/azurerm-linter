package azbp014

import (
	"github.com/hashicorp/go-azure-sdk/resource-manager/compute/2024-03-01/virtualmachines"
)

func invalidCase1() {
	_ = virtualmachines.GetOperationOptions{} // want `AZBP014`
}

func invalidCase2() {
	opts := virtualmachines.GetOperationOptions{} // want `AZBP014`
	s := "all"
	opts.Expand = &s
	_ = opts
}

// No Default* exists
func validCase1() {
	_ = virtualmachines.ListOperationOptions{}
}

// Non-empty literal
func validCase2() {
	s := "all"
	_ = virtualmachines.GetOperationOptions{Expand: &s}
}
