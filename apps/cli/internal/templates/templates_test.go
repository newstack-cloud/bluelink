package templates

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type TemplatesSuite struct {
	suite.Suite
}

func TestTemplatesSuite(t *testing.T) {
	suite.Run(t, new(TemplatesSuite))
}

func (s *TemplatesSuite) Test_get_templates_returns_non_empty() {
	templates := GetTemplates()
	s.NotEmpty(templates, "GetTemplates should return at least one template")
}

func (s *TemplatesSuite) Test_templates_have_required_fields() {
	templates := GetTemplates()
	for _, t := range templates {
		s.NotEmpty(t.Key, "Template Key must not be empty")
		s.NotEmpty(t.Label, "Template Label must not be empty")
		s.NotEmpty(t.Description, "Template Description must not be empty")
	}
}

func (s *TemplatesSuite) Test_template_keys_are_unique() {
	templates := GetTemplates()
	seen := make(map[string]bool, len(templates))
	for _, t := range templates {
		s.False(seen[t.Key], "Duplicate template key: %s", t.Key)
		seen[t.Key] = true
	}
}
