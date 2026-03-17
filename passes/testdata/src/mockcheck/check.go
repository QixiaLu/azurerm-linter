package check

// Mock types matching the acceptance check package API:
//   check.That(resourceName).Key(key).HasValue(value)

type ThatType struct{}
type ThatWithKeyType struct{}
type TestCheckFunc func()

func That(resourceName string) ThatType {
	return ThatType{}
}

func (t ThatType) Key(key string) ThatWithKeyType {
	return ThatWithKeyType{}
}

func (t ThatType) ExistsInAzure(r interface{}) TestCheckFunc {
	return nil
}

func (t ThatWithKeyType) HasValue(value string) TestCheckFunc {
	return nil
}

func (t ThatWithKeyType) Exists() TestCheckFunc {
	return nil
}

func (t ThatWithKeyType) IsNotEmpty() TestCheckFunc {
	return nil
}

func (t ThatWithKeyType) IsEmpty() TestCheckFunc {
	return nil
}

func (t ThatWithKeyType) IsSet() TestCheckFunc {
	return nil
}

func (t ThatWithKeyType) DoesNotExist() TestCheckFunc {
	return nil
}

func (t ThatWithKeyType) MatchesRegex(pattern string) TestCheckFunc {
	return nil
}
