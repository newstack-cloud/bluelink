package shared

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/stretchr/testify/suite"
)

type ActionsTestSuite struct {
	suite.Suite
	testStyles *styles.Styles
}

func TestActionsTestSuite(t *testing.T) {
	suite.Run(t, new(ActionsTestSuite))
}

func (s *ActionsTestSuite) SetupTest() {
	s.testStyles = styles.NewStyles(lipgloss.NewRenderer(os.Stdout), styles.NewBluelinkPalette())
}

func (s *ActionsTestSuite) Test_DetermineResourceAction_returns_CREATE_for_new() {
	action := DetermineResourceAction(true, false, false, nil)
	s.Equal(ActionCreate, action)
}

func (s *ActionsTestSuite) Test_DetermineResourceAction_returns_DELETE_for_removed() {
	action := DetermineResourceAction(false, true, false, nil)
	s.Equal(ActionDelete, action)
}

func (s *ActionsTestSuite) Test_DetermineResourceAction_returns_RECREATE_for_recreate() {
	action := DetermineResourceAction(false, false, true, nil)
	s.Equal(ActionRecreate, action)
}

func (s *ActionsTestSuite) Test_DetermineResourceAction_returns_UPDATE_for_changes() {
	changes := &provider.Changes{
		ModifiedFields: []provider.FieldChange{
			{FieldPath: "spec.replicas"},
		},
	}
	action := DetermineResourceAction(false, false, false, changes)
	s.Equal(ActionUpdate, action)
}

func (s *ActionsTestSuite) Test_DetermineResourceAction_returns_NOCHANGE_for_no_changes() {
	action := DetermineResourceAction(false, false, false, nil)
	s.Equal(ActionNoChange, action)
}

func (s *ActionsTestSuite) Test_DetermineResourceAction_returns_NOCHANGE_for_empty_changes() {
	action := DetermineResourceAction(false, false, false, &provider.Changes{})
	s.Equal(ActionNoChange, action)
}

func (s *ActionsTestSuite) Test_DetermineResourceAction_new_takes_precedence_over_removed() {
	action := DetermineResourceAction(true, true, false, nil)
	s.Equal(ActionCreate, action)
}

func (s *ActionsTestSuite) Test_DetermineResourceAction_removed_takes_precedence_over_recreate() {
	action := DetermineResourceAction(false, true, true, nil)
	s.Equal(ActionDelete, action)
}

func (s *ActionsTestSuite) Test_ActionIcon_returns_correct_icons() {
	s.Equal("✓", ActionIcon(ActionCreate))
	s.Equal("±", ActionIcon(ActionUpdate))
	s.Equal("-", ActionIcon(ActionDelete))
	s.Equal("↻", ActionIcon(ActionRecreate))
	s.Equal("○", ActionIcon(ActionNoChange))
}

func (s *ActionsTestSuite) Test_StyledActionIcon_without_style_returns_plain_icon() {
	icon := StyledActionIcon(ActionCreate, nil, false)
	s.Equal("✓", icon)
}

func (s *ActionsTestSuite) Test_StyledActionIcon_with_style_returns_styled_icon() {
	icon := StyledActionIcon(ActionCreate, s.testStyles, true)
	s.Contains(icon, "✓")
}

func (s *ActionsTestSuite) Test_RenderActionBadge_includes_action_text() {
	s.Contains(RenderActionBadge(ActionCreate, s.testStyles), "CREATE")
	s.Contains(RenderActionBadge(ActionUpdate, s.testStyles), "UPDATE")
	s.Contains(RenderActionBadge(ActionDelete, s.testStyles), "DELETE")
	s.Contains(RenderActionBadge(ActionRecreate, s.testStyles), "RECREATE")
	s.Contains(RenderActionBadge(ActionNoChange, s.testStyles), "NO CHANGE")
}
