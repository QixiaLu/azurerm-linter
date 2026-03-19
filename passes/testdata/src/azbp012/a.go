package azbp012

import (
	"github.com/hashicorp/go-azure-helpers/lang/pointer"
)

type VirtualNetworkType string

const (
	VirtualNetworkTypeManaged   VirtualNetworkType = "Managed"
	VirtualNetworkTypeUnmanaged VirtualNetworkType = "Unmanaged"
)

type Properties struct {
	VirtualNetworkType *VirtualNetworkType
	Enabled            *bool
}

type Parameters struct {
	Properties *Properties
}

type Model struct {
	ManagedVirtualNetworkRegions []string
	IsEnabled                    bool
}

func invalidCase1(model Model) {
	var parameters Parameters
	parameters.Properties = &Properties{}
	if len(model.ManagedVirtualNetworkRegions) != 0 { // want `AZBP012`
		parameters.Properties.VirtualNetworkType = pointer.To(VirtualNetworkTypeManaged)
	} else {
		parameters.Properties.VirtualNetworkType = pointer.To(VirtualNetworkTypeUnmanaged)
	}
	use(parameters)
}

func invalidCase2(model Model) {
	var mode string
	if model.IsEnabled { // want `AZBP012`
		mode = "active"
	} else {
		mode = "inactive"
	}
	useString(mode)
}

func validCase1(model Model) {
	var parameters Parameters
	parameters.Properties = &Properties{}
	parameters.Properties.VirtualNetworkType = pointer.To(VirtualNetworkTypeUnmanaged)
	if len(model.ManagedVirtualNetworkRegions) != 0 {
		parameters.Properties.VirtualNetworkType = pointer.To(VirtualNetworkTypeManaged)
	}
	use(parameters)
}

func validCase2(model Model) {
	mode := "default"
	if model.IsEnabled {
		mode = "active"
	}
	useString(mode)
}

func validCase3(model Model) {
	var a, b string
	if model.IsEnabled {
		a = "yes"
	} else {
		b = "no"
	}
	useString(a)
	useString(b)
}

func validCase4(model Model) {
	var mode string
	if model.IsEnabled {
		mode = "active"
		doSomething()
	} else {
		mode = "inactive"
	}
	useString(mode)
}

func validCase5(model Model) {
	var tier string
	if model.IsEnabled {
		tier = "premium"
	} else if !model.IsEnabled {
		tier = "standard"
	} else {
		tier = "basic"
	}
	useString(tier)
}

func validCase6(model Model) {
	var mode string
	if model.IsEnabled {
		mode = "active"
		doSomething()
	} else {
		mode = "inactive"
		doSomethingElse()
	}
	useString(mode)
}

func use(p Parameters)   {}
func useString(s string) {}
func doSomething()       {}
func doSomethingElse()   {}
