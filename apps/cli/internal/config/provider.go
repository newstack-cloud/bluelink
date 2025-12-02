package config

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

// Provider is a simple config provider for the CLI that can fall back to
// a config file or environment variables.
// This is an alternative approach to using the viper library that was a pain
// to integrate with cobra in providing config file and env var defaults
// across multiple levels of subcommands.
//
// The precendence of config values is as follows:
// 1. Flags
// 2. Environment variables
// 3. Config file
// 4. Flag defaults
// 5. Config provider defaults
//
// Provider only supports strings, booleans, integers and floats
// as configuration values. Complex types such as arrays, structs and maps
// are not supported.
//
// YAML, JSON and TOML are supported as config file formats.
type Provider struct {
	config   map[string]string
	pFlags   map[string]*pflag.Flag
	envVars  map[string]string
	defaults map[string]string
}

// NewProvider creates a new Provider of configuration
// values for the CLI.
func NewProvider() *Provider {
	return &Provider{
		config:   map[string]string{},
		pFlags:   map[string]*pflag.Flag{},
		envVars:  map[string]string{},
		defaults: map[string]string{},
	}
}

func (p *Provider) LoadConfigFile(configFilePath string) error {
	configFile, err := os.Open(configFilePath)
	if err != nil {
		return err
	}

	if strings.HasSuffix(configFilePath, ".yaml") || strings.HasSuffix(configFilePath, ".yml") {
		return yaml.NewDecoder(configFile).Decode(&p.config)
	}

	if strings.HasSuffix(configFilePath, ".json") {
		return json.NewDecoder(configFile).Decode(&p.config)
	}

	if strings.HasSuffix(configFilePath, ".toml") {
		_, err := toml.NewDecoder(configFile).Decode(&p.config)
		return err
	}

	return ErrUnsupportedConfigFileFormat
}

func (p *Provider) BindEnvVar(configName string, envVarName string) {
	p.envVars[configName] = envVarName
}

func (p *Provider) BindPFlag(configName string, flag *pflag.Flag) {
	p.pFlags[configName] = flag
}

func (p *Provider) SetDefault(configName, value string) {
	p.defaults[configName] = value
}

// GetString returns the value of a configuration value as a string.
// It also returns a boolean indicating whether the value was set by the user
// or if it's a default value. `true` means the value is a default value.
func (p *Provider) GetString(configName string) (string, bool) {
	flag, hasFlag := p.pFlags[configName]
	defaultFlagValue := ""
	if hasFlag && flag != nil {
		value := flag.Value.String()
		if strings.TrimSpace(value) != "" {
			if flag.Changed {
				// Flag set by user.
				return value, false
			} else {
				// Flag not set by user, fallback to default value.
				defaultFlagValue = value
			}
		}
	}

	envVarName, hasEnvVarName := p.envVars[configName]
	if hasEnvVarName {
		envVar, envVarExists := os.LookupEnv(envVarName)
		if envVarExists && strings.TrimSpace(envVar) != "" {
			return envVar, false
		}
	}

	configValue, hasConfigValue := p.config[configName]
	if hasConfigValue {
		return configValue, false
	}

	if defaultFlagValue != "" {
		return defaultFlagValue, true
	}

	return p.defaults[configName], true
}

func (p *Provider) GetInt32(configName string) (int32, bool) {
	strVal, isDefault := p.GetString(configName)
	if strVal == "" {
		return 0, isDefault
	}

	intVal, err := strconv.ParseInt(strVal, 10, 32)
	if err != nil {
		return 0, isDefault
	}

	return int32(intVal), isDefault
}

func (p *Provider) GetInt64(configName string) (int64, bool) {
	strVal, isDefault := p.GetString(configName)
	if strVal == "" {
		return 0, isDefault
	}

	intVal, err := strconv.ParseInt(strVal, 10, 64)
	if err != nil {
		return 0, isDefault
	}

	return intVal, isDefault
}

func (p *Provider) GetUint32(configName string) (uint32, bool) {
	strVal, isDefault := p.GetString(configName)
	if strVal == "" {
		return 0, isDefault
	}

	intVal, err := strconv.ParseUint(strVal, 10, 32)
	if err != nil {
		return 0, isDefault
	}

	return uint32(intVal), isDefault
}

func (p *Provider) GetUint64(configName string) (uint64, bool) {
	strVal, isDefault := p.GetString(configName)
	if strVal == "" {
		return 0, isDefault
	}

	intVal, err := strconv.ParseUint(strVal, 10, 64)
	if err != nil {
		return 0, isDefault
	}

	return intVal, isDefault
}

func (p *Provider) GetFloat32(configName string) (float32, bool) {
	strVal, isDefault := p.GetString(configName)
	if strVal == "" {
		return 0.0, isDefault
	}

	floatVal, err := strconv.ParseFloat(strVal, 32)
	if err != nil {
		return 0.0, isDefault
	}

	return float32(floatVal), isDefault
}

func (p *Provider) GetFloat64(configName string) (float64, bool) {
	strVal, isDefault := p.GetString(configName)
	if strVal == "" {
		return 0.0, isDefault
	}

	floatVal, err := strconv.ParseFloat(strVal, 64)
	if err != nil {
		return 0.0, isDefault
	}

	return floatVal, isDefault
}

func (p *Provider) GetBool(configName string) (bool, bool) {
	strVal, isDefault := p.GetString(configName)
	if strVal == "" {
		return false, isDefault
	}

	boolVal, err := strconv.ParseBool(strVal)
	if err != nil {
		return false, isDefault
	}

	return boolVal, isDefault
}

var (
	// ErrUnsupportedConfigFileFormat is returned when the config file format
	// is not supported by the provider.
	ErrUnsupportedConfigFileFormat = errors.New("unsupported config file format, only yaml, json and toml are supported")
)
