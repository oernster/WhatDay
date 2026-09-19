# Architecture

WhatDay answers one question, so the interesting parts are the ones that make
the answer right every time: the day boundary in any timezone, the strip's
place on a desk of mixed-scale monitors and a program that never dies in
silence. This document says how the code is divided, which rules the tests
hold it to and why each design choice was made.

[REQUIREMENTS.md](REQUIREMENTS.md) is the specification; requirement numbers
below (FR-003 and so on) refer to it.

## Invariants

Each is enforced by a test, not by convention. Every structural assertion was
proved to bite by planting a violation and watching it fail.

| Invariant | Enforced by |
|---|---|
| The domain imports nothing of WhatDay's own outside the domain. | [`TestDomainDependsOnNothingOfOurs`](tests/structural/boundary_test.go) |
| The application imports only the domain and itself. | [`TestApplicationDependsOnDomainOnly`](tests/structural/boundary_test.go) |
| Domain and application perform no I/O: no `os`, `os/exec`, `path/filepath`, `syscall`, `unsafe`, `golang.org/x/sys/windows` or random source. | [`TestCoreIsPure`](tests/structural/boundary_test.go) |
| Domain and application never read the clock: no `time.Now`, `time.Since` or `time.Until`. The instant arrives through the `Clock` port. | [`TestCoreIsPure`](tests/structural/boundary_test.go) |
| UI and infrastructure never import each other. | [`TestUIAndInfrastructureStayApart`](tests/structural/boundary_test.go) |
| Only `cmd/whatday/main.go` wires the application to infrastructure. | [`TestCompositionRootIsWhitelisted`](tests/structural/boundary_test.go) |
| No code in the repository imports `net` or any `net/` package. | [`TestNoNetworkImports`](tests/structural/boundary_test.go) |
| Test support is imported by tests only. | [`TestTestSupportIsForTestsOnly`](tests/structural/boundary_test.go) |
| No Go file exceeds 400 lines; none sits in the 381 to 399 danger band. | [`TestNoFileExceedsLineLimit`, `TestNoFileInDangerBand`](tests/structural/boundary_test.go) |
| Every exported type has a doc comment. | [`TestEveryExportedTypeIsDocumented`](tests/structural/boundary_test.go) |
| Domain and application are at 100% statement coverage. | [`test.ps1`](test.ps1), run first by `build.ps1` |
| The support address is WhatDay's own and is `https`. | [`TestDonateURLIsWhatDaysOwn`](internal/application/menu_test.go) |

## Layers

```
cmd/whatday          composition root: builds the adapters, wires them, runs
internal/ui          Win32: the strip, the tray, the message loop
internal/application use cases and the ports they need
internal/domain      rules: the day, midnight, geometry, the palette
internal/infrastructure/
    clock            the system clock
    zone             the zone Windows is set to, named through ICU
    settings         settings.json, saved atomically
    runlog           the log, plus crash output pointed at it
    instance         one copy per user session
    setup            the install policy the setup program uses
installer            the Wails setup program, a facade over setup
tests/structural     the invariants above
```

Dependencies point inwards: `ui` and `infrastructure` depend on
`application`, which depends on `domain`. Only the composition root sees
everything.

### Domain

Pure functions over values handed in; no clock, no disk, no window.

- `DayName(instant, zone)`: the English weekday of an instant in a zone.
- `NextMidnight(instant, zone)`: the first instant after `instant` at which
  the zone's civil date moves on (see Midnight below).
- `WakeDelay(now, target, maxGap)`: how long to sleep on the way to a target,
  never longer than `maxGap`.
- Geometry: the strip's size from the taskbar height, clamping into a work
  area, the default position, placement across monitors and the drag
  threshold.
- The palette: six named colours, one home for each value.

### Application

One use-case object, `Indicator`, behind six ports declared in
[`ports.go`](internal/application/ports.go): `Clock`, `Zones`, `View`,
`Scheduler`, `SettingsStore` and `Log`. It also owns the menu model and the
About text, so the words WhatDay shows are decided here rather than in Win32
code.

