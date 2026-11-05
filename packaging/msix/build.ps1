param(
  [Parameter(Mandatory = $true)][string]$GuiExe,
  [Parameter(Mandatory = $true)][string]$TrayExe,
  [Parameter(Mandatory = $true)][string]$Version,
  [Parameter(Mandatory = $true)][string]$Out
)

$ErrorActionPreference = 'Stop'
$here = Split-Path -Parent $MyInvocation.MyCommand.Path

$identity = Get-Content (Join-Path $here 'identity.json') -Raw | ConvertFrom-Json
foreach ($field in 'identity_name', 'publisher', 'publisher_display_name', 'display_name') {
  if ([string]::IsNullOrWhiteSpace($identity.$field)) {
    Write-Host "::notice::packaging/msix/identity.json has no $field yet; skipping the MSIX package."
    exit 0
  }
}

$core = ($Version -replace '^v', '') -replace '[-+].*$', ''
if ($core -notmatch '^\d+\.\d+\.\d+$') { $core = '0.0.0' }
$msixVersion = "$core.0"

$stage = Join-Path ([System.IO.Path]::GetTempPath()) ('msix-' + [guid]::NewGuid())
New-Item -ItemType Directory -Path $stage | Out-Null
Copy-Item $GuiExe (Join-Path $stage 'InternetMonitor.exe')
Copy-Item $TrayExe (Join-Path $stage 'InternetMonitorTray.exe')
Copy-Item (Join-Path $here 'Assets') (Join-Path $stage 'Assets') -Recurse

function Escape-Xml([string]$s) { [System.Security.SecurityElement]::Escape($s) }

$manifest = Get-Content (Join-Path $here 'AppxManifest.xml') -Raw
$manifest = $manifest.Replace('{{IDENTITY_NAME}}', (Escape-Xml $identity.identity_name))
$manifest = $manifest.Replace('{{PUBLISHER}}', (Escape-Xml $identity.publisher))
$manifest = $manifest.Replace('{{PUBLISHER_DISPLAY_NAME}}', (Escape-Xml $identity.publisher_display_name))
$manifest = $manifest.Replace('{{DISPLAY_NAME}}', (Escape-Xml $identity.display_name))
$manifest = $manifest.Replace('{{VERSION}}', $msixVersion)
Set-Content -Path (Join-Path $stage 'AppxManifest.xml') -Value $manifest -Encoding UTF8

$sdk = Get-ChildItem 'C:\Program Files (x86)\Windows Kits\10\bin\10.*' -Directory |
  Sort-Object { [version]$_.Name } -Descending |
  Where-Object { Test-Path (Join-Path $_.FullName 'x64\makeappx.exe') } |
  Select-Object -First 1
if (-not $sdk) { throw 'Windows SDK with makeappx.exe not found' }
$bin = Join-Path $sdk.FullName 'x64'

$priConfig = Join-Path ([System.IO.Path]::GetTempPath()) ('priconfig-' + [guid]::NewGuid() + '.xml')
& (Join-Path $bin 'makepri.exe') createconfig /cf $priConfig /dq en-US /o | Out-Host
if ($LASTEXITCODE -ne 0) { throw "makepri createconfig failed ($LASTEXITCODE)" }
& (Join-Path $bin 'makepri.exe') new /pr $stage /cf $priConfig /of (Join-Path $stage 'resources.pri') /o | Out-Host
if ($LASTEXITCODE -ne 0) { throw "makepri new failed ($LASTEXITCODE)" }

& (Join-Path $bin 'makeappx.exe') pack /d $stage /p $Out /o | Out-Host
if ($LASTEXITCODE -ne 0) { throw "makeappx pack failed ($LASTEXITCODE)" }

Write-Host "Built $Out (version $msixVersion)"
