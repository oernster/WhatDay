# Testing

How WhatDay is tested, what the gate checks and what is checked by hand.
Every command is PowerShell, run from the repository root.

## The gate

```powershell
./test.ps1
```

`build.ps1` runs it before building anything and cannot skip it. In order:

1. `gofmt -l .` must list nothing.
2. `go vet ./...` must pass.
3. staticcheck, at the version pinned in `test.ps1`, must report nothing.
4. `go test -count=1 ./...`: the whole suite, never from cache.
5. Coverage of `internal/domain` and `internal/application` together must be
   100%. Any function short of it is named.
6. Each infrastructure package and `internal/ui` must reach its own floor
   (below).

It ends with `All green.` Trust the exit code, not the text:

```powershell
./test.ps1; $LASTEXITCODE
```

`0` means every step passed. A failing step stops the script with the reason.

`-Floor` changes the core floor for a deliberate check, never for a release:

```powershell
./test.ps1 -Floor 95
```

### The coverage floors

The core is held at 100% because it is pure: no disk, no clock, no window,
so there is nothing that cannot be reached. The rest is held at the number
it actually reaches. A floor is a measurement, never an aspiration: it fails
the moment cover is lost; it is raised when cover rises.

| Package | Floor | Why not 100 |
|---|---|---|
| `internal/domain`, `internal/application` | 100 | |
| `infrastructure/clock` | 100 | |
| `infrastructure/instance` | 100 | |
| `infrastructure/settings` | 100 | |
| `infrastructure/zone` | 92 | Three failures cannot be caused on demand: Windows refusing the zone query, `icu.dll` missing, ICU reporting an error. |
| `infrastructure/runlog` | 74 | Its crash paths run in child processes the tests start on purpose. The tests prove them by reading the child's log; coverage cannot see into another process. Two more failures cannot be caused on demand. |
| `infrastructure/setup` | 54 | The route, the extraction, the step log and the process lookup are tested. The registry, shortcut and process-ending work change the real machine, so an install on the reference machine is their test. |
| `internal/ui` | 31 | The menu mapping, the drawing, the support entry, the icon and the monitor reading are tested. The message loop and window handling need a desktop and a person at it. |

## What the tests prove

### The day is right

- **Every date for four centuries.** `TestEveryDayAgainstIndependentFormula`
  walks all 146,097 dates from 1 January 2000 to 31 December 2399 and
  checks each weekday against Sakamoto's method, written into the test
  with its own leap-year rule. It does not trust Go's `time` package to check
  Go's `time` package.
- **Every midnight in London for four centuries.** `TestMidnightChain` follows
  `NextMidnight` from 2000 to 2399 and asserts each step lands on the next
  civil date at 00:00, through every leap day and every clock change.
- **Every midnight in every zone for a century.** `TestEveryZoneMidnightChain`
  does the same in every zone of the embedded tz database from 2000 to 2100.
  The only dates it allows to be skipped are the one Samoa and Tokelau
  skipped: 30 December 2011.
- **Midnights that never happen.** `TestMidnightInAGapStartsAtTheJump`: where
  clocks jump from 00:00 to 01:00, the new day starts at the jump.
- **Travel.** `TestTravelFollowsNewZone`: the same instant reads Saturday in
  London and New York but Sunday in Sydney; each waits for its own
  midnight.

### The colours are readable

`palette_test.go` holds every colour to two limits:

- WCAG 2 contrast of at least 3.0:1 (large text) against the measured taskbar
  mean `#1C222F` and a provisional lighter worst case `#3A3A3A`.
- A CIEDE2000 difference of at least 20 between every pair, so no two choices
  look alike.

The colour maths is itself tested: `TestDeltaE2000MatchesReference` checks the
CIEDE2000 formula against Sharma, Wu and Dalal's published pairs;
`TestContrastMatchesReference` checks the contrast formula against known
values.

### The structure holds

`tests/structural/boundary_test.go` parses every Go file in the repository
and fails on a layer violation, an I/O import or a clock read in the core, a
second composition root, a `net` import, test support used outside tests, a
file over 400 lines or in the 381 to 399 band, an undocumented exported
type. Each assertion was proved by planting a violation.
[ARCHITECTURE.md](ARCHITECTURE.md) lists them against the invariants they
guard.

### The rest

- **Settings**: a real temporary folder. Round trips, the file's exact shape,
  a missing file, a corrupt file, an unreadable file, an unwritable folder and
  an interrupted save that must leave the old file whole.
- **Single instance**: a real named mutex, held by a second process.
- **The log**: a child process that really panics, with and without a
  standard error, whose log is then read.
- **The zone**: fakes for every failure, plus the real Windows zone and the
  real ICU on the machine running the tests.
- **Setup**: every route decision, the extraction fence against a crafted
  archive, the step log and the process lookup.
- **The support entry**: the address asserted literally and as `https`; the
  entry asks for that one address; a refusal is logged and shown; anything but
  `https` is refused before the desktop sees it.

## What the tests never do

- **Touch a running WhatDay.** The owner's copy is usually running. The
  process tests use the test binary's own name and a name that never runs,
  so the suite neither ends WhatDay nor measures differently because of it.
- **Open a browser.** The window's opener is replaced in the tests; the real
  one is only handed addresses it refuses.
- **Read the real settings or log.** Tests work in temporary folders.

## Running part of the suite

```powershell
go test ./internal/domain/...
go test -run TestMidnightChain -v ./internal/domain
go test -cover ./internal/infrastructure/zone
```

The domain suite takes several seconds, most of it the every-zone midnight
walk.

## Checked by hand

Some requirements need a real desktop, a real clock or a real person. They
are checked on the reference machine (a Windows 11 desktop with monitors at
100% and 250% scale) and recorded in [REQUIREMENTS.md](REQUIREMENTS.md)
beside each requirement.

| Check | How |
|---|---|
| The day changes at a real midnight (FR-003) | Leave it running; the log shows `showing <day>` at 00:00. |
| Resume from sleep (FR-004) | Sleep before midnight, wake after; the day is right within a second. |
| No frame, always on top, no taskbar button or Alt+Tab entry (FR-010 to FR-013) | Look. |
| Height on the 250% monitor (FR-014) | Drag it there; it matches that taskbar. |
| Hides for fullscreen, not for a screenshot (FR-020) | A fullscreen video hides it; Win+Shift+S does not. |
| Display, DPI and taskbar changes (FR-023) | Change resolution and scale, unplug a monitor. |
| Explorer restart (FR-024) | Restart Windows Explorer from Task Manager; the tray icon returns. |
| Quit (FR-035) | The log says `quit from the tray` and the process ends. |
| Support (FR-036) | The PayPal page opens in the browser. |
| Start at sign-in (FR-060) | Install, reboot, sign in. |
| Setup (FR-070 to FR-074) | Install, update, go back, repair and uninstall, each once, with WhatDay running; the registry and folders inspected afterwards. |
| Idle CPU, memory, startup time (NFR-PERF-001 to 003) | `Get-Process` over ten minutes and 24 hours; the log's start and shown lines. |
