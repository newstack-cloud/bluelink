#!/bin/bash

set -e

POSITIONAL=()
PLUGIN_SPECS=()
ONLY=()
while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    -h|--help)
    HELP=yes
    shift # past argument
    ;;
    --provider)
    PLUGIN_SPECS+=("providers|$2")
    shift # past argument
    shift # past value
    ;;
    --transformer)
    PLUGIN_SPECS+=("transformers|$2")
    shift # past argument
    shift # past value
    ;;
    --config)
    CONFIG_FILE="$2"
    shift # past argument
    shift # past value
    ;;
    --only)
    ONLY+=("$2")
    shift # past argument
    shift # past value
    ;;
    --list)
    LIST=yes
    shift # past argument
    ;;
    *)    # unknown option
    POSITIONAL+=("$1") # save it in an array for later
    shift # past argument
    ;;
esac
done
set -- "${POSITIONAL[@]}" # restore positional parameters

DEPLOY_ENGINE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLUGIN_BIN_DIR="$DEPLOY_ENGINE_DIR/.bluelink/deploy-engine/plugins/bin"
DEFAULT_CONFIG_FILE="${BLUELINK_LOCAL_PLUGINS_CONFIG:-$DEPLOY_ENGINE_DIR/local-plugins.conf}"

function help {
  cat << EOF
Local plugin builder for the deploy engine

Builds plugins from their source repositories and installs them into the local
plugin directory tree used by the deploy engine when running locally:

  .bluelink/deploy-engine/plugins/bin/providers/<namespace>/<name>/<version>/plugin
  .bluelink/deploy-engine/plugins/bin/transformers/<namespace>/<name>/<version>/plugin

Plugins are given as specs of the form:

  <namespace>/<name>=<source directory>[@<version>]

The source directory is the directory holding the plugin's main package, it can
be anywhere on the machine and a leading "~/" is expanded. When no version is
given it is derived from the latest git tag in the source directory's
repository, falling back to 0.0.0-dev when there are no tags. The version is
used both for the install directory and for the version compiled into the
plugin binary via -ldflags "-X main.version=<version>".

To build plugins passed on the command line:
bash scripts/build-local-plugins.sh \\
  --provider newstack-cloud/aws=~/repos/bluelink-provider-aws \\
  --transformer newstack-cloud/celerity=~/repos/bluelink-transformer-celerity

To pin a version instead of deriving it from git tags:
bash scripts/build-local-plugins.sh --provider newstack-cloud/aws=~/repos/aws@0.4.2

Both options can be repeated to build any number of plugins.

When no --provider or --transformer option is given, plugin specs are read from
a config file so the locations on your machine do not have to be retyped:

  $DEFAULT_CONFIG_FILE

Override the config file location with --config <file> or the
BLUELINK_LOCAL_PLUGINS_CONFIG environment variable. See
local-plugins.example.conf for the file format.

To build only some of the plugins in the config file:
bash scripts/build-local-plugins.sh --only newstack-cloud/aws

To list the plugins that would be built without building them:
bash scripts/build-local-plugins.sh --list
EOF
}

if [ -n "$HELP" ]; then
  help
  exit 0
fi

function fail {
  echo "Error: $1" >&2
  exit 1
}

# Expands a leading "~/" in a path, config files are read rather than evaluated
# by the shell so tilde expansion does not happen for free.
function expand_path {
  local path="$1"

  case "$path" in
    "~/"*) echo "$HOME/${path#\~/}" ;;
    *) echo "$path" ;;
  esac
}

# Derives a version for a plugin from the latest git tag in the repository the
# source directory belongs to, stripping any leading "v" and falling back to a
# development version when there are no tags.
function resolve_version {
  local source_dir="$1"
  local version

  version="$(git -C "$source_dir" describe --tags --abbrev=0 2>/dev/null || true)"
  if [ -z "$version" ]; then
    echo "0.0.0-dev"
    return
  fi

  echo "${version#v}"
}

