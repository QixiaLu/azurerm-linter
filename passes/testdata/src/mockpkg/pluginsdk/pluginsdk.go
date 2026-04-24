package pluginsdk

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// This file is to mock pluginsdk in azurerm

const (
	TypeString = schema.TypeString
	TypeBool   = schema.TypeBool
	TypeInt    = schema.TypeInt
	TypeMap    = schema.TypeMap
	TypeList   = schema.TypeList
)

type (
	Resource     = schema.Resource
	Schema       = schema.Schema
	ResourceData = schema.ResourceData
)

// GetWriteOnly retrieves a write-only attribute value from the ResourceData.
func GetWriteOnly(d *ResourceData, key string, valType interface{}) (interface{}, error) {
	return d.Get(key), nil
}
