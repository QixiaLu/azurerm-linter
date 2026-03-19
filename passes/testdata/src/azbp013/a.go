package azbp013

import "fmt"

type Model struct {
	Id         *string
	Properties *Properties
}

type Properties struct {
	Name     *string
	SubProps *SubProperties
}

type SubProperties struct {
	Value *string
}

type Response struct {
	Model *Model
}

type ResourceId struct {
	Name string
}

// Two chained nil checks returning error
func invalidCase1(resp Response, id ResourceId) error {
	if resp.Model == nil || resp.Model.Properties == nil { // want `AZBP013`
		return fmt.Errorf("retrieving %s: model was nil", id.Name)
	}
	return nil
}

// Three chained nil checks
func invalidCase2(resp Response, id ResourceId) error {
	if resp.Model == nil || resp.Model.Properties == nil || resp.Model.Properties.SubProps == nil { // want `AZBP013`
		return fmt.Errorf("cannot read %s", id.Name)
	}
	return nil
}

// Error as second return value
func invalidCase3(resp Response, id ResourceId) (*Model, error) {
	if resp.Model == nil || resp.Model.Id == nil { // want `AZBP013`
		return nil, fmt.Errorf("retrieving %s: model/id was nil", id.Name)
	}
	return resp.Model, nil
}

// Chained but body returns non-error (flatten pattern)
func validCase1(model *Model) []interface{} {
	if model == nil || model.Properties == nil {
		return []interface{}{}
	}
	return []interface{}{model.Properties}
}

// Non-chained sibling checks
func validCase2(model *Model) error {
	if model.Properties == nil || model.Id == nil {
		return fmt.Errorf("properties or id was nil")
	}
	return nil
}

// && instead of ||
func validCase3(resp Response, id ResourceId) error {
	if resp.Model != nil && resp.Model.Properties == nil {
		return fmt.Errorf("retrieving %s: properties was nil", id.Name)
	}
	return nil
}

// Single nil check
func validCase4(resp Response, id ResourceId) error {
	if resp.Model == nil {
		return fmt.Errorf("retrieving %s: model was nil", id.Name)
	}
	return nil
}
