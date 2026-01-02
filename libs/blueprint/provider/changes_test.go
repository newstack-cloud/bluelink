package provider

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ChangesTestSuite struct {
	suite.Suite
}

func (s *ChangesTestSuite) Test_ChangesHasFieldChanges_returns_false_for_nil_changes() {
	result := ChangesHasFieldChanges(nil)
	s.False(result)
}

func (s *ChangesTestSuite) Test_ChangesHasFieldChanges_returns_false_for_empty_changes() {
	changes := &Changes{}
	result := ChangesHasFieldChanges(changes)
	s.False(result)
}

func (s *ChangesTestSuite) Test_ChangesHasFieldChanges_returns_false_for_only_unchanged_fields() {
	changes := &Changes{
		UnchangedFields: []string{"spec.field1", "spec.field2"},
	}
	result := ChangesHasFieldChanges(changes)
	s.False(result)
}

func (s *ChangesTestSuite) Test_ChangesHasFieldChanges_returns_true_for_modified_fields() {
	changes := &Changes{
		ModifiedFields: []FieldChange{
			{FieldPath: "spec.field1"},
		},
	}
	result := ChangesHasFieldChanges(changes)
	s.True(result)
}

func (s *ChangesTestSuite) Test_ChangesHasFieldChanges_returns_true_for_new_fields() {
	changes := &Changes{
		NewFields: []FieldChange{
			{FieldPath: "spec.field1"},
		},
	}
	result := ChangesHasFieldChanges(changes)
	s.True(result)
}

func (s *ChangesTestSuite) Test_ChangesHasFieldChanges_returns_true_for_removed_fields() {
	changes := &Changes{
		RemovedFields: []string{"spec.field1"},
	}
	result := ChangesHasFieldChanges(changes)
	s.True(result)
}

func (s *ChangesTestSuite) Test_LinkChangesHasFieldChanges_returns_false_for_nil_changes() {
	result := LinkChangesHasFieldChanges(nil)
	s.False(result)
}

func (s *ChangesTestSuite) Test_LinkChangesHasFieldChanges_returns_false_for_empty_changes() {
	changes := &LinkChanges{}
	result := LinkChangesHasFieldChanges(changes)
	s.False(result)
}

func (s *ChangesTestSuite) Test_LinkChangesHasFieldChanges_returns_false_for_only_unchanged_fields() {
	changes := &LinkChanges{
		UnchangedFields: []string{"link.field1", "link.field2"},
	}
	result := LinkChangesHasFieldChanges(changes)
	s.False(result)
}

func (s *ChangesTestSuite) Test_LinkChangesHasFieldChanges_returns_true_for_modified_fields() {
	changes := &LinkChanges{
		ModifiedFields: []*FieldChange{
			{FieldPath: "link.field1"},
		},
	}
	result := LinkChangesHasFieldChanges(changes)
	s.True(result)
}

func (s *ChangesTestSuite) Test_LinkChangesHasFieldChanges_returns_true_for_new_fields() {
	changes := &LinkChanges{
		NewFields: []*FieldChange{
			{FieldPath: "link.field1"},
		},
	}
	result := LinkChangesHasFieldChanges(changes)
	s.True(result)
}

func (s *ChangesTestSuite) Test_LinkChangesHasFieldChanges_returns_true_for_removed_fields() {
	changes := &LinkChanges{
		RemovedFields: []string{"link.field1"},
	}
	result := LinkChangesHasFieldChanges(changes)
	s.True(result)
}

func (s *ChangesTestSuite) Test_HasAnyChanges_returns_false_for_nil_changes() {
	result := HasAnyChanges(nil)
	s.False(result)
}

func (s *ChangesTestSuite) Test_HasAnyChanges_returns_false_for_empty_changes() {
	changes := &Changes{}
	result := HasAnyChanges(changes)
	s.False(result)
}

func (s *ChangesTestSuite) Test_HasAnyChanges_returns_false_for_only_unchanged_fields() {
	changes := &Changes{
		UnchangedFields: []string{"spec.field1", "spec.field2"},
	}
	result := HasAnyChanges(changes)
	s.False(result)
}

func (s *ChangesTestSuite) Test_HasAnyChanges_returns_true_for_modified_fields() {
	changes := &Changes{
		ModifiedFields: []FieldChange{
			{FieldPath: "spec.field1"},
		},
	}
	result := HasAnyChanges(changes)
	s.True(result)
}

func (s *ChangesTestSuite) Test_HasAnyChanges_returns_true_for_new_outbound_links() {
	changes := &Changes{
		NewOutboundLinks: map[string]LinkChanges{
			"resourceB": {},
		},
	}
	result := HasAnyChanges(changes)
	s.True(result)
}

func (s *ChangesTestSuite) Test_HasAnyChanges_returns_true_for_outbound_link_changes() {
	changes := &Changes{
		OutboundLinkChanges: map[string]LinkChanges{
			"resourceB": {},
		},
	}
	result := HasAnyChanges(changes)
	s.True(result)
}

func (s *ChangesTestSuite) Test_HasAnyChanges_returns_true_for_removed_outbound_links() {
	changes := &Changes{
		RemovedOutboundLinks: []string{"resourceB"},
	}
	result := HasAnyChanges(changes)
	s.True(result)
}

func TestChangesTestSuite(t *testing.T) {
	suite.Run(t, new(ChangesTestSuite))
}
