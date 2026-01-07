package shared

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ResultTypesTestSuite struct {
	suite.Suite
}

func TestResultTypesTestSuite(t *testing.T) {
	suite.Run(t, new(ResultTypesTestSuite))
}

// BuildMapKey tests

func (s *ResultTypesTestSuite) Test_BuildMapKey_empty_prefix_returns_name() {
	result := BuildMapKey("", "resourceName")
	s.Equal("resourceName", result)
}

func (s *ResultTypesTestSuite) Test_BuildMapKey_with_prefix_joins_with_slash() {
	result := BuildMapKey("parent", "resourceName")
	s.Equal("parent/resourceName", result)
}

func (s *ResultTypesTestSuite) Test_BuildMapKey_nested_prefix() {
	result := BuildMapKey("parent/child", "resourceName")
	s.Equal("parent/child/resourceName", result)
}

// BuildElementPath tests

func (s *ResultTypesTestSuite) Test_BuildElementPath_empty_parent_returns_segment() {
	result := BuildElementPath("", "resources", "myResource")
	s.Equal("resources.myResource", result)
}

func (s *ResultTypesTestSuite) Test_BuildElementPath_with_parent_joins_with_colons() {
	result := BuildElementPath("children.parent", "resources", "myResource")
	s.Equal("children.parent::resources.myResource", result)
}

func (s *ResultTypesTestSuite) Test_BuildElementPath_nested_path() {
	result := BuildElementPath("children.a::children.b", "resources", "myResource")
	s.Equal("children.a::children.b::resources.myResource", result)
}

// LookupByKey tests

type testItem struct {
	Name string
}

func (s *ResultTypesTestSuite) Test_LookupByKey_finds_by_path_key() {
	m := map[string]*testItem{
		"parent/item1": {Name: "item1"},
	}
	result := LookupByKey(m, "parent/item1", "item1")
	s.NotNil(result)
	s.Equal("item1", result.Name)
}

func (s *ResultTypesTestSuite) Test_LookupByKey_falls_back_to_name() {
	m := map[string]*testItem{
		"item1": {Name: "item1"},
	}
	result := LookupByKey(m, "parent/item1", "item1")
	s.NotNil(result)
	s.Equal("item1", result.Name)
}

func (s *ResultTypesTestSuite) Test_LookupByKey_returns_nil_when_not_found() {
	m := map[string]*testItem{}
	result := LookupByKey(m, "parent/item1", "item1")
	s.Nil(result)
}

func (s *ResultTypesTestSuite) Test_LookupByKey_prefers_path_key_over_name() {
	m := map[string]*testItem{
		"parent/item1": {Name: "path-item"},
		"item1":        {Name: "name-item"},
	}
	result := LookupByKey(m, "parent/item1", "item1")
	s.NotNil(result)
	s.Equal("path-item", result.Name)
}
