# POTUS Installation Script for Windows
# Run with: iwr -useb https://potus.dev/install.ps1 | iex

$ErrorActionPreference = 'Stop'

# Configuration
$Version = if ($env:POTUS_VERSION) { $env:POTUS_VERSION } else { "latest" }
$InstallDir = if ($env:POTUS_INSTALL_DIR) { $env:POTUS_INSTALL_DIR } else { "$env:LOCALAPPDATA\potus\bin" }
$BinaryName = "potus.exe"

# Colors
function Write-Info($message) {
    Write-Host "ℹ $message" -ForegroundColor Blue
}

function Write-Success($message) {
    Write-Host "✓ $message" -ForegroundColor Green
}

function Write-Error-Custom($message) {
    Write-Host "✗ $message" -ForegroundColor Red
    exit 1
}

function Write-Warn($message) {
    Write-Host "⚠ $message" -ForegroundColor Yellow
}

# Detect architecture
function Get-Architecture {
    $arch = [System.Environment]::Is64BitOperatingSystem
    if ($arch) {
        return "x86_64"
    } else {
        Write-Error-Custom "32-bit Windows is not supported"
    }
}

# Get latest version
function Get-LatestVersion {
    if ($Version -eq "latest") {
        Write-Info "Fetching latest version..."
        try {
            $response = Invoke-RestMethod -Uri "https://api.github.com/repos/taaha3244/potus/releases/latest"
            $script:Version = $response.tag_name
            Write-Success "Latest version: $Version"
        } catch {
            Write-Error-Custom "Failed to fetch latest version: $_"
        }
    }
}

# Download and install
function Install-POTUS {
    $arch = Get-Architecture
    $filename = "potus_${Version}_Windows_${arch}.zip"
    $downloadUrl = "https://github.com/taaha3244/potus/releases/download/${Version}/${filename}"
    $tempDir = Join-Path $env:TEMP "potus-install-$(Get-Random)"
    $zipPath = Join-Path $tempDir $filename

    Write-Info "Downloading POTUS $Version..."

    try {
        New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
        Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -UseBasicParsing
    } catch {
        Write-Error-Custom "Failed to download POTUS: $_"
    }

    Write-Info "Extracting archive..."
    try {
        Expand-Archive -Path $zipPath -DestinationPath $tempDir -Force
    } catch {
        Write-Error-Custom "Failed to extract archive: $_"
    }

    Write-Info "Installing to $InstallDir..."
    try {
        if (!(Test-Path $InstallDir)) {
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }

        $sourcePath = Join-Path $tempDir $BinaryName
        $destPath = Join-Path $InstallDir $BinaryName

        Copy-Item -Path $sourcePath -Destination $destPath -Force
    } catch {
        Write-Error-Custom "Failed to install: $_"
    }

    # Cleanup
    Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue

    Write-Success "POTUS installed successfully!"
}

# Add to PATH
function Add-ToPath {
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")

    if ($currentPath -notlike "*$InstallDir*") {
        Write-Info "Adding $InstallDir to PATH..."
        [Environment]::SetEnvironmentVariable(
            "Path",
            "$currentPath;$InstallDir",
            "User"
        )
        $env:Path = "$env:Path;$InstallDir"
        Write-Success "Added to PATH"
        Write-Warn "Please restart your terminal for PATH changes to take effect"
    } else {
        Write-Info "$InstallDir is already in PATH"
    }
}

# Verify installation
function Test-Installation {
    Write-Info "Verifying installation..."

    $potusPath = Get-Command potus -ErrorAction SilentlyContinue
    if (!$potusPath) {
        Write-Warn "POTUS not found in PATH. Trying to add it..."
        $env:Path = "$env:Path;$InstallDir"
        $potusPath = Get-Command potus -ErrorAction SilentlyContinue
    }

    if ($potusPath) {
        $version = & potus --version 2>&1 | Select-Object -First 1
        Write-Success "Installed: $version"
    } else {
        Write-Error-Custom "POTUS binary not found. Please add $InstallDir to your PATH manually."
    }
}

# Show next steps
function Show-NextSteps {
    Write-Host ""
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Green
    Write-Host "  POTUS Installation Complete!" -ForegroundColor Green
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Green
    Write-Host ""
    Write-Host "Next steps:"
    Write-Host ""
    Write-Host "1. Set up your API key:" -ForegroundColor Cyan
    Write-Host '   $env:ANTHROPIC_API_KEY = "your-api-key"' -ForegroundColor Blue
    Write-Host '   $env:OPENAI_API_KEY = "your-api-key"' -ForegroundColor Blue
    Write-Host ""
    Write-Host "2. Run POTUS:" -ForegroundColor Cyan
    Write-Host "   potus" -ForegroundColor Blue
    Write-Host ""
    Write-Host "3. Get help:" -ForegroundColor Cyan
    Write-Host "   potus --help" -ForegroundColor Blue
    Write-Host ""
    Write-Host "4. Read the quick start guide:" -ForegroundColor Cyan
    Write-Host "   https://github.com/taaha3244/potus/blob/main/QUICKSTART.md" -ForegroundColor Blue
    Write-Host ""
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Green
}

# Main
function Main {
    Write-Host ""
    Write-Host "╔═══════════════════════════════════════════════════════════╗" -ForegroundColor Blue
    Write-Host "║                                                           ║" -ForegroundColor Blue
    Write-Host "║           POTUS - Power Of The Universal Shell            ║" -ForegroundColor Blue
    Write-Host "║              AI Coding Agent Installation                 ║" -ForegroundColor Blue
    Write-Host "║                                                           ║" -ForegroundColor Blue
    Write-Host "╚═══════════════════════════════════════════════════════════╝" -ForegroundColor Blue
    Write-Host ""

    Get-LatestVersion
    Install-POTUS
    Add-ToPath
    Test-Installation
    Show-NextSteps
}

Main
