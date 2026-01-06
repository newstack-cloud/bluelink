package commands

import (
	"errors"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/cmd/utils"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/jsonout"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/inspectui"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/listui"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var errInspectFailed = errors.New("inspect failed")
var errListFailed = errors.New("list instances failed")

func setupInstancesCommand(rootCmd *cobra.Command, confProvider *config.Provider) {
	instancesCmd := &cobra.Command{
		Use:   "instances",
		Short: "Manage and view blueprint instances",
		Long: `Commands for managing and viewing blueprint instances deployed via the
deploy engine. Use subcommands to list, inspect, or manage instances.`,
	}

	setupInstancesInspectCommand(instancesCmd, confProvider)
	setupInstancesListCommand(instancesCmd, confProvider)

	rootCmd.AddCommand(instancesCmd)
}

func setupInstancesInspectCommand(instancesCmd *cobra.Command, confProvider *config.Provider) {
	inspectCmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect a blueprint instance",
		Long: `Displays the current state of a blueprint instance including resources,
links, child blueprints, and deployment status.

If a deployment or destroy operation is currently in progress, the command
streams real-time updates until the operation completes.

Examples:
  # Interactive mode - enter instance name when prompted
  bluelink instances inspect

  # Inspect by instance name
  bluelink instances inspect --instance-name my-app

  # Inspect by instance ID
  bluelink instances inspect --instance-id abc123

  # Output as JSON (useful for CI/CD or scripting)
  bluelink instances inspect --instance-name my-app --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspect(cmd, confProvider)
		},
	}

	inspectCmd.PersistentFlags().String(
		"instance-id",
		"",
		"The system-generated ID of the blueprint instance to inspect. "+
			"Leave empty if using --instance-name.",
	)
	confProvider.BindPFlag("instancesInspectInstanceID", inspectCmd.PersistentFlags().Lookup("instance-id"))
	confProvider.BindEnvVar("instancesInspectInstanceID", "BLUELINK_CLI_INSTANCES_INSPECT_INSTANCE_ID")

	inspectCmd.PersistentFlags().String(
		"instance-name",
		"",
		"The user-defined unique name of the blueprint instance to inspect. "+
			"Leave empty if using --instance-id.",
	)
	confProvider.BindPFlag("instancesInspectInstanceName", inspectCmd.PersistentFlags().Lookup("instance-name"))
	confProvider.BindEnvVar("instancesInspectInstanceName", "BLUELINK_CLI_INSTANCES_INSPECT_INSTANCE_NAME")

	inspectCmd.PersistentFlags().Bool(
		"json",
		false,
		"Output the instance state as JSON. "+
			"Implies non-interactive mode (no TUI).",
	)
	confProvider.BindPFlag("instancesInspectJson", inspectCmd.PersistentFlags().Lookup("json"))
	confProvider.BindEnvVar("instancesInspectJson", "BLUELINK_CLI_INSTANCES_INSPECT_JSON")

	instancesCmd.AddCommand(inspectCmd)
}

func runInspect(cmd *cobra.Command, confProvider *config.Provider) error {
	logger, handle, err := utils.SetupLogger()
	if err != nil {
		return err
	}
	defer handle.Close()

	deployEngine, err := engine.Create(confProvider, logger)
	if err != nil {
		return err
	}

	instanceID, instanceIDIsDefault := confProvider.GetString("instancesInspectInstanceID")
	instanceName, instanceNameIsDefault := confProvider.GetString("instancesInspectInstanceName")
	jsonMode, _ := confProvider.GetBool("instancesInspectJson")

	if jsonMode {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
	}

	instanceNameFlag := headless.Flag{
		Name:      "instance-name",
		Value:     instanceName,
		IsDefault: instanceNameIsDefault,
	}
	instanceIDFlag := headless.Flag{
		Name:      "instance-id",
		Value:     instanceID,
		IsDefault: instanceIDIsDefault,
	}

	// In JSON mode, require an instance identifier (can't prompt for input)
	if jsonMode {
		oneOfRule := headless.OneOf(instanceNameFlag, instanceIDFlag)
		if err := oneOfRule.Validate(); err != nil {
			jsonout.WriteJSON(os.Stdout, jsonout.NewErrorOutput(err))
			return errInspectFailed
		}
	}

	// Standard headless validation (for non-terminal environments)
	err = headless.Validate(headless.OneOf(instanceNameFlag, instanceIDFlag))
	if err != nil {
		if jsonMode {
			jsonout.WriteJSON(os.Stdout, jsonout.NewErrorOutput(err))
			return errInspectFailed
		}
		return err
	}

	if _, err := tea.LogToFile("bluelink-output.log", "simple"); err != nil {
		log.Fatal(err)
	}

	cmd.SilenceUsage = true

	styles := stylespkg.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		stylespkg.NewBluelinkPalette(),
	)
	inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	headlessMode := !inTerminal || jsonMode

	app, err := inspectui.NewInspectApp(
		deployEngine,
		logger,
		instanceID,
		instanceName,
		styles,
		headlessMode,
		os.Stdout,
		jsonMode,
	)
	if err != nil {
		return err
	}

	options := []tea.ProgramOption{}
	if !headlessMode {
		options = append(options, tea.WithAltScreen(), tea.WithMouseCellMotion())
	} else {
		options = append(options, tea.WithInput(nil), tea.WithoutRenderer())
	}

	finalModel, err := tea.NewProgram(app, options...).Run()
	if err != nil {
		return err
	}
	finalApp := finalModel.(inspectui.MainModel)

	if finalApp.Error != nil {
		cmd.SilenceErrors = true
		return errInspectFailed
	}

	return nil
}

func setupInstancesListCommand(instancesCmd *cobra.Command, confProvider *config.Provider) {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List blueprint instances",
		Long: `Lists all blueprint instances managed by the deploy engine.

In interactive mode, the list is paginated and you can filter instances using search.
Selecting an instance navigates to the inspect view.

Examples:
  # Interactive mode - browse and filter instances
  bluelink instances list

  # Filter instances by name
  bluelink instances list --search "production"

  # Output as JSON (useful for CI/CD or scripting)
  bluelink instances list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListInstances(cmd, confProvider)
		},
	}

	listCmd.PersistentFlags().String(
		"search",
		"",
		"Filter instances by name (case-insensitive substring match).",
	)
	confProvider.BindPFlag("instancesListSearch", listCmd.PersistentFlags().Lookup("search"))
	confProvider.BindEnvVar("instancesListSearch", "BLUELINK_CLI_INSTANCES_LIST_SEARCH")

	listCmd.PersistentFlags().Bool(
		"json",
		false,
		"Output the instance list as JSON. Implies non-interactive mode.",
	)
	confProvider.BindPFlag("instancesListJson", listCmd.PersistentFlags().Lookup("json"))
	confProvider.BindEnvVar("instancesListJson", "BLUELINK_CLI_INSTANCES_LIST_JSON")

	instancesCmd.AddCommand(listCmd)
}

