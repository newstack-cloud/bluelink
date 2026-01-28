package docmodel

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type DuplicateKeyDetectionSuite struct {
	suite.Suite
}

func (s *DuplicateKeyDetectionSuite) TestDetectDuplicateKeys_YAML_TopLevel() {
	content := `version: 2021-12-18
resources:
  myTable:
    type: aws/dynamodb/table
  myTable:
    type: aws/lambda/function
`
	node, err := ParseYAMLToUnified(content)
	s.Require().NoError(err)

	result := DetectDuplicateKeys(node)
	s.Require().NotNil(result)

	myTableErrors := filterByKey(result.Errors, "myTable")
	s.Require().Len(myTableErrors, 2, "Should detect 2 occurrences of duplicate key 'myTable'")

	s.Assert().True(myTableErrors[0].IsFirst)
	s.Assert().False(myTableErrors[1].IsFirst)
}

func (s *DuplicateKeyDetectionSuite) TestDetectDuplicateKeys_YAML_Nested() {
	content := `resources:
  myResource:
    spec:
      tableName: test
      tableName: test2
`
	node, err := ParseYAMLToUnified(content)
	s.Require().NoError(err)

	result := DetectDuplicateKeys(node)
	s.Require().NotNil(result)

	tableNameErrors := filterByKey(result.Errors, "tableName")
	s.Require().Len(tableNameErrors, 2)
	s.Assert().Equal("tableName", tableNameErrors[0].Key)
}

func (s *DuplicateKeyDetectionSuite) TestDetectDuplicateKeys_YAML_FlowMapping() {
	content := `metadata:
  labels: {app: test, app: test2}
`
	node, err := ParseYAMLToUnified(content)
	s.Require().NoError(err)

	result := DetectDuplicateKeys(node)
	s.Require().NotNil(result)

	appErrors := filterByKey(result.Errors, "app")
	s.Require().Len(appErrors, 2)
	s.Assert().Equal("app", appErrors[0].Key)
}

func (s *DuplicateKeyDetectionSuite) TestDetectDuplicateKeys_JSON_Simple() {
	content := `{
  "version": "2021-12-18",
  "resources": {
    "myTable": {"type": "aws/dynamodb/table"},
    "myTable": {"type": "aws/lambda/function"}
  }
}`
	node, err := ParseJSONCToUnified(content)
	s.Require().NoError(err)

	result := DetectDuplicateKeys(node)
	s.Require().NotNil(result)

	myTableErrors := filterByKey(result.Errors, "myTable")
	s.Require().Len(myTableErrors, 2)
	s.Assert().Equal("myTable", myTableErrors[0].Key)
}

func (s *DuplicateKeyDetectionSuite) TestDetectDuplicateKeys_JSON_Nested() {
	content := `{
  "resources": {
    "myResource": {
      "spec": {
        "tableName": "test",
        "tableName": "test2"
      }
    }
  }
}`
	node, err := ParseJSONCToUnified(content)
	s.Require().NoError(err)

	result := DetectDuplicateKeys(node)
	s.Require().NotNil(result)

	tableNameErrors := filterByKey(result.Errors, "tableName")
	s.Require().Len(tableNameErrors, 2)
}

func (s *DuplicateKeyDetectionSuite) TestDetectDuplicateKeys_NoDuplicates() {
	content := `version: 2021-12-18
resources:
  myTable:
    type: aws/dynamodb/table
  myFunction:
    type: aws/lambda/function
`
	node, err := ParseYAMLToUnified(content)
	s.Require().NoError(err)

	result := DetectDuplicateKeys(node)
	s.Require().NotNil(result)
	s.Assert().Empty(result.Errors)
}

func (s *DuplicateKeyDetectionSuite) TestDetectDuplicateKeys_MultipleGroups() {
	content := `resources:
  a:
    type: t1
  a:
    type: t2
  b:
    type: t3
  b:
    type: t4
`
	node, err := ParseYAMLToUnified(content)
	s.Require().NoError(err)

	result := DetectDuplicateKeys(node)
	s.Require().NotNil(result)

	aErrors := filterByKey(result.Errors, "a")
	bErrors := filterByKey(result.Errors, "b")
	s.Assert().Len(aErrors, 2, "Should have 2 errors for 'a'")
	s.Assert().Len(bErrors, 2, "Should have 2 errors for 'b'")
}

func (s *DuplicateKeyDetectionSuite) TestDetectDuplicateKeys_TripleDuplicate() {
	content := `resources:
  myTable:
    type: t1
  myTable:
    type: t2
  myTable:
    type: t3
`
	node, err := ParseYAMLToUnified(content)
	s.Require().NoError(err)

	result := DetectDuplicateKeys(node)
	s.Require().NotNil(result)

	myTableErrors := filterByKey(result.Errors, "myTable")
	s.Require().Len(myTableErrors, 3, "Should have 3 errors for triple duplicate")
	s.Assert().True(myTableErrors[0].IsFirst)
	s.Assert().False(myTableErrors[1].IsFirst)
	s.Assert().False(myTableErrors[2].IsFirst)
}

func (s *DuplicateKeyDetectionSuite) TestDetectDuplicateKeys_NilRoot() {
	result := DetectDuplicateKeys(nil)
	s.Require().NotNil(result)
	s.Assert().Empty(result.Errors)
}

func (s *DuplicateKeyDetectionSuite) TestDetectDuplicateKeys_HasKeyRange() {
	content := `resources:
  myTable:
    type: test
  myTable:
    type: test2
`
	node, err := ParseYAMLToUnified(content)
	s.Require().NoError(err)

	result := DetectDuplicateKeys(node)
	s.Require().NotNil(result)

	myTableErrors := filterByKey(result.Errors, "myTable")
	s.Require().Len(myTableErrors, 2)

	for _, err := range myTableErrors {
		s.Assert().NotNil(err.KeyRange, "KeyRange should be set for duplicate key errors")
	}
}

func filterByKey(errors []*DuplicateKeyError, key string) []*DuplicateKeyError {
	var filtered []*DuplicateKeyError
	for _, err := range errors {
		if err.Key == key {
			filtered = append(filtered, err)
		}
	}
	return filtered
}

func TestDuplicateKeyDetectionSuite(t *testing.T) {
	suite.Run(t, new(DuplicateKeyDetectionSuite))
}
