# Setup-Bluelink.ps1
# Post-installation script to configure Bluelink CLI and Deploy Engine
# This script generates an API key and configures both components to communicate locally.
# Service installation/management is handled by the MSI installer (WiX ServiceInstall).

param(
    [switch]$Force,         # Force regeneration of API key even if config exists
    [switch]$Uninstall      # Clean up configuration (service handled by MSI)
)

$ErrorActionPreference = "Stop"

# Paths
$BluelinkAppData = Join-Path $env:LOCALAPPDATA "NewStack\Bluelink"
$ConfigDir = Join-Path $BluelinkAppData "config"
$EngineDir = Join-Path $BluelinkAppData "engine"

$CliAuthConfigPath = Join-Path $ConfigDir "engine.auth.json"
$EngineConfigPath = Join-Path $EngineDir "config.json"

function New-ApiKey {
    # Generate a secure random API key (32 bytes = 64 hex chars)
    $bytes = New-Object byte[] 32
    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    $rng.GetBytes($bytes)
    return [System.BitConverter]::ToString($bytes).Replace("-", "").ToLower()
}

function Initialize-Directories {
    $dirs = @(
        $ConfigDir,
        (Join-Path $EngineDir "plugins"),
        (Join-Path $EngineDir "plugins\logs"),
        (Join-Path $EngineDir "state")
    )

    foreach ($dir in $dirs) {
        if (-not (Test-Path $dir)) {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
        }
    }
}

function Set-CliAuthConfig {
    param([string]$ApiKey)

    $config = @{
        method = "apiKey"
        apiKey = $ApiKey
    }

    $json = $config | ConvertTo-Json -Depth 10
    Set-Content -Path $CliAuthConfigPath -Value $json -Encoding UTF8
}

function Set-EngineConfig {
    param([string]$ApiKey)

    # Load existing config if present, otherwise create new
    if (Test-Path $EngineConfigPath) {
        $config = Get-Content $EngineConfigPath -Raw | ConvertFrom-Json -AsHashtable
    } else {
        $config = @{}
    }

    # Ensure auth section exists
    if (-not $config.ContainsKey("auth")) {
        $config["auth"] = @{}
    }

    # Set API key (replace any existing keys for clean setup)
    $config["auth"]["bluelink_api_keys"] = @($ApiKey)

    # Set loopback only for security (only accept local connections)
    $config["loopback_only"] = $true

    $json = $config | ConvertTo-Json -Depth 10
    Set-Content -Path $EngineConfigPath -Value $json -Encoding UTF8
}

function Main {
    if ($Uninstall) {
        # Nothing to clean up - config files are preserved intentionally
        # User data should not be deleted on uninstall
        exit 0
    }

    # Check if already configured (don't overwrite existing config)
    if ((Test-Path $CliAuthConfigPath) -and -not $Force) {
        exit 0
    }

    # Initialize directories
    Initialize-Directories

    # Generate API key
    $apiKey = New-ApiKey

    # Configure CLI
    Set-CliAuthConfig -ApiKey $apiKey

    # Configure Deploy Engine
    Set-EngineConfig -ApiKey $apiKey
}

Main