func runListInstances(cmd *cobra.Command, confProvider *config.Provider) error {
	logger, handle, err := utils.SetupLogger()
	if err != nil {
		return err
	}
	defer handle.Close()

	deployEngine, err := engine.Create(confProvider, logger)
	if err != nil {
		return err
	}

	search, _ := confProvider.GetString("instancesListSearch")
	jsonMode, _ := confProvider.GetBool("instancesListJson")

	if jsonMode {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
	}

	cmd.SilenceUsage = true

	styles := stylespkg.NewStyles(
		lipgloss.NewRenderer(os.Stdout),
		stylespkg.NewBluelinkPalette(),
	)
	inTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	headlessMode := !inTerminal || jsonMode

	app, err := listui.NewListApp(
		deployEngine,
		logger,
		search,
		styles,
		headlessMode,
		os.Stdout,
		jsonMode,
	)
	if err != nil {
		return err
	}

	options := []tea.ProgramOption{}
	if !headlessMode {
		options = append(options, tea.WithAltScreen(), tea.WithMouseCellMotion())
	} else {
		options = append(options, tea.WithInput(nil), tea.WithoutRenderer())
	}

	finalModel, err := tea.NewProgram(app, options...).Run()
	if err != nil {
		return err
	}
	listApp := finalModel.(listui.MainModel)

	if listApp.Error != nil {
		cmd.SilenceErrors = true
		return errListFailed
	}

	return nil
}
