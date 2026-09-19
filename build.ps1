# Builds WhatDay's setup program, as PigeonPost's build does: the application,
# zipped as the payload, inside a Wails setup program.
#
#   ./build.ps1                 build the setup program
#   ./build.ps1 -SkipInstaller  build only the application
#
# Outputs:
#   build/bin/WhatDay.exe               the application (the payload)
#   dist-installer/WhatDaySetup.exe     the setup program; this is what ships
#
# The gate runs first and cannot be skipped: a gate that can be skipped is
# skipped on the day it would have caught something.
#
# The version comes from VERSION alone. It reaches each program through
# -ldflags (appVersion is a var, since -X cannot reach a const) and each
# executable's properties through files generated here, never committed.
param(
    [switch]$SkipInstaller
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root

& (Join-Path $root 'test.ps1')

$version = (Get-Content (Join-Path $root 'VERSION') -Raw).Trim()
if ($version -notmatch '^(\d+)\.(\d+)\.(\d+)$') { throw "VERSION holds '$version', not major.minor.patch" }
$major, $minor, $patch = [int]$Matches[1], [int]$Matches[2], [int]$Matches[3]
Write-Host "Building WhatDay $version"

$icon = Join-Path $root 'assets/application-icon.ico'
if (-not (Test-Path $icon)) { throw 'Missing assets/application-icon.ico: run python tools/genicons.py first.' }

$company = 'Oliver Ernster'
$copyright = [char]0x00A9 + ' 2026 Oliver Ernster'
$dotted = "$version.0"

# goversioninfo is pinned so a new release cannot change a build of unchanged code.
$goversioninfo = 'github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.7.0'
$resource = Join-Path $root 'cmd/whatday/resource.syso'
$info = Join-Path ([System.IO.Path]::GetTempPath()) 'whatday-versioninfo.json'
$number = [ordered]@{ Major = $major; Minor = $minor; Patch = $patch; Build = 0 }
$versionInfo = [ordered]@{
    FixedFileInfo  = [ordered]@{ FileVersion = $number; ProductVersion = $number; FileFlagsMask = '3f'; FileFlags = '00'; FileOS = '040004'; FileType = '01'; FileSubType = '00' }
    StringFileInfo = [ordered]@{
        CompanyName = $company; FileDescription = 'WhatDay'; FileVersion = $dotted; InternalName = 'WhatDay'
        LegalCopyright = $copyright; OriginalFilename = 'WhatDay.exe'; ProductName = 'WhatDay'; ProductVersion = $dotted
    }
    VarFileInfo    = [ordered]@{ Translation = [ordered]@{ LangID = '0409'; CharsetID = '04B0' } }
    IconPath       = $icon
}

Write-Host 'Embedding the icon and version in the application...'
try {
    $versionInfo | ConvertTo-Json -Depth 5 | Set-Content -Path $info -Encoding utf8
    go run $goversioninfo -64 -o $resource $info
    if ($LASTEXITCODE -ne 0) { throw "goversioninfo failed with exit code $LASTEXITCODE" }
} finally {
    if (Test-Path $info) { Remove-Item $info -Force }
}

Write-Host 'Building the application...'
$bin = Join-Path $root 'build/bin'
New-Item -ItemType Directory -Force -Path $bin | Out-Null
$appExe = Join-Path $bin 'WhatDay.exe'
go build -trimpath -ldflags "-H windowsgui -s -w -X main.appVersion=$version" -o $appExe ./cmd/whatday
if ($LASTEXITCODE -ne 0) { throw "go build failed with exit code $LASTEXITCODE" }

if ($SkipInstaller) {
    Write-Host "Done: build/bin/WhatDay.exe ($version)"
    exit 0
}

$payload = Join-Path $root 'installer/payload.zip'
try {
    Write-Host 'Packaging the application as the setup payload...'
    if (Test-Path $payload) { Remove-Item $payload }
    Compress-Archive -Path $appExe -DestinationPath $payload

    # Wails takes the setup program's icon and properties from build/windows,
    # generating defaults only where a file is missing. Its default properties
    # claim version 1.0.0, so both are written here from the one source.
    $windowsBuild = Join-Path $root 'installer/build/windows'
    New-Item -ItemType Directory -Force -Path $windowsBuild | Out-Null
    Copy-Item $icon (Join-Path $windowsBuild 'icon.ico') -Force
    $setupInfo = [ordered]@{
        fixed = [ordered]@{ file_version = $version }
        info  = [ordered]@{ '0000' = [ordered]@{
                ProductVersion = $version; CompanyName = $company; FileDescription = 'WhatDay Setup'
                LegalCopyright = $copyright; ProductName = 'WhatDay Setup'; Comments = 'Installs WhatDay for your account only.'
            } }
    }
    $setupInfo | ConvertTo-Json -Depth 5 | Set-Content -Path (Join-Path $windowsBuild 'info.json') -Encoding utf8

    Write-Host 'Building the setup program (wails)...'
    Push-Location (Join-Path $root 'installer')
    try {
        wails build -trimpath -ldflags "-X main.appVersion=$version"
        if ($LASTEXITCODE -ne 0) { throw "wails build failed with exit code $LASTEXITCODE" }
    } finally {
        Pop-Location
    }

    Write-Host 'Collecting the setup program...'
    $distDir = Join-Path $root 'dist-installer'
    New-Item -ItemType Directory -Force -Path $distDir | Out-Null
    Copy-Item (Join-Path $root 'installer/build/bin/WhatDaySetup.exe') (Join-Path $distDir 'WhatDaySetup.exe') -Force
} finally {
    # Restore the empty-zip placeholder so `go build ./...` works without a full build.
    $placeholder = New-Object byte[] 22
    $placeholder[0] = 0x50; $placeholder[1] = 0x4B; $placeholder[2] = 0x05; $placeholder[3] = 0x06
    [System.IO.File]::WriteAllBytes($payload, $placeholder)
}

Write-Host "Done: dist-installer/WhatDaySetup.exe ($version)"
