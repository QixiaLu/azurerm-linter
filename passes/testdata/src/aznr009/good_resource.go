package aznr009

import (
	"context"

	"testdata/src/mockpkg/pluginsdk"
	"testdata/src/mockpkg/sdk"
)

// Test Case 3: ID is parsed before assignment - should NOT report error

type GoodResourceModel struct {
	Name     string `tfschema:"name"`
	SubnetId string `tfschema:"subnet_id"`
}

type GoodResource struct{}

var _ sdk.Resource = GoodResource{}

func (r GoodResource) ResourceType() string {
	return "azurerm_good_resource"
}

func (r GoodResource) ModelObject() interface{} {
	return &GoodResourceModel{}
}

func (r GoodResource) Arguments() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{
		"name": {
			Type:     pluginsdk.TypeString,
			Required: true,
			ForceNew: true,
		},
		"subnet_id": {
			Type:     pluginsdk.TypeString,
			Required: true,
			ForceNew: true,
		},
	}
}

func (r GoodResource) Attributes() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{}
}

func (r GoodResource) Create() sdk.ResourceFunc {
	return sdk.ResourceFunc{}
}

func (r GoodResource) Read() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			resp := getResponse()

			// ID is parsed first via method call, then assigned - this is fine
			parsedSubnetId, _ := parseSomethingInsensitively(resp.Properties.Subnet.Id)
			state := GoodResourceModel{
				Name:     "test",
				SubnetId: parsedSubnetId.ID(),
			}

			_ = state
			return nil
		},
	}
}

func (r GoodResource) Delete() sdk.ResourceFunc {
	return sdk.ResourceFunc{}
}

// Test Case 4: Model-to-model copy - should NOT report error

type ModelCopyResourceModel struct {
	Name     string `tfschema:"name"`
	SubnetId string `tfschema:"subnet_id"`
}

type ModelCopyResource struct{}

var _ sdk.Resource = ModelCopyResource{}

func (r ModelCopyResource) ResourceType() string {
	return "azurerm_model_copy_resource"
}

func (r ModelCopyResource) ModelObject() interface{} {
	return &ModelCopyResourceModel{}
}

func (r ModelCopyResource) Arguments() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{
		"name": {
			Type:     pluginsdk.TypeString,
			Required: true,
			ForceNew: true,
		},
		"subnet_id": {
			Type:     pluginsdk.TypeString,
			Required: true,
			ForceNew: true,
		},
	}
}

func (r ModelCopyResource) Attributes() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{}
}

func (r ModelCopyResource) Create() sdk.ResourceFunc {
	return sdk.ResourceFunc{}
}

func (r ModelCopyResource) Read() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			var config ModelCopyResourceModel
			_ = metadata.Decode(&config)

			var state ModelCopyResourceModel
			// Model-to-model copy - should NOT be flagged
			state.SubnetId = config.SubnetId

			_ = state
			return nil
		},
	}
}

func (r ModelCopyResource) Delete() sdk.ResourceFunc {
	return sdk.ResourceFunc{}
}
