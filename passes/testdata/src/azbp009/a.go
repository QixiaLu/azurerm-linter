package azbp009

import (
	"context"
	. "errors"
	"fmt"
	_ "strings"

	helper "github.com/hashicorp/go-azure-helpers/lang/pointer"
	ptr "github.com/hashicorp/go-azure-helpers/lang/pointer"
)

func badFunction() {
	context := "invalid" // want `AZBP009`
	ptr := 42            // want `AZBP009`
	helper := "bad"      // want `AZBP009`
	fmt := "bad"         // want `AZBP009`
	pointer := "dsds"
	_ = context
	_ = ptr
	_ = helper
	_ = fmt
	_ = pointer
}

func testFunction() {
	contest, ptr := "invalid", 42 // want `AZBP009`
	_ = contest
	_ = ptr
}

func goodFunction() {
	// Use imports to avoid unused import errors
	ctx := context.Background()
	fmt.Println("hello")
	err := New("test error")
	val := ptr.To("test")
	val2 := helper.From(&val)
	_ = ctx
	_ = err
	_ = val2
}
