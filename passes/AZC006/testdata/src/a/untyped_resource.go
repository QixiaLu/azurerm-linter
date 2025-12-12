package a

import (
    "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceInValid() *schema.Resource {
    return &schema.Resource{
        Create: resourceValidCreate,

        Schema: map[string]*schema.Schema{ // want `name, resource_group_name, location, account_replication_type, account_tier, enable_https, tags, primary_key`
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

            "location": {
                Type:     schema.TypeString,
                Required: true,
                ForceNew: true,
            },

            "account_replication_type": {
                Type:     schema.TypeString,
                Required: true,
            },

            "account_tier": {
                Type:     schema.TypeString,
                Required: true,
            },

            "enable_https": {
                Type:     schema.TypeBool,
                Optional: true,
            },

            "tags": {
                Type:     schema.TypeMap,
                Optional: true,
            },

            "primary_key": {
                Type:     schema.TypeString,
                Computed: true,
            },
        },
    }
}

func resourceValidCreate(d *schema.ResourceData, meta interface{}) error {
    resourceGroupName := d.Get("resource_group_name").(string)
    name := d.Get("name").(string)
    subscriptionId := "randomId"

    id := parse.NewResourceID(subscriptionId, resourceGroupName, name)
    d.SetId(id.ID())
    return nil
}

// Mock parse package - declare as variable to simulate package
var parse parsePackage

type parsePackage struct{}

type ResourceID struct {
    SubscriptionId string
    ResourceGroup string
    Name          string
}

func (parsePackage) NewResourceID(subscriptionId, resourceGroup, name string) ResourceID {
    return ResourceID{
        SubscriptionId: "11",
        ResourceGroup: resourceGroup,
        Name:          name,
    }
}

func (id ResourceID) ID() string {
    return "/subscriptions/sub/resourceGroups/" + id.ResourceGroup + "/providers/Test/resources/" + id.Name
}