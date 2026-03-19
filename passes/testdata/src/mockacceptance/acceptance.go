package acceptance

// Mock types matching the acceptance package API for testing

type TestData struct {
	ResourceName string
}

type TestStep struct{}

func (d TestData) ImportStep(ignoreFields ...string) TestStep {
	return TestStep{}
}

func BuildTestData(t interface{}, resourceType string, resourceLabel string) TestData {
	return TestData{}
}
