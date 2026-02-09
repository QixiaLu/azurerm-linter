package azbp009

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-azure-helpers/lang/pointer"
)

func badFunction() {
	context := "invalid" // want `AZBP009`
	pointer := 42        // want `AZBP009`
	fmt := "bad"         // want `AZBP009`
	_ = context
	_ = pointer
	_ = fmt
}

func testFunction() {
	contest, pointer := "invalid", 42 // want `AZBP009`
	_ = contest
	_ = pointer
}

func goodFunction() {
	ctx := context.Background()
	_ = ctx
}

// Using the imported packages correctly
func usingImports() {
	value := pointer.To("test")
	_ = value
}

func properFmtUsage(data fmt.Stringer) {
	_ = data
}

func badFmtUsage() {
	fmt := "invalid string" // want `AZBP009`
	_ = fmt
}
