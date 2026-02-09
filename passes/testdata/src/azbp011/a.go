package azbp011

import (
	"strings"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
)

type HibernateSupport string

const HibernateSupportDisabled HibernateSupport = "Disabled"
const HibernateSupportEnabled HibernateSupport = "Enabled"

func badStringCastingComparison() bool {
	var hibernateSupport *HibernateSupport

	result1 := strings.EqualFold(string(pointer.From(hibernateSupport)), string(HibernateSupportDisabled)) // want `AZBP011`
	result2 := strings.EqualFold(string(HibernateSupportEnabled), string(HibernateSupportDisabled))        // want `AZBP011`

	return result1 || result2
}

func goodDirectEnumComparison() bool {
	var hibernateSupport *HibernateSupport

	result1 := pointer.From(hibernateSupport) == HibernateSupportEnabled
	return result1
}

func legitimateStringEqualFold() bool {
	userInput := "enabled"

	result1 := strings.EqualFold(userInput, string(HibernateSupportEnabled))
	return result1
}