`Refresh` is the single response to every event that could change the day:
the midnight wake-up, resume from sleep, a clock change, a timezone change.
It reads the zone, paints the day and re-arms the wake-up. One path means one
place to be right.

### Infrastructure

Each package implements a port or a piece of setup policy against the real
machine. Wherever the machine allows, their tests run against the real thing:
a real temporary folder, a real named mutex held by a second process, a child
process that really crashes, the real ICU.

### UI

The Win32 surface, written against `golang.org/x/sys/windows` with no toolkit.
It implements `View` and `Scheduler` and talks to the application through a
`Controller` interface. Its testable pieces (the menu mapping, the drawing,
the support entry, the icon, the monitor reading) are tested; the message loop
and window handling are verified by hand.

## How a day is shown

1. `main` opens the log, takes the single-instance lock and builds the parts.
2. `Window.Run` makes the strip, applies the dark Acrylic backdrop and asks
   the application to `Start`.
3. `Start` loads the settings (missing means defaults; unreadable means
   defaults plus a log line), paints the colour, then calls `Refresh`.
4. `Refresh` asks `Zones` for the zone, `Clock` for the instant, paints
   `DayName` and arms `Scheduler.WakeAt(NextMidnight(...))`.
5. The window's timer fires at the earlier of that midnight and one minute
   from now. Every firing calls `Refresh`, which re-arms.

### Why the timer never waits more than a minute

A Win32 timer counts elapsed time rather than wall-clock time; it also pauses
while the machine sleeps. Trusting one timer for hours would drift. Waking at least
once a minute puts any drift right within a minute; it also catches a missed
resume or time-change message and a timezone change nobody announced. The
cost is one refresh a minute; the idle CPU budget for it (NFR-PERF-001) has
not yet been measured.

### Midnight

Usually midnight is 00:00 on the next date. Some zones move their clocks
forward at midnight, so 00:00 never happens: Go's `time.Date` then answers an
instant still on the old date. WhatDay checks the candidate; where it fails,
it bisects to the nanosecond for the first instant whose date is later. The
naive version was measured wrong at 746 midnights across 598 zones between
2000 and 2100.

### The timezone

Go's `time.Local` is read once when the process starts, so a laptop that
changes zone would keep the old one. WhatDay reads the Windows zone key
(`GetDynamicTimeZoneInformation`) at every refresh and asks Windows' own ICU
(`icu.dll`, `ucal_getTimeZoneIDForWindowsID`) for the IANA name. The rules
come from the IANA database embedded in the binary (`time/tzdata`), so the
answer never depends on what the machine has installed. Where "Adjust for
daylight saving time automatically" is off, WhatDay uses the zone's standard
offset all year, as the Windows clock does. If the zone cannot be read, the
last good zone is kept and the fault is logged once.

## The strip

- A `WS_POPUP` window with `WS_EX_TOPMOST` and `WS_EX_TOOLWINDOW`: no frame,
  always on top, no taskbar button, no Alt+Tab entry.
- DWM gives it dark mode, rounded corners and the Acrylic backdrop, with the
  frame extended over the whole window.
- The day name is drawn white on a transparent 32-bit bitmap; each pixel's
  coverage then becomes the alpha of the chosen colour, premultiplied. Everywhere
  but the letters stays transparent, so the Acrylic shows through.
- The width fits the widest day name at the current height, so it never
  changes from day to day.
- The process is per-monitor DPI aware (V2). The height is the taskbar height
  of the monitor the strip is on, which Windows already reports in that
  monitor's pixels.

### Why it stays out of the taskbar rather than sitting on it

Measured before any code: a topmost window over the taskbar is covered by
the taskbar and stays covered. Rather than fight for the top, the strip is
clamped into the monitor's work area while it is dragged, which removes the
problem instead of watching for it.

### Fullscreen

The strip registers as an appbar and hides on `ABN_FULLSCREENAPP`, the signal
the taskbar itself hides on. The notification-state API was measured first
and rejected: it reported "busy" during a screenshot and hid the strip wrongly.

### A drag across monitors of different scales

