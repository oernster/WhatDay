# Verifies WhatDay: formatting, vet, staticcheck, the whole suite and the
# coverage floor. Ported from ED Voyage Companion's test.ps1.
#
#   ./test.ps1              run everything
#   ./test.ps1 -Floor 95    run with a different coverage floor, for a deliberate check
#
# build.ps1 runs this before it builds, so a release cannot be cut from a tree
# that fails it. Run it directly while working.
#
# The floor is the measured number, not a target. Domain and application are at
# 100% (NFR-MAINT-001). A floor at the measured number fails the moment cover is
# lost, which is the only moment it is worth being told.
param(
    [double]$Floor = 100
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root

# The gate covers the layers that can be tested without a machine: pure decision
# logic with no filesystem, clock or window. Infrastructure and UI need Win32 and
# sit outside the floor rather than dragging it to a number that means nothing.
$gated = './internal/domain/...', './internal/application/...'

# staticcheck is pinned so a new release cannot fail a build over code that has
# not changed.
$staticcheck = 'honnef.co/go/tools/cmd/staticcheck@v0.8.1'

Write-Host 'Checking formatting...'
$unformatted = gofmt -l .
if ($unformatted) { throw "gofmt reports unformatted files:`n$($unformatted -join "`n")" }

Write-Host 'Vetting...'
go vet ./...
if ($LASTEXITCODE -ne 0) { throw "go vet failed with exit code $LASTEXITCODE" }

Write-Host 'Running staticcheck...'
go run $staticcheck ./...
if ($LASTEXITCODE -ne 0) { throw "staticcheck failed with exit code $LASTEXITCODE" }

Write-Host 'Running the whole suite...'
go test -count=1 ./...
if ($LASTEXITCODE -ne 0) { throw "go test failed with exit code $LASTEXITCODE" }

Write-Host "Measuring coverage of $($gated -join ', ')..."
$profilePath = Join-Path ([System.IO.Path]::GetTempPath()) 'whatday-coverage.out'
try {
    # Each gated package is measured against its own statements.
    go test -count=1 "-coverprofile=$profilePath" @gated
    if ($LASTEXITCODE -ne 0) { throw "the coverage run failed with exit code $LASTEXITCODE" }

    $summary = go tool cover "-func=$profilePath"
    if ($LASTEXITCODE -ne 0) { throw "go tool cover failed with exit code $LASTEXITCODE" }

    # The last line of the report is the total. Read the exit code and this
    # line, never the run's own output.
    $total = ($summary | Select-Object -Last 1)
    if ($total -notmatch '([0-9]+(?:\.[0-9]+)?)%\s*$') { throw "could not read a total from: $total" }
    $percent = [double]$Matches[1]

    $below = $summary | Where-Object { $_ -notmatch '100\.0%\s*$' -and $_ -notmatch '^total:' }
    if ($percent -lt $Floor) {
        Write-Host 'Not covered:'
        $below | ForEach-Object { Write-Host "  $_" }
        throw "coverage is $percent%, below the floor of $Floor%"
    }
    Write-Host "Coverage $percent%, floor $Floor%."
} finally {
    if (Test-Path $profilePath) { Remove-Item $profilePath -Force }
}

# Infrastructure, each package held at the number it actually reaches. These
# floors are measurements, never aspirations; raise one when its cover rises.
#
# runlog sits below 100 because its crash paths run inside child processes the
# tests start on purpose. The tests prove them by reading the child's log;
# coverage cannot see into another process. The rest of the shortfall is two
# failures that cannot be caused on demand: a write failing straight after a
# successful open; Go refusing a crash-output file.
#
# setup sits at its route, extraction and step log. Its registry, shortcut and
# process work change the real machine, so the install on the reference
# machine is its test rather than every run of the suite.
#
# zone sits below 100 by three failures that cannot be caused on demand:
# Windows refusing the zone query, icu.dll missing, ICU reporting an error.
#
# ui is the Win32 surface. Its menu mapping, drawing, icon and monitor reading
# are tested; the message loop and window handling need a desktop and a person
# at it, so they are verified by hand against REQUIREMENTS.md.
$measured = [ordered]@{
    './internal/infrastructure/clock'    = 100
    './internal/infrastructure/instance' = 100
    './internal/infrastructure/runlog'   = 74
    './internal/infrastructure/settings' = 100
    './internal/infrastructure/setup'    = 54
    './internal/infrastructure/zone'     = 92
    './internal/ui'                      = 31
}

Write-Host 'Measuring infrastructure...'
foreach ($package in $measured.Keys) {
    $floor = $measured[$package]
    $reported = go test -count=1 -cover $package
    if ($LASTEXITCODE -ne 0) { throw "$package failed with exit code $LASTEXITCODE" }

    $line = $reported | Where-Object { $_ -match 'coverage: ' } | Select-Object -First 1
    if ($line -notmatch 'coverage: ([0-9]+(?:\.[0-9]+)?)%') {
        throw "could not read a coverage figure for ${package}: $line"
    }
    $reached = [double]$Matches[1]
    if ($reached -lt $floor) {
        throw "$package is at $reached%, below its floor of $floor%"
    }
    Write-Host ("  {0,-38} {1,5}%  floor {2}%" -f $package, $reached, $floor)
}

Write-Host 'All green.'
