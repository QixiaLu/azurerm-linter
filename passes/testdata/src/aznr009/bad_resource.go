package aznr009

import (
	"context"
	"fmt"

	"testdata/src/mockpkg/pluginsdk"
	"testdata/src/mockpkg/sdk"
)

// Test Case 1: Direct assignment of API response ID in Read - should report error

type BadResourceModel struct {
	Name                           string `tfschema:"name"`
	SubnetId                       string `tfschema:"subnet_id"`
	WebApplicationFirewallPolicyId string `tfschema:"web_application_firewall_policy_id"`
}

type BadResource struct{}

var _ sdk.Resource = BadResource{}

func (r BadResource) ResourceType() string {
	return "azurerm_bad_resource"
}

func (r BadResource) ModelObject() interface{} {
	return &BadResourceModel{}
}

func (r BadResource) Arguments() map[string]*pluginsdk.Schema {
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
		"web_application_firewall_policy_id": {
			Type:     pluginsdk.TypeString,
			Required: true,
			ForceNew: true,
		},
	}
}

func (r BadResource) Attributes() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{}
}

func (r BadResource) Create() sdk.ResourceFunc {
	return sdk.ResourceFunc{}
}

func (r BadResource) Read() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			var state BadResourceModel

			resp := getResponse()
			ptrResp := getPtrResponse()

			// Direct assignment of API response ID - should flag
			state.SubnetId = resp.Properties.Subnet.Id               // want `AZNR009`
			state.WebApplicationFirewallPolicyId = resp.WafPolicy.Id // want `AZNR009`

			// Pointer dereference of API response ID - should also flag
			state.SubnetId = *ptrResp.Properties.Subnet.Id // want `AZNR009`

			_ = state
			return nil
		},
	}
}

func (r BadResource) Delete() sdk.ResourceFunc {
	return sdk.ResourceFunc{}
}

// Test Case 2: Direct assignment in composite literal - should report error

type BadCompositeLitResourceModel struct {
	Name     string `tfschema:"name"`
	SubnetId string `tfschema:"subnet_id"`
}

type BadCompositeLitResource struct{}

var _ sdk.Resource = BadCompositeLitResource{}

func (r BadCompositeLitResource) ResourceType() string {
	return "azurerm_bad_composite_lit_resource"
}

func (r BadCompositeLitResource) ModelObject() interface{} {
	return &BadCompositeLitResourceModel{}
}

func (r BadCompositeLitResource) Arguments() map[string]*pluginsdk.Schema {
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

func (r BadCompositeLitResource) Attributes() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{}
}

func (r BadCompositeLitResource) Create() sdk.ResourceFunc {
	return sdk.ResourceFunc{}
}

func (r BadCompositeLitResource) Read() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			resp := getResponse()

			state := BadCompositeLitResourceModel{
				Name:     "test",
				SubnetId: resp.Properties.Subnet.Id, // want `AZNR009`
			}

			_ = state
			return nil
		},
	}
}

func (r BadCompositeLitResource) Delete() sdk.ResourceFunc {
	return sdk.ResourceFunc{}
}

// Mock API response types

type ApiResponse struct {
	Properties *ApiProperties
	WafPolicy  *WafPolicyRef
}

type ApiProperties struct {
	Subnet *SubnetRef
}

type SubnetRef struct {
	Id string
}

type WafPolicyRef struct {
	Id string
}

func getResponse() *ApiResponse {
	return &ApiResponse{
		Properties: &ApiProperties{
			Subnet: &SubnetRef{Id: "/subscriptions/xxx/resourceGroups/xxx/providers/Microsoft.Network/virtualNetworks/xxx/subnets/xxx"},
		},
		WafPolicy: &WafPolicyRef{Id: "/subscriptions/xxx/resourceGroups/xxx/providers/Microsoft.Network/ApplicationGatewayWebApplicationFirewallPolicies/xxx"},
	}
}

type PtrApiResponse struct {
	Properties *PtrApiProperties
}

type PtrApiProperties struct {
	Subnet *PtrSubnetRef
}

type PtrSubnetRef struct {
	Id *string
}

func getPtrResponse() *PtrApiResponse {
	id := "/subscriptions/xxx/resourceGroups/xxx/providers/Microsoft.Network/virtualNetworks/xxx/subnets/xxx"
	return &PtrApiResponse{
		Properties: &PtrApiProperties{
			Subnet: &PtrSubnetRef{Id: &id},
		},
	}
}

type ParsedId struct{}

func (p *ParsedId) ID() string { return "" }

func parseSomethingInsensitively(id string) (*ParsedId, error) {
	return &ParsedId{}, nil
}

// Suppress unused import warning
var _ = fmt.Sprintf
