# Cluckers Windows Installer
# Download and install the cluckers binary from GitHub Releases.
#
# Usage:
#   (iwr https://raw.githubusercontent.com/0xc0re/cluckers/master/install.ps1).Content | iex
#   powershell -ExecutionPolicy Bypass -File install.ps1

$ErrorActionPreference = "Stop"

# --------------------------------------------------------------------------- #
#  Color / UX helpers
# --------------------------------------------------------------------------- #

function Write-Info    { param([string]$Msg) Write-Host ":: $Msg" -ForegroundColor Blue }
function Write-Success { param([string]$Msg) Write-Host "   $Msg" -ForegroundColor Green }
function Write-Warn    { param([string]$Msg) Write-Host "!! $Msg" -ForegroundColor Yellow }
function Write-Err     { param([string]$Msg) Write-Host "   $Msg" -ForegroundColor Red }
function Write-Step    { param([string]$Msg) Write-Host "=> $Msg" -ForegroundColor White }

# --------------------------------------------------------------------------- #
#  Safety checks
# --------------------------------------------------------------------------- #

$currentPrincipal = New-Object Security.Principal.WindowsPrincipal(
    [Security.Principal.WindowsIdentity]::GetCurrent()
)
if ($currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Warn "Running as Administrator is not required. Consider running as a normal user."
}

# --------------------------------------------------------------------------- #
#  Platform detection
# --------------------------------------------------------------------------- #

if ($env:OS -ne "Windows_NT") {
    Write-Err "This installer is for Windows only."
    Write-Host "  For Linux, use install.sh instead."
    exit 1
}

$arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLower()
if ($arch -ne "x64") {
    Write-Err "Cluckers only supports x86_64 (amd64). Detected architecture: $arch"
    exit 1
}

# --------------------------------------------------------------------------- #
#  Install location
# --------------------------------------------------------------------------- #

$InstallDir = "$env:LOCALAPPDATA\cluckers\bin"

if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$InstallPath = Join-Path $InstallDir "cluckers.exe"

# --------------------------------------------------------------------------- #
#  Discover latest release
# --------------------------------------------------------------------------- #

$GitHubAPI = "https://api.github.com/repos/0xc0re/cluckers/releases/latest"

Write-Step "Checking latest cluckers release..."

try {
    $release = Invoke-RestMethod -Uri $GitHubAPI -Headers @{
        "User-Agent" = "cluckers-installer"
        "Accept"     = "application/vnd.github+json"
    }
} catch {
    Write-Err "Failed to fetch release information from GitHub."
    Write-Host "  URL: $GitHubAPI"
    Write-Host "  Error: $_"
    exit 1
}

$LatestTag = $release.tag_name
$LatestVersion = $LatestTag -replace "^v", ""

if ([string]::IsNullOrEmpty($LatestVersion)) {
    Write-Err "Could not determine latest release version."
    exit 1
}

# Find the windows_amd64 zip asset.
$ZipAsset = $release.assets | Where-Object { $_.name -match "^cluckers_.*windows_amd64\.zip$" } | Select-Object -First 1
$ChecksumsAsset = $release.assets | Where-Object { $_.name -eq "checksums.txt" } | Select-Object -First 1

if (-not $ZipAsset) {
    Write-Err "Could not find windows_amd64 release asset."
    Write-Host "  Check: https://github.com/0xc0re/cluckers/releases/latest"
    exit 1
}

Write-Info "Latest version: $LatestVersion"

# --------------------------------------------------------------------------- #
#  Idempotency -- check existing install
# --------------------------------------------------------------------------- #

if (Test-Path $InstallPath) {
    try {
        $currentOutput = & $InstallPath --version 2>&1 | Select-Object -First 1
        $currentVersion = ($currentOutput -replace ".*version ", "" -replace " .*", "").Trim()
        if ($currentVersion -eq $LatestVersion) {
            Write-Success "cluckers $LatestVersion is already installed and up to date."
            Write-Host "  Location: $InstallPath"
            exit 0
        }
        if (-not [string]::IsNullOrEmpty($currentVersion)) {
            Write-Info "Updating cluckers from $currentVersion to $LatestVersion"
        }
    } catch {
        # Could not determine current version; proceed with install.
    }
}