Crossing onto a monitor of another scale sends `WM_DPICHANGED` mid-drag.
Laying out then would snap the strip back to the saved position, which is the
old one until the drag ends. So the drag sizes the strip itself and the
message is ignored while the button is down. This was found from the log and
confirmed fixed on the reference machine.

## Settings

`%APPDATA%\WhatDay\settings.json` holds the colour name, plus the position
once the strip has been dragged. The file's shape is its own type in the settings package, kept
apart from the port's `Settings` on purpose: the file is a format and the
port is a type. Saves go to a temporary file first, then one rename replaces
the target, so an interrupted save leaves the previous file intact. A save
that fails keeps the choice for the running session and logs why.

## Failing without falling over

A windowed program has no console, so an error that ends the run before the
window opens reaches nobody.

- The log is opened first. The process's standard error is pointed at it
  (`SetStdHandle`) before anything else runs. The Go runtime's own crash
  report therefore lands in the log.
- Where a standard error exists, crash output is copied to the log as well
  (`debug.SetCrashOutput`).
- No log folder means lines go to standard error; that is not a reason to
  refuse to start.
- A failed single-instance check starts anyway and says so: two copies beat
  none.
- Apart from a second copy stepping aside for the first, the only exit before
  the window opens is a window that cannot be created at all; that is
  logged.

## The support entry

`Support WhatDay (opens your browser)` hands WhatDay's PayPal address to the
desktop through `ShellExecute`. WhatDay opens no connection; the browser does.
Only an `https` address is handed over. A refusal is logged and shown in a
message box, so the entry never appears to do nothing. The address lives once,
in `internal/application/about.go`, beside the rest of the product's identity.

## The setup program

`installer/` is a second Go `main` package: a Wails application embedding the
built `WhatDay.exe` as a zip, with a hand-written page and no front-end build
step. It is a facade; the install policy lives in
`internal/infrastructure/setup`, ported from PigeonPost.

- **One reading of the machine decides the route.** `Decide` compares the
  recorded version with this one: install, update, go back or manage
  (repair). Asked with `-uninstall`, it goes straight to removal.
- **Per user.** Files under `%LOCALAPPDATA%\Programs\WhatDay`; the Apps list
  entry and the sign-in entry under `HKCU`. Windows never asks for
  administrator rights.
- **WhatDay is closed first** by image name, never by process tree, before any
  file is touched.
- **The payload is fenced**: every archive entry is checked against the
  install folder before any is written.
- **A step log** in `%TEMP%\WhatDaySetup.log`, flushed after every step, since
  the worst setup failures never raise.
- Uninstall removes the shortcut, the sign-in entry, the Apps list entry,
  WhatDay's settings and log folders, then the install folder by a hidden
  shell that waits for setup to exit and release its own executable.

## Versioning

`VERSION` is the single source. `build.ps1` passes it to both programs through
`-ldflags "-X main.appVersion=..."` (a `var`, since `-X` cannot reach a
`const`) and writes it into each executable's properties from generated files
that are never committed. Nothing in the source holds a version.

## Decisions

| Decision | Chosen | Rejected: what and why |
|---|---|---|
| UI toolkit for the strip | Raw Win32 through `x/sys/windows` | A toolkit: constraint C-2 keeps the application to the standard library and `x/sys`; one popup window, one menu and one message box need nothing more. |
| Where the day appears | A strip above the taskbar | A taskbar button: abandoned before measurement in favour of the strip. Drawing on the taskbar: measured, it covers a topmost window. |
| Background | Dark Acrylic | Mica, Mica Alt and solid colours: tried by eye on the reference machine; they looked alike and none matched the taskbar, which is tinted by the wallpaper. |
| Timezone | Follow Windows, re-read every refresh | A fixed zone (the first design): a laptop travels. `time.Local`: read once per process. |
| Fullscreen signal | `ABN_FULLSCREENAPP` | The notification-state API: measured hiding the strip for a screenshot. |
| Clicking the strip | Does nothing | Opening the menu: the tray was decided as the one control surface (Q-1). |
| Light mode | None: always dark | Following the Windows mode: the owner uses dark mode only. |
| Setup program | Wails, ported from PigeonPost | Designing one afresh: the house installer already exists. |
