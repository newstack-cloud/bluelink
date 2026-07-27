package profile

// Shared components used by all profiles.
var (
	DeployEngine = Component{
		Name:        "Deploy Engine",
		RepoOwner:   "newstack-cloud",
		RepoName:    "bluelink",
		TagPrefix:   "apps/deploy-engine",
		ArchiveName: "deploy-engine",
		BinaryName:  "deploy-engine",
		Shared:      true,
	}

	BlueprintLS = Component{
		Name:        "Blueprint LS",
		RepoOwner:   "newstack-cloud",
		RepoName:    "bluelink",
		TagPrefix:   "tools/blueprint-ls",
		ArchiveName: "blueprint-ls",
		BinaryName:  "blueprint-ls",
		Shared:      true,
	}
)

// Bluelink is the default profile for Bluelink users.
var Bluelink = Profile{
	Name:        "bluelink",
	DisplayName: "Bluelink",
	Components: []Component{
		{
			Name:        "Bluelink CLI",
			RepoOwner:   "newstack-cloud",
			RepoName:    "bluelink",
			TagPrefix:   "apps/cli",
			ArchiveName: "bluelink",
			BinaryName:  "bluelink",
			Shared:      false,
		},
		DeployEngine,
		BlueprintLS,
	},
	CorePlugins: []string{"newstack-cloud/aws"},
	CLIBinary:   "bluelink",
	RegistryURL: "https://registry.bluelink.dev",
}

// Celerity is the profile for Celerity users.
var Celerity = Profile{
	Name:        "celerity",
	DisplayName: "Celerity",
	Components: []Component{
		{
			Name:        "Celerity CLI",
			RepoOwner:   "newstack-cloud",
			RepoName:    "celerity",
			TagPrefix:   "apps/cli",
			ArchiveName: "celerity",
			BinaryName:  "celerity",
			Shared:      false,
		},
		DeployEngine,
		BlueprintLS,
	},
	CorePlugins: []string{"newstack-cloud/aws", "newstack-cloud/celerity"},
	CLIBinary:   "celerity",
	RegistryURL: "https://registry.bluelink.dev",
}
