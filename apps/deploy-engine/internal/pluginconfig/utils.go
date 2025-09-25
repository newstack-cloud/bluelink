package pluginconfig

import (
	"fmt"
	"strconv"
	"strings"
)

// ToConfigDefinitionProviders is a helper function that converts a map of
// plugins (either transformers or providers) to a map of the simplified
// PluginConfigDefinitionProvider interface.
// Maps and slices of an interface aren't interchangeable between super
// type and subtypes.
func ToConfigDefinitionProviders[PluginSuperType DefinitionProvider](
	providers map[string]PluginSuperType,
) map[string]DefinitionProvider {
	converted := make(map[string]DefinitionProvider, len(providers))
	for k, v := range providers {
		converted[k] = v
	}
	return converted
}

// CheckPluginVersionCompatibility checks if the installed plugin version
// is compatible with the given version constraint.
// Version constraints are in the format of:
//   - `^1.2.3` for minor range constraint
//   - `~1.2.3` for patch range constraint
//   - `1.2.3` for exact version
func CheckPluginVersionCompatibility(
	installedPluginVersion string,
	versionConstraint string,
) (bool, error) {
	installedVersionParts, err := extractVersionConstraintParts(installedPluginVersion)
	if err != nil {
		return false, err
	}

	versionConstraintParts, err := extractVersionConstraintParts(versionConstraint)
	if err != nil {
		return false, err
	}

	if installedVersionParts.major != versionConstraintParts.major {
		return false, nil
	}

	if versionConstraintParts.minorRangeConstraint &&
		installedVersionParts.minor < versionConstraintParts.minor {
		return false, nil
	} else if versionConstraintParts.patchRangeConstraint &&
		installedVersionParts.patch < versionConstraintParts.patch {
		return false, nil
	}

	return true, nil
}

type pluginVersionParts struct {
	major                int
	minor                int
	patch                int
	minorRangeConstraint bool
	patchRangeConstraint bool
}

func extractVersionConstraintParts(versionConstraint string) (pluginVersionParts, error) {
	parts := pluginVersionParts{}

	var semver string
	var hasPrefix bool
	if semver, hasPrefix = strings.CutPrefix(versionConstraint, "^"); hasPrefix {
		parts.minorRangeConstraint = true
	} else if semver, hasPrefix = strings.CutPrefix(versionConstraint, "~"); hasPrefix {
		parts.patchRangeConstraint = true
	}

	withoutPrerelease := strings.Split(semver, "-")[0]
	semverParts := strings.Split(withoutPrerelease, ".")
	if len(semverParts) == 3 {
		majorVersion, err := strconv.Atoi(semverParts[0])
		if err != nil {
			return pluginVersionParts{}, errInvalidVersionOrConstraintFormat(
				versionConstraint,
			)
		}

		minorVersion, err := strconv.Atoi(semverParts[1])
		if err != nil {
			return pluginVersionParts{}, errInvalidVersionOrConstraintFormat(
				versionConstraint,
			)
		}

		patchVersion, err := strconv.Atoi(semverParts[2])
		if err != nil {
			return pluginVersionParts{}, errInvalidVersionOrConstraintFormat(
				versionConstraint,
			)
		}

		parts.major = majorVersion
		parts.minor = minorVersion
		parts.patch = patchVersion
		return parts, nil
	}

	return pluginVersionParts{}, errInvalidVersionOrConstraintFormat(
		versionConstraint,
	)
}

func errInvalidVersionOrConstraintFormat(versionConstraint string) error {
	return fmt.Errorf("invalid version or constraint format: %s", versionConstraint)
}