# --------------------------------------------------------------------------- #
#  Download and verify
# --------------------------------------------------------------------------- #

$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) "cluckers-install-$([System.Guid]::NewGuid().ToString('N').Substring(0,8))"
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

try {
    $ZipPath = Join-Path $TempDir "cluckers.zip"

    Write-Step "Downloading cluckers $LatestVersion..."
    Invoke-WebRequest -Uri $ZipAsset.browser_download_url -OutFile $ZipPath -UseBasicParsing

    # Verify checksum if checksums.txt is available.
    if ($ChecksumsAsset) {
        Write-Step "Verifying checksum..."
        try {
            $ChecksumsPath = Join-Path $TempDir "checksums.txt"
            Invoke-WebRequest -Uri $ChecksumsAsset.browser_download_url -OutFile $ChecksumsPath -UseBasicParsing

            $checksumLines = Get-Content $ChecksumsPath
            $expectedLine = $checksumLines | Where-Object { $_ -match "^[a-f0-9]+\s+cluckers_.*windows_amd64\.zip" } | Select-Object -First 1
            if ($expectedLine) {
                $expectedHash = ($expectedLine -split "\s+")[0]
                $actualHash = (Get-FileHash -Path $ZipPath -Algorithm SHA256).Hash.ToLower()
                if ($expectedHash -ne $actualHash) {
                    Write-Err "Checksum verification failed!"
                    Write-Host "  Expected: $expectedHash"
                    Write-Host "  Got:      $actualHash"
                    exit 1
                }
                Write-Success "Checksum verified."
            } else {
                Write-Warn "No matching checksum found for windows zip; skipping verification."
            }
        } catch {
            Write-Warn "Could not verify checksum: $_"
        }
    } else {
        Write-Warn "No checksums.txt found in release; skipping verification."
    }

    # --------------------------------------------------------------------------- #
    #  Extract and install
    # --------------------------------------------------------------------------- #

    Write-Step "Installing to $InstallPath..."

    $ExtractDir = Join-Path $TempDir "extracted"
    Expand-Archive -Path $ZipPath -DestinationPath $ExtractDir -Force

    $Binary = Get-ChildItem -Path $ExtractDir -Recurse -Filter "cluckers.exe" | Select-Object -First 1

    if (-not $Binary) {
        Write-Err "Binary not found in archive."
        Get-ChildItem -Path $ExtractDir -Recurse | ForEach-Object { Write-Host "  $_" }
        exit 1
    }

    Copy-Item -Path $Binary.FullName -Destination $InstallPath -Force

} finally {
    # Clean up temp directory.
    if (Test-Path $TempDir) {
        Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# --------------------------------------------------------------------------- #
#  PATH check
# --------------------------------------------------------------------------- #

$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
$PathAdded = $false

if ($UserPath -split ";" | ForEach-Object { $_.TrimEnd("\") } | Where-Object { $_ -eq $InstallDir.TrimEnd("\") }) {
    # Already in PATH.
} else {
    # Add to user PATH.
    $NewPath = if ([string]::IsNullOrEmpty($UserPath)) { $InstallDir } else { "$UserPath;$InstallDir" }
    [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
    $PathAdded = $true
}

# --------------------------------------------------------------------------- #
#  Summary
# --------------------------------------------------------------------------- #

Write-Host ""
Write-Host "================================================" -ForegroundColor White
Write-Host "  Cluckers installed successfully" -ForegroundColor White
Write-Host "================================================" -ForegroundColor White
Write-Host ""
Write-Host "  Location:  $InstallPath"
Write-Host "  Version:   $LatestVersion"
Write-Host ""

if ($PathAdded) {
    Write-Success "Added $InstallDir to your PATH."
    Write-Host "  Open a new terminal (or restart PowerShell) for the PATH change to take effect."
    Write-Host ""
}

Write-Host "  Next steps:"
Write-Host "    cluckers            Launch GUI"
Write-Host "    cluckers launch     Launch game (CLI)"
Write-Host "    cluckers status     Check system readiness"
Write-Host ""
Write-Host "  On first launch, cluckers will prompt for your Project Crown"
Write-Host "  credentials."
Write-Host ""
