package engine

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/config"
	"github.com/newstack-cloud/bluelink/libs/common/sigv1"
	deployengine "github.com/newstack-cloud/bluelink/libs/deploy-engine-client"
	"go.uber.org/zap"
)

// Create a new deploy engine client based on how the CLI is configured.
func Create(confProvider *config.Provider, logger *zap.Logger) (DeployEngine, error) {
	engineEndpoint, _ := confProvider.GetString("engineEndpoint")
	connectProtocol, err := getConnectProtocol(confProvider)
	if err != nil {
		return nil, err
	}

	engineAuthConfig, err := config.LoadEngineAuthConfig(confProvider)
	if err != nil {
		return nil, err
	}
	authMethod, err := toDeployEngineAuthMethod(engineAuthConfig.Method)
	if err != nil {
		return nil, err
	}

	options := []deployengine.ClientOption{
		deployengine.WithClientEndpoint(engineEndpoint),
		deployengine.WithClientConnectProtocol(connectProtocol),
		deployengine.WithClientAuthMethod(authMethod),
	}

	switch authMethod {
	case deployengine.AuthMethodAPIKey:
		options = append(options, deployengine.WithClientAPIKey(engineAuthConfig.APIKey))
	case deployengine.AuthMethodOAuth2:
		options = append(options, deployengine.WithClientOAuth2Config(&deployengine.OAuth2Config{
			ProviderBaseURL: engineAuthConfig.OAuth2.ProviderBaseURL,
			TokenEndpoint:   engineAuthConfig.OAuth2.TokenEndpoint,
			ClientID:        engineAuthConfig.OAuth2.ClientID,
			ClientSecret:    engineAuthConfig.OAuth2.ClientSecret,
		}))
	case deployengine.AuthMethodBluelinkSignatureV1:
		options = append(
			options,
			deployengine.WithClientBluelinkSigv1KeyPair(&sigv1.KeyPair{
				KeyID:     engineAuthConfig.BluelinkSignatureV1.KeyPair.KeyID,
				SecretKey: engineAuthConfig.BluelinkSignatureV1.KeyPair.SecretKey,
			}),
			deployengine.WithClientBluelinkSigv1CustomHeaders(engineAuthConfig.BluelinkSignatureV1.CustomHeaders),
		)
	}

	return deployengine.NewClient(
		options...,
	)
}

func getConnectProtocol(confProvider *config.Provider) (deployengine.ConnectProtocol, error) {
	connectProtocolStr, _ := confProvider.GetString("connectProtocol")

	switch connectProtocolStr {
	case "tcp":
		return deployengine.ConnectProtocolTCP, nil
	case "unix":
		return deployengine.ConnectProtocolUnixDomainSocket, nil
	default:
		return 0, fmt.Errorf(
			"invalid connect protocol: %s, must be either 'tcp' or 'unix'",
			connectProtocolStr,
		)
	}
}

func toDeployEngineAuthMethod(method string) (deployengine.AuthMethod, error) {
	switch method {
	case "apiKey":
		return deployengine.AuthMethodAPIKey, nil
	case "oauth2":
		return deployengine.AuthMethodOAuth2, nil
	case "bluelinkSignatureV1":
		return deployengine.AuthMethodBluelinkSignatureV1, nil
	default:
		return 0, fmt.Errorf(
			"invalid auth method: %s, must be either 'apiKey', 'oauth2' or 'bluelinkSignatureV1'",
			method,
		)
	}
}
