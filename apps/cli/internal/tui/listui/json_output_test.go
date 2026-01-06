package listui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/jsonout"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/stretchr/testify/suite"
)

type ListJSONOutputTestSuite struct {
	suite.Suite
	styles *stylespkg.Styles
}

func TestListJSONOutputTestSuite(t *testing.T) {
	suite.Run(t, new(ListJSONOutputTestSuite))
}

func (s *ListJSONOutputTestSuite) SetupTest() {
	s.styles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

func (s *ListJSONOutputTestSuite) Test_outputJSON_outputs_structured_list() {
	jsonOutput := &bytes.Buffer{}

	instances := []state.InstanceSummary{
		{
			InstanceID:            "instance-1",
			InstanceName:          "my-app",
			Status:                core.InstanceStatusDeployed,
			LastDeployedTimestamp: 1704067200,
		},
		{
			InstanceID:            "instance-2",
			InstanceName:          "my-other-app",
			Status:                core.InstanceStatusUpdated,
			LastDeployedTimestamp: 1704153600,
		},
	}

	model := &MainModel{
		headlessWriter: jsonOutput,
		jsonMode:       true,
		headless:       true,
		searchTerm:     "",
	}

	model.outputJSON(instances, 2)

	var output jsonout.ListInstancesOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err, "JSON output: %s", jsonOutput.String())

	s.True(output.Success)
	s.Equal(2, output.TotalCount)
	s.Len(output.Instances, 2)

	s.Equal("instance-1", output.Instances[0].InstanceID)
	s.Equal("my-app", output.Instances[0].InstanceName)
	s.Equal("DEPLOYED", output.Instances[0].Status)

	s.Equal("instance-2", output.Instances[1].InstanceID)
	s.Equal("my-other-app", output.Instances[1].InstanceName)
	s.Equal("UPDATED", output.Instances[1].Status)
}

func (s *ListJSONOutputTestSuite) Test_outputJSON_includes_search_term() {
	jsonOutput := &bytes.Buffer{}

	instances := []state.InstanceSummary{
		{
			InstanceID:            "instance-1",
			InstanceName:          "prod-app",
			Status:                core.InstanceStatusDeployed,
			LastDeployedTimestamp: 1704067200,
		},
	}

	model := &MainModel{
		headlessWriter: jsonOutput,
		jsonMode:       true,
		headless:       true,
		searchTerm:     "prod",
	}

	model.outputJSON(instances, 1)

	var output jsonout.ListInstancesOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.Equal("prod", output.Search)
	s.Len(output.Instances, 1)
}

func (s *ListJSONOutputTestSuite) Test_outputJSON_empty_list() {
	jsonOutput := &bytes.Buffer{}

	model := &MainModel{
		headlessWriter: jsonOutput,
		jsonMode:       true,
		headless:       true,
	}

	model.outputJSON(nil, 0)

	var output jsonout.ListInstancesOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.True(output.Success)
	s.Equal(0, output.TotalCount)
	s.Empty(output.Instances)
}

func (s *ListJSONOutputTestSuite) Test_outputJSONError() {
	jsonOutput := &bytes.Buffer{}

	model := &MainModel{
		headlessWriter: jsonOutput,
		jsonMode:       true,
		headless:       true,
	}

	model.outputJSONError(fmt.Errorf("instance not found"))

	var output jsonout.ErrorOutput
	err := json.Unmarshal(jsonOutput.Bytes(), &output)
	s.Require().NoError(err)

	s.False(output.Success)
	s.Contains(output.Error.Message, "instance not found")
}
