package deployconfig

import (
	"encoding/json"
	"fmt"
	"os"
)

// BluelinkDeployConfigFile is the conventional deploy configuration file name
// for a Bluelink project, found in the project root.
const BluelinkDeployConfigFile = "bluelink.deploy.jsonc"

// The file name the Celerity CLI writes its
// converted deploy configuration to, under a .celerity directory.
const celerityGeneratedConfigFile = "deploy-config.json"

// The prefix shared by the conventional Bluelink deploy
// configuration file and any per-environment variants of it.
const bluelinkConfigPrefix = "bluelink.deploy."

// BluelinkSource reads the canonical Bluelink deploy configuration format, which
// is the format the deploy engine consumes, so no conversion is needed.
type BluelinkSource struct{}

// Recognises matches the conventional name and its per-environment variants, so
// that an explicitly named bluelink.deploy.dev.jsonc is read as canonical too.
func (s *BluelinkSource) Recognises(path string) bool {
	return hasBaseNamePrefix(path, bluelinkConfigPrefix)
}

func (s *BluelinkSource) Load(path string) (*Config, error) {
	return loadCanonicalConfig(path)
}

// CelerityGeneratedSource reads the deploy configuration the Celerity CLI
// generates from its application-level configuration.
//
// The file is already in the canonical format, so it needs no conversion, but it
// only exists once a Celerity CLI command has been run and is regenerated on
// each one. It is therefore a fallback behind the authoring file it is derived
// from.
type CelerityGeneratedSource struct{}

func (s *CelerityGeneratedSource) Recognises(path string) bool {
	return hasBaseName(path, celerityGeneratedConfigFile)
}

func (s *CelerityGeneratedSource) Load(path string) (*Config, error) {
	return loadCanonicalConfig(path)
}

// loadCanonicalConfig reads a deploy configuration that is already in the format
// the deploy engine accepts. Comments and trailing commas are tolerated for
// every canonical file, since the conventional name carries a .jsonc extension.
func loadCanonicalConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading deploy config %s: %w", path, err)
	}

	config := &Config{}
	if err := json.Unmarshal([]byte(stripJSONCComments(string(data))), config); err != nil {
		return nil, fmt.Errorf("parsing deploy config %s: %w", path, err)
	}

	return config, nil
}
