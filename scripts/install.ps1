<#
.SYNOPSIS
    Installs the Windows/WSL2 fork of trellis-cli (qwatts-dev/trellis-cli).

.DESCRIPTION
    Downloads the trellis.exe Windows release, verifies its SHA256 checksum,
    installs it to a per-user folder, and adds that folder to your user PATH.
    No administrator rights required.

    This is the Windows analog of upstream's `curl -sL .../get | bash` script.
    Only trellis.exe is installed; the Linux binary is fetched into each WSL
    distro automatically during `trellis vm start`.

.PARAMETER Version
    Release tag to install (e.g. v1.19.0-wsl2.1). Defaults to the latest release.

.PARAMETER InstallDir
    Target install folder. Defaults to $env:LOCALAPPDATA\Programs\trellis.

.PARAMETER AddToPath
    Add InstallDir to the user PATH. Defaults to $true.

.EXAMPLE
    irm https://raw.githubusercontent.com/qwatts-dev/trellis-cli/master/scripts/install.ps1 | iex

.EXAMPLE
    .\install.ps1 -Version v1.19.0-wsl2.1 -InstallDir C:\trellis-wsl
#>
[CmdletBinding()]
param(
    [string]$Version = 'latest',
    [string]$InstallDir = (Join-Path $env:LOCALAPPDATA 'Programs\trellis'),
    [bool]$AddToPath = $true
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$Owner = 'qwatts-dev'
$Repo = 'trellis-cli'
$Asset = 'trellis_Windows_x86_64.zip'
$Checksums = 'trellis_checksums.txt'

function Write-Info { param([string]$Message) Write-Host "==> $Message" -ForegroundColor Cyan }
function Write-Ok { param([string]$Message) Write-Host "[ok] $Message" -ForegroundColor Green }

# Resolve the release tag (follow the /latest redirect when no version is given).
if ($Version -eq 'latest' -or [string]::IsNullOrWhiteSpace($Version)) {
    Write-Info 'Resolving latest release...'
    $api = "https://api.github.com/repos/$Owner/$Repo/releases/latest"
    try {
        $release = Invoke-RestMethod -Uri $api -Headers @{ 'User-Agent' = 'trellis-install' }
        $Version = $release.tag_name
    } catch {
        throw "Could not resolve the latest release from $api : $($_.Exception.Message)"
    }
}
Write-Info "Installing trellis-cli $Version"

$base = "https://github.com/$Owner/$Repo/releases/download/$Version"
$assetUrl = "$base/$Asset"
$checksumUrl = "$base/$Checksums"

$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("trellis-install-" + [System.Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tmp -Force | Out-Null

try {
    $zipPath = Join-Path $tmp $Asset
    $sumPath = Join-Path $tmp $Checksums

    Write-Info "Downloading $Asset..."
    Invoke-WebRequest -Uri $assetUrl -OutFile $zipPath -Headers @{ 'User-Agent' = 'trellis-install' }

    Write-Info 'Verifying checksum...'
    Invoke-WebRequest -Uri $checksumUrl -OutFile $sumPath -Headers @{ 'User-Agent' = 'trellis-install' }
    $expected = (Get-Content $sumPath |
        Where-Object { $_ -match [regex]::Escape($Asset) } |
        ForEach-Object { ($_ -split '\s+')[0] } |
        Select-Object -First 1)
    if (-not $expected) { throw "Checksum for $Asset not found in $Checksums" }
    $actual = (Get-FileHash -Algorithm SHA256 -Path $zipPath).Hash
    if ($actual.ToLower() -ne $expected.Trim().ToLower()) {
        throw "Checksum mismatch for $Asset`n  expected: $($expected.Trim())`n  actual:   $actual"
    }
    Write-Ok 'Checksum verified'

    Write-Info "Installing to $InstallDir"
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Expand-Archive -Path $zipPath -DestinationPath $tmp -Force
    $exe = Get-ChildItem -Path $tmp -Filter 'trellis.exe' -Recurse | Select-Object -First 1
    if (-not $exe) { throw 'trellis.exe not found in the downloaded archive' }
    Copy-Item -Path $exe.FullName -Destination (Join-Path $InstallDir 'trellis.exe') -Force
    Write-Ok "Installed $(Join-Path $InstallDir 'trellis.exe')"

    if ($AddToPath) {
        $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
        $entries = ($userPath -split ';') | Where-Object { $_ -and $_.TrimEnd('\') -ieq $InstallDir.TrimEnd('\') }
        if (-not $entries) {
            $newPath = if ([string]::IsNullOrEmpty($userPath)) { $InstallDir } else { "$userPath;$InstallDir" }
            [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
            $env:Path = "$env:Path;$InstallDir"
            Write-Ok "Added $InstallDir to your user PATH (restart terminals to pick it up)"
        } else {
            Write-Ok "$InstallDir already on your user PATH"
        }
    }

    Write-Host ''
    Write-Ok "trellis-cli $Version installed. Run 'trellis --version' in a new terminal to confirm."
} finally {
    Remove-Item -Path $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
