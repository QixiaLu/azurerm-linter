package a

// Mock parse package - declare as variable to simulate package
var parse parsePackage

type parsePackage struct{}

type ResourceID struct {
	SubscriptionId string
	ResourceGroup  string
	Name           string
}

func (parsePackage) NewResourceID(subscriptionId, resourceGroup, name string) ResourceID {
	return ResourceID{
		SubscriptionId: subscriptionId,
		ResourceGroup:  resourceGroup,
		Name:           name,
	}
}

func (parsePackage) ParseResourceID(id string) ResourceID {
	return ResourceID{
		SubscriptionId: "parsed-sub",
		ResourceGroup:  "parsed-rg",
		Name:           "parsed-name",
	}
}

func (id ResourceID) ID() string {
	return "/subscriptions/" + id.SubscriptionId + "/resourceGroups/" + id.ResourceGroup + "/providers/Test/resources/" + id.Name
}
