# Builds WhatDay: runs the test gate, embeds the icon and version, then
# compiles the application into dist\WhatDay.exe.
#
#   ./build.ps1
#
# The gate runs first and cannot be skipped: a gate that can be skipped is
# skipped on the day it would have caught something.
#
# The version comes from VERSION alone. It reaches the program through
# -ldflags (appVersion is a var, since -X cannot reach a const) and the
# executable's properties through a versioninfo file generated here, never
# committed.

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root

& (Join-Path $root 'test.ps1')

$version = (Get-Content (Join-Path $root 'VERSION') -Raw).Trim()
if ($version -notmatch '^(\d+)\.(\d+)\.(\d+)$') { throw "VERSION holds '$version', not major.minor.patch" }
$major, $minor, $patch = [int]$Matches[1], [int]$Matches[2], [int]$Matches[3]

$icon = Join-Path $root 'assets/application-icon.ico'
if (-not (Test-Path $icon)) { throw 'Missing assets/application-icon.ico: run python tools/genicons.py first.' }

# goversioninfo is pinned so a new release cannot change a build of unchanged code.
$goversioninfo = 'github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.7.0'
$resource = Join-Path $root 'cmd/whatday/resource.syso'
$info = Join-Path ([System.IO.Path]::GetTempPath()) 'whatday-versioninfo.json'

$number = [ordered]@{ Major = $major; Minor = $minor; Patch = $patch; Build = 0 }
$dotted = "$version.0"
$versionInfo = [ordered]@{
    FixedFileInfo  = [ordered]@{ FileVersion = $number; ProductVersion = $number; FileFlagsMask = '3f'; FileFlags = '00'; FileOS = '040004'; FileType = '01'; FileSubType = '00' }
    StringFileInfo = [ordered]@{
        CompanyName      = 'Oliver Ernster'
        FileDescription  = 'WhatDay'
        FileVersion      = $dotted
        InternalName     = 'WhatDay'
        LegalCopyright   = [char]0x00A9 + ' 2026 Oliver Ernster'
        OriginalFilename = 'WhatDay.exe'
        ProductName      = 'WhatDay'
        ProductVersion   = $dotted
    }
    VarFileInfo    = [ordered]@{ Translation = [ordered]@{ LangID = '0409'; CharsetID = '04B0' } }
    IconPath       = $icon
}

Write-Host "Embedding the icon and version $version..."
try {
    $versionInfo | ConvertTo-Json -Depth 5 | Set-Content -Path $info -Encoding utf8
    go run $goversioninfo -64 -o $resource $info
    if ($LASTEXITCODE -ne 0) { throw "goversioninfo failed with exit code $LASTEXITCODE" }
} finally {
    if (Test-Path $info) { Remove-Item $info -Force }
}

Write-Host 'Building dist\WhatDay.exe...'
$dist = Join-Path $root 'dist'
New-Item -ItemType Directory -Force -Path $dist | Out-Null
go build -trimpath -ldflags "-H windowsgui -s -w -X main.appVersion=$version" -o (Join-Path $dist 'WhatDay.exe') ./cmd/whatday
if ($LASTEXITCODE -ne 0) { throw "go build failed with exit code $LASTEXITCODE" }

Write-Host "Built dist\WhatDay.exe, version $version."
