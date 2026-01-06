package listui

import (
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/stretchr/testify/suite"
)

type ListItemsTestSuite struct {
	suite.Suite
	testStyles *styles.Styles
}

func TestListItemsTestSuite(t *testing.T) {
	suite.Run(t, new(ListItemsTestSuite))
}

func (s *ListItemsTestSuite) SetupTest() {
	s.testStyles = styles.NewStyles(lipgloss.NewRenderer(os.Stdout), styles.NewBluelinkPalette())
}

// renderStatus tests

func (s *ListItemsTestSuite) Test_renderStatus_deployed() {
	result := renderStatus(core.InstanceStatusDeployed, s.testStyles)
	s.Contains(result, "deployed")
}

func (s *ListItemsTestSuite) Test_renderStatus_updated() {
	result := renderStatus(core.InstanceStatusUpdated, s.testStyles)
	s.Contains(result, "updated")
}

func (s *ListItemsTestSuite) Test_renderStatus_deploying() {
	result := renderStatus(core.InstanceStatusDeploying, s.testStyles)
	s.Contains(result, "deploying")
}

func (s *ListItemsTestSuite) Test_renderStatus_updating() {
	result := renderStatus(core.InstanceStatusUpdating, s.testStyles)
	s.Contains(result, "updating")
}

func (s *ListItemsTestSuite) Test_renderStatus_destroying() {
	result := renderStatus(core.InstanceStatusDestroying, s.testStyles)
	s.Contains(result, "destroying")
}

func (s *ListItemsTestSuite) Test_renderStatus_deploy_failed() {
	result := renderStatus(core.InstanceStatusDeployFailed, s.testStyles)
	s.Contains(result, "deploy failed")
}

func (s *ListItemsTestSuite) Test_renderStatus_update_failed() {
	result := renderStatus(core.InstanceStatusUpdateFailed, s.testStyles)
	s.Contains(result, "update failed")
}

func (s *ListItemsTestSuite) Test_renderStatus_destroy_failed() {
	result := renderStatus(core.InstanceStatusDestroyFailed, s.testStyles)
	s.Contains(result, "destroy failed")
}

func (s *ListItemsTestSuite) Test_renderStatus_destroyed() {
	result := renderStatus(core.InstanceStatusDestroyed, s.testStyles)
	s.Contains(result, "destroyed")
}

func (s *ListItemsTestSuite) Test_renderStatus_unknown_uses_string() {
	result := renderStatus(core.InstanceStatusPreparing, s.testStyles)
	s.Contains(result, "PREPARING")
}

// formatTimestamp tests

func (s *ListItemsTestSuite) Test_formatTimestamp_zero_returns_never() {
	result := formatTimestamp(0)
	s.Equal("Never", result)
}

func (s *ListItemsTestSuite) Test_formatTimestamp_formats_unix_timestamp() {
	// Date far in the past shows "Mon DD, YYYY" format
	timestamp := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC).Unix()
	result := formatTimestamp(timestamp)
	s.Equal("Mar 15, 2024", result)
}

func (s *ListItemsTestSuite) Test_formatTimestamp_just_now() {
	timestamp := time.Now().Unix()
	result := formatTimestamp(timestamp)
	s.Equal("Just now", result)
}

func (s *ListItemsTestSuite) Test_formatTimestamp_minutes_ago() {
	timestamp := time.Now().Add(-5 * time.Minute).Unix()
	result := formatTimestamp(timestamp)
	s.Equal("5 minutes ago", result)
}

func (s *ListItemsTestSuite) Test_formatTimestamp_one_minute_ago() {
	timestamp := time.Now().Add(-90 * time.Second).Unix()
	result := formatTimestamp(timestamp)
	s.Equal("1 minute ago", result)
}

func (s *ListItemsTestSuite) Test_formatTimestamp_hours_ago() {
	timestamp := time.Now().Add(-3 * time.Hour).Unix()
	result := formatTimestamp(timestamp)
	s.Equal("3 hours ago", result)
}

func (s *ListItemsTestSuite) Test_formatTimestamp_one_hour_ago() {
	timestamp := time.Now().Add(-90 * time.Minute).Unix()
	result := formatTimestamp(timestamp)
	s.Equal("1 hour ago", result)
}

func (s *ListItemsTestSuite) Test_formatTimestamp_yesterday() {
	timestamp := time.Now().Add(-36 * time.Hour).Unix()
	result := formatTimestamp(timestamp)
	s.Equal("Yesterday", result)
}

func (s *ListItemsTestSuite) Test_formatTimestamp_days_ago() {
	timestamp := time.Now().Add(-5 * 24 * time.Hour).Unix()
	result := formatTimestamp(timestamp)
	s.Equal("5 days ago", result)
}

// padRight tests

func (s *ListItemsTestSuite) Test_padRight_adds_spaces() {
	result := padRight("test", 10)
	s.Equal("test      ", result)
	s.Len(result, 10)
}

func (s *ListItemsTestSuite) Test_padRight_returns_original_if_longer() {
	result := padRight("longstring", 5)
	s.Equal("longstring", result)
}

func (s *ListItemsTestSuite) Test_padRight_returns_original_if_equal() {
	result := padRight("test", 4)
	s.Equal("test", result)
}

// spaces tests

func (s *ListItemsTestSuite) Test_spaces_returns_n_spaces() {
	result := spaces(5)
	s.Equal("     ", result)
}

func (s *ListItemsTestSuite) Test_spaces_returns_empty_for_zero() {
	result := spaces(0)
	s.Equal("", result)
}

func (s *ListItemsTestSuite) Test_spaces_returns_empty_for_negative() {
	result := spaces(-3)
	s.Equal("", result)
}

// itoa tests

func (s *ListItemsTestSuite) Test_itoa_converts_positive_int() {
	result := itoa(42)
	s.Equal("42", result)
}

func (s *ListItemsTestSuite) Test_itoa_converts_zero() {
	result := itoa(0)
	s.Equal("0", result)
}

func (s *ListItemsTestSuite) Test_itoa_converts_negative() {
	result := itoa(-5)
	s.Equal("-5", result)
}