# Reads plugin specs from a config file, lines are of the form
# "provider <spec>" or "transformer <spec>", blank lines and lines starting
# with "#" are ignored.
function load_config {
  local config_file="$1"
  local line kind spec

  while IFS= read -r line || [ -n "$line" ]; do
    line="$(echo "$line" | sed -e 's/#.*//' -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
    kind="${line%%[[:space:]]*}"
    spec="$(echo "${line#$kind}" | sed -e 's/^[[:space:]]*//')"
    if [ -z "$kind" ]; then
      continue
    fi
    if [ -z "$spec" ]; then
      fail "missing plugin spec for \"$kind\" in $config_file"
    fi

    case "$kind" in
      provider) PLUGIN_SPECS+=("providers|$spec") ;;
      transformer) PLUGIN_SPECS+=("transformers|$spec") ;;
      *) fail "unknown plugin kind \"$kind\" in $config_file, expected \"provider\" or \"transformer\"" ;;
    esac
  done < "$config_file"
}

function build_plugin {
  local plugin_type="$1"
  local qualified_name="$2"
  local source_dir="$3"
  local version="$4"

  local install_dir="$PLUGIN_BIN_DIR/$plugin_type/$qualified_name/$version"

  echo "Building $qualified_name $version from $source_dir ..."
  mkdir -p "$install_dir"
  (
    cd "$source_dir"
    go build -ldflags "-X main.version=$version" -o "$install_dir/plugin" .
  )
  echo "Installed plugin at $install_dir/plugin"
}

# Splits a plugin spec into its qualified name, source directory and version,
# validating the parts and filling in the version when it is not pinned in the
# spec. The parts are reported through globals as bash functions can not return
# multiple values.
function parse_spec {
  local spec="$1"
  local location

  SPEC_NAME="${spec%%=*}"
  location="${spec#*=}"

  if [ -z "$SPEC_NAME" ] || [ "$SPEC_NAME" = "$spec" ] || [ -z "$location" ]; then
    fail "invalid plugin spec \"$spec\", expected <namespace>/<name>=<source directory>[@<version>]"
  fi

  case "$SPEC_NAME" in
    */*/*|*/) fail "invalid plugin name \"$SPEC_NAME\" in \"$spec\", expected <namespace>/<name>" ;;
    */*) ;;
    *) fail "invalid plugin name \"$SPEC_NAME\" in \"$spec\", expected <namespace>/<name>" ;;
  esac

  SPEC_VERSION=""
  case "$location" in
    *@*)
      SPEC_VERSION="${location##*@}"
      location="${location%@*}"
      ;;
  esac

  SPEC_DIR="$(expand_path "$location")"
  if [ ! -d "$SPEC_DIR" ]; then
    fail "source directory for $SPEC_NAME not found at $SPEC_DIR"
  fi

  if [ -z "$SPEC_VERSION" ]; then
    SPEC_VERSION="$(resolve_version "$SPEC_DIR")"
  fi
}

function included {
  local qualified_name="$1"
  local only

  if [ ${#ONLY[@]} -eq 0 ]; then
    return 0
  fi

  for only in "${ONLY[@]}"; do
    if [ "$only" = "$qualified_name" ]; then
      return 0
    fi
  done

  return 1
}

if [ ${#PLUGIN_SPECS[@]} -eq 0 ]; then
  CONFIG_FILE="${CONFIG_FILE:-$DEFAULT_CONFIG_FILE}"
  if [ ! -f "$CONFIG_FILE" ]; then
    fail "no plugins given and no config file at $CONFIG_FILE, see \"bash scripts/build-local-plugins.sh --help\""
  fi
  load_config "$CONFIG_FILE"
fi

if [ ${#PLUGIN_SPECS[@]} -eq 0 ]; then
  fail "no plugins to build"
fi

BUILT=0
for entry in "${PLUGIN_SPECS[@]}"; do
  PLUGIN_TYPE="${entry%%|*}"
  parse_spec "${entry#*|}"

  if ! included "$SPEC_NAME"; then
    continue
  fi

  BUILT=$((BUILT + 1))
  if [ -n "$LIST" ]; then
    echo "$PLUGIN_TYPE $SPEC_NAME $SPEC_VERSION $SPEC_DIR"
    continue
  fi

  build_plugin "$PLUGIN_TYPE" "$SPEC_NAME" "$SPEC_DIR" "$SPEC_VERSION"
done

if [ "$BUILT" -eq 0 ]; then
  fail "no plugins matched --only, nothing to build"
fi
