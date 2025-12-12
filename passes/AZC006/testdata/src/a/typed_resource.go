package a

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Test: Typed resource using model fields instead of d.Get()
func typedResourceCorrectOrder() *schema.Resource {
	return &schema.Resource{
		Create: typedResourceCreate,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": {
				Type:     schema.TypeString,
				Required: true,
			},

			"sku_name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"tags": {
				Type:     schema.TypeMap,
				Optional: true,
			},
		},
	}
}

// Test: Typed resource with wrong order
func typedResourceWrongOrder() *schema.Resource {
	return &schema.Resource{
		Create: typedResourceCreate,

		Schema: map[string]*schema.Schema{ // want `name, resource_group_name, location, sku_name, enabled, tags`
			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"tags": {
				Type:     schema.TypeMap,
				Optional: true,
			},

			"location": {
				Type:     schema.TypeString,
				Required: true,
			},

			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"sku_name": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

type TypedResourceModel struct {
	Name              string            `tfschema:"name"`
	ResourceGroupName string            `tfschema:"resource_group_name"`
	Location          string            `tfschema:"location"`
	SkuName           string            `tfschema:"sku_name"`
	Enabled           bool              `tfschema:"enabled"`
	Tags              map[string]string `tfschema:"tags"`
}

type mockMetadata struct {
	model TypedResourceModel
}

func (m *mockMetadata) Decode(target interface{}) error {
	// Simulate decoding
	return nil
}

func typedResourceCreate(d *schema.ResourceData, meta interface{}) error {
	subscriptionId := "sub-typed"

	// Simulate typed SDK pattern
	metadata := &mockMetadata{}
	var model TypedResourceModel
	metadata.Decode(&model)

	// Access via model fields instead of d.Get()
	id := parse.NewResourceID(subscriptionId, model.ResourceGroupName, model.Name)
	d.SetId(id.ID())
	return nil
}
