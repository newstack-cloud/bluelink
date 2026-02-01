package commands

import (
	"fmt"
	"os"
	"runtime"

	"github.com/newstack-cloud/bluelink/apps/cli/cmd/utils"
	"github.com/newstack-cloud/deploy-cli-sdk/config"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	var configFile string

	confProvider := config.NewProvider()

	cobra.AddTemplateFunc("wrappedFlagUsages", utils.WrappedFlagUsages)
	cobra.AddTemplateFunc("versionInfo", utils.VersionInfo)
	rootCmd := &cobra.Command{
		Use:   "bluelink",
		Short: "CLI for managing blueprint deployments and plugins",
		Long: `The CLI for managing and deploying your infrastructure blueprints.
This CLI validates, stages changes for, and deploys blueprints.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			loadConfig := cmd.Flags().Lookup("config").Changed
			if !loadConfig {
				if _, statErr := os.Stat(configFile); statErr == nil {
					loadConfig = true
				}
			}
			if loadConfig {
				if err := confProvider.LoadConfigFile(configFile); err != nil {
					return err
				}
			}

			connectProtocol, _ := confProvider.GetString("connectProtocol")
			err := validateConnectProtocol(connectProtocol)
			if err != nil {
				return err
			}

			return nil
		},
	}

	rootCmd.SetUsageTemplate(utils.UsageTemplate)
	rootCmd.SetHelpTemplate(utils.HelpTemplate)

	rootCmd.PersistentFlags().StringVar(
		&configFile,
		"config",
		"bluelink.config.toml",
		"Specify a config file to source config from as an alternative to flags",
	)

	rootCmd.PersistentFlags().String(
		"deploy-config-file",
		"bluelink.deploy.json",
		"The path to the deployment configuration JSON file that will be used as "+
			"a source of blueprint variable overrides, provider configuration, "+
			"transformer configuration and general configuration. "+
			"The contents of this file is sent in requests to the deploy engine for "+
			"validation, change staging and deployment.",
	)
	confProvider.BindPFlag("deployConfigFile", rootCmd.PersistentFlags().Lookup("deploy-config-file"))
	confProvider.BindEnvVar("deployConfigFile", "BLUELINK_CLI_DEPLOY_CONFIG_FILE")

	rootCmd.PersistentFlags().String(
		"connect-protocol",
		// Connect to a local instance of the deploy engine
		// via a unix socket by default.
		"unix",
		"The protocol to connect to the deploy engine with, "+
			"this can be either \"unix\" or \"tcp\". A unix socket can only be used on linux, macos, and other unix-like operating systems. "+
			"To use a \"unix\" socket on windows, you will need to use WSL 2 or above.",
	)
	confProvider.BindPFlag("connectProtocol", rootCmd.PersistentFlags().Lookup("connect-protocol"))
	confProvider.BindEnvVar("connectProtocol", "BLUELINK_CLI_CONNECT_PROTOCOL")

	rootCmd.PersistentFlags().String(
		"engine-endpoint",
		"http://localhost:8325",
		"The endpoint of the deploy engine api, this is used if --connect-protocol is set to \"tcp\"",
	)
	confProvider.BindPFlag("engineEndpoint", rootCmd.PersistentFlags().Lookup("engine-endpoint"))
	confProvider.BindEnvVar("engineEndpoint", "BLUELINK_CLI_ENGINE_ENDPOINT")

	rootCmd.PersistentFlags().String(
		"engine-auth-config-file",
		"engine.auth.json",
		"The path to the authentication configuration file to use to connect to the deploy engine, this must be a JSON file.",
	)
	confProvider.BindPFlag("engineAuthConfigFile", rootCmd.PersistentFlags().Lookup("engine-auth-config-file"))
	confProvider.BindEnvVar("engineAuthConfigFile", "BLUELINK_CLI_ENGINE_AUTH_CONFIG_FILE")

	rootCmd.PersistentFlags().Bool(
		"skip-plugin-config-validation",
		false,
		"Skip validation of the plugin-specific entries in the deploy configuration file for commands that interact with the deploy engine.",
	)
	confProvider.BindPFlag("skipPluginConfigValidation", rootCmd.PersistentFlags().Lookup("skip-plugin-config-validation"))
	confProvider.BindEnvVar("skipPluginConfigValidation", "BLUELINK_CLI_SKIP_PLUGIN_CONFIG_VALIDATION")

	rootCmd.PersistentFlags().Bool(
		"skip-plugin-check",
		false,
		"Skip the automatic plugin dependency check before "+
			"validate, stage, deploy, and destroy commands.",
	)
	confProvider.BindPFlag("skipPluginCheck", rootCmd.PersistentFlags().Lookup("skip-plugin-check"))
	confProvider.BindEnvVar("skipPluginCheck", "BLUELINK_CLI_SKIP_PLUGIN_CHECK")

	setupVersionCommand(rootCmd)
	setupInitCommand(rootCmd, confProvider)
	setupValidateCommand(rootCmd, confProvider)
	setupStageCommand(rootCmd, confProvider)
	setupDeployCommand(rootCmd, confProvider)
	setupDestroyCommand(rootCmd, confProvider)
	setupInstancesCommand(rootCmd, confProvider)
	setupStateCommand(rootCmd, confProvider)
	setupCleanupCommand(rootCmd, confProvider)
	setupPluginsCommand(rootCmd, confProvider)
	setupTemplatesCommand(rootCmd, confProvider)

	return rootCmd
}

func validateConnectProtocol(protocol string) error {
	if protocol == "tcp" {
		return nil
	}

	if protocol == "unix" {
		os := runtime.GOOS
		if os == "windows" {
			return fmt.Errorf(
				"\"unix\" socket is not supported on windows, please use \"tcp\" " +
					"or set up Windows Subsystem for Linux (WSL) version 2 or above to use a unix socket",
			)
		}

		return nil
	}

	return fmt.Errorf(
		"invalid connect protocol \"%s\" provided, must be either \"unix\" or \"tcp\"",
		protocol,
	)
}
