#!/bin/bash

set -e

POSITIONAL=()
while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    -h|--help)
    HELP=yes
    shift # past argument
    ;;
    --version)
    PLUGIN_VERSION="$2"
    shift # past argument
    shift # past value
    ;;
    *)    # unknown option
    POSITIONAL+=("$1") # save it in an array for later
    shift # past argument
    ;;
esac
done
set -- "${POSITIONAL[@]}" # restore positional parameters

function help {
  cat << EOF
Test plugin builder for the deploy engine

Builds the plugins under testplugins/ and installs them into the local plugin
directory tree used by the deploy engine when running locally:

  .bluelink/deploy-engine/plugins/bin/providers/newstack-cloud/example/<version>/plugin

These plugins are for exercising the deploy engine locally, they do not talk to
any upstream system.

To build and install:
bash scripts/build-test-plugins.sh

To override the version used for the install directory and the version compiled
into the plugin binary:
bash scripts/build-test-plugins.sh --version 0.2.0
EOF
}

if [ -n "$HELP" ]; then
  help
  exit 0
fi

PLUGIN_VERSION="${PLUGIN_VERSION:-0.1.0}"

DEPLOY_ENGINE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLUGIN_BIN_DIR="$DEPLOY_ENGINE_DIR/.bluelink/deploy-engine/plugins/bin"

INSTALL_DIR="$PLUGIN_BIN_DIR/providers/newstack-cloud/example/$PLUGIN_VERSION"

echo "Building example provider $PLUGIN_VERSION ..."
mkdir -p "$INSTALL_DIR"
(
  cd "$DEPLOY_ENGINE_DIR"
  go build \
    -ldflags "-X main.version=$PLUGIN_VERSION" \
    -o "$INSTALL_DIR/plugin" \
    ./testplugins/exampleprovider
)
echo "Installed example provider plugin at $INSTALL_DIR/plugin"
