package a

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Test: Resource with Parse*ID from API response - should be skipped
func resourceParseFromAPI() *schema.Resource {
	return &schema.Resource{
		Create: apiParserResourceCreate,

		Schema: map[string]*schema.Schema{
			// ID from API response, not extractable
			"location": {
				Type:     schema.TypeString,
				Required: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"parent_id": {
				Type:     schema.TypeString,
				Required: true,
			},

			"tags": {
				Type:     schema.TypeMap,
				Optional: true,
			},
		},
	}
}

func apiParserResourceCreate(d *schema.ResourceData, meta interface{}) error {
	name := d.Get("name").(string)
	parentId := d.Get("parent_id").(string)

	// Simulate API call that returns an ID
	resp := apiClient.parseParentResourceId(parentId, name)

	// Parse ID from API response - not from New*ID constructor
	id := parse.ParseResourceID(resp.ID)
	d.SetId(id.ID())
	return nil
}

// Mock API client
var apiClient mockAPIClient

type mockAPIClient struct{}

type APIResponse struct {
	ID string
}

func (mockAPIClient) parseParentResourceId(parentId, name string) APIResponse {
	return APIResponse{ID: "/some/api/generated/id"}
}
