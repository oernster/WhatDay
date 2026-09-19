# WhatDay: Software Requirements Specification

Status: baselined 2026-09-19. Changes from here arrive as numbered amendments
with a reason (section 6). One open question remains (Q-5); it gates only the
choice of palette shades, not the rest of the build.

## 1. Introduction

### 1.1 Purpose

WhatDay tells its user the day of the week. That is all it does. Every clock
on a Windows desktop shows the time and the date; none shows the day. WhatDay
puts the day name in a small, always visible strip beside the taskbar.

### 1.2 Intended audience

The owner (Oliver Ernster), who is both the only user and the developer, plus
any AI assistant working on the code. The repository is public; the product is
built for one person and makes no attempt to serve anyone else.

### 1.3 Scope

In scope:

- A borderless, always-on-top strip showing the current day name, in
  English, in whatever timezone Windows is set to, so it follows a laptop
  that travels (Amendment 4).
- A notification-area (tray) icon whose menu is the only control surface.
- A choice of colour for the day name, remembered across restarts.
- The strip's position, chosen by dragging, remembered across restarts.
- Start at login, as an option in the setup program (Amendment 7).
- A setup program in the house style, ported from PigeonPost's `installer/`.

Out of scope (decided; see also 3.5 Won't this time):

- Any language other than English.
- A timezone choice inside WhatDay. The zone is the one Windows is set to.
- Showing the date, the time or a week number.
- A taskbar button, a pinned taskbar button or anything drawn inside the
  taskbar itself (feasibility measured and rejected: Appendix A, M-1).
- Any background choice. The background is fixed (FR-015).
- Windows 10 and every operating system other than Windows. WhatDay is a
  Windows application only, permanently.
- Any network access, including an update check.
- Correcting the Windows clock. WhatDay trusts the system clock; keeping it
  right is Windows' job (time synchronisation), as is any leap second.

### 1.4 Definitions

| Term | Meaning |
|---|---|
| Indicator | The borderless strip that shows the day name. |
| Tray icon | WhatDay's icon in the Windows notification area. |
| Menu | The popup menu opened from the tray icon. |
| Windows zone | The timezone Windows is set to now, by hand or by "Set time zone automatically". WhatDay reads its key and translates it to an IANA zone with Windows' own ICU. |
| Local day | The day of the week of the current instant in the Windows zone. |
| Local midnight | The first instant at which the Windows zone's civil date moves on. Where clocks jump forward at 00:00, that is the moment of the jump. |
| System clock | The Windows clock as the process reads it. Every "within 1 s" in this document is measured against it. |
| Gregorian calendar | The proleptic Gregorian calendar: a year is a leap year when divisible by 4, except century years, which are leap years only when divisible by 400 (2000 and 2400 leap; 1900 and 2100 common). |
| Windows mode | The Windows setting "Choose your default Windows mode", stored as `SystemUsesLightTheme`. It is what the taskbar follows; it is distinct from the app mode. |
| Work area | The rectangle of a monitor not occupied by the taskbar, as reported by Windows. |
| Drag threshold | The Windows system drag distance (`SM_CXDRAG`, `SM_CYDRAG`). |
| Acrylic | The Windows 11 translucent system backdrop material (`DWMSBT_TRANSIENTWINDOW`). |
| Fullscreen signal | The shell's `ABN_FULLSCREENAPP` appbar notification; the same signal the taskbar hides on. |
| Reference machine | Oliver's desktop: Windows 11 build 26200, four monitors including 3440x1440 at 100% scale and 3840x2400 at 250% scale. |

### 1.5 References

- `C:\Users\Oliver\Development\pigeonpost\installer\`: the installer to port.
- WCAG 2.2 success criterion 1.4.3 (contrast minimum, large text 3:1).
- Appendix A of this document: feasibility measurements taken 2026-09-19.

## 2. Overall description

### 2.1 Product perspective

A new, standalone Windows desktop utility. It has no server, no network
access, no dependencies on other applications and no data other than its own
settings file.

### 2.2 User classes

One: the owner, on the reference machine. No administrator rights are needed
or requested at any point.

### 2.3 Operating environment

Windows 11 22H2 (build 22621) or later on x64. The lower bound is set by the
Acrylic backdrop attribute (`DWMWA_SYSTEMBACKDROP_TYPE`), which first exists in
that build.

### 2.4 Constraints

- C-1 Language: Go 1.26, pure Go with cgo disabled.
- C-2 Dependencies: the Go standard library plus `golang.org/x/sys/windows`
  for the application. No Wails, walk or other UI toolkit in the application;
  the setup program is a Wails application because that is the house
  installer (C-5).
- C-3 Layering: `internal/{domain,application,infrastructure,ui}` with a
  `cmd/whatday` composition root; enforced by a structural test.
- C-4 Timezone rules are embedded in the binary (`time/tzdata`), so the day
  never depends on what the machine has installed.
- C-5 The setup program is ported from PigeonPost's `installer/`, not
  designed afresh.
- C-6 Licence: GPL-3.0 (already in the repository).
- C-7 `VERSION` at the repository root is the single source of the version.

### 2.5 Assumptions and dependencies

| ID | Assumption | Owner | Status |
|---|---|---|---|
| A-1 | Timezone rules change now and then (a country moves its clocks). The rules are embedded in the binary, so a change reaches WhatDay by rebuilding with a newer Go toolchain; until then the affected zone may be wrong. | Oliver | Confirmed 2026-09-19; widened from UK rules by Amendment 4 |
| A-2 | The taskbar stays at the bottom edge and is not set to auto-hide. The work-area rule (FR-018) relies on a taskbar that reserves space. | Oliver | Confirmed 2026-09-19 |
| A-3 | Oliver supplies the tray, application and installer artwork as a master PNG with a transparent background. | Oliver | Confirmed and met 2026-09-19: `assets/application-icon.png`, 1254x1254 RGBA, all four corners alpha 0 (measured, Appendix A M-8). |

## 3. Requirements

Priorities use MoSCoW. Every requirement names the test or the manual check
that verifies it.

### 3.1 Functional requirements: the day

**FR-001 Day name**
- Priority: Must
- Requirement: The indicator shall display the local day as one of
  `Monday`, `Tuesday`, `Wednesday`, `Thursday`, `Friday`, `Saturday` or
  `Sunday`: English, full word, initial capital.
- Acceptance: Given the instant 2026-09-19T15:44:49+01:00, the indicator reads
  `Saturday`.
- Verified by: `internal/domain/day_test.go::TestDayNameAtInstant`

**FR-002 Follow the Windows zone** (Amendment 4)
- Priority: Must
- Requirement: The day service shall derive the local day from the zone
  Windows is set to at each refresh, so a change of zone while WhatDay runs
  takes effect within 1 minute. Day names stay English in every zone.
- Acceptance: At 22:30 UTC on Saturday 19 September 2026: in London the
  indicator reads `Saturday` and waits for 23:00 UTC; after the zone changes
  to New York it reads `Saturday` and waits for 04:00 UTC; after it changes to
  Sydney it reads `Sunday`.
- Verified by: `internal/application/zone_test.go::TestTravelFollowsNewZone`;
  `internal/infrastructure/zone/zone_test.go` (key to IANA name through ICU,
  measured on the reference machine).

**FR-007 Zone failure**
- Priority: Must
- Requirement: If the Windows zone cannot be read or translated, then the day
  service shall keep using the last zone it read (else the zone Go read at
  startup) and shall log the fault once until it changes.
- Verified by: `zone_test.go::TestFailureAnswersTheLastGoodZone`,
  `internal/application/zone_test.go::TestZoneFaultLoggedOncePerFault`

**FR-008 Daylight saving switched off**
- Priority: Must
- Requirement: Where "Adjust for daylight saving time automatically" is off,
  the day service shall keep the zone's standard offset all year, as the
  Windows clock does.
- Verified by: `zone_test.go::TestDaylightSavingOffKeepsStandardTime`

**FR-003 Change at local midnight**
- Priority: Must
- Requirement: When local midnight passes, the indicator shall show the new
  day name within 1 s.
- Rationale: "It will always accurately change at midnight."
- Acceptance: In London, given the scheduler computes the next midnight after
  2026-03-29T12:00:00+01:00 (the first BST day, 23 hours long), then the
  answer is 2026-03-29T23:00:00Z. Given 2026-10-25T12:00:00Z (the first GMT
  day, 25 hours long), then the answer is 2026-10-26T00:00:00Z. In
  America/Sao_Paulo, whose clocks jumped from 00:00 to 01:00 on 2018-11-04,
  Sunday begins at 03:00 UTC, the moment of the jump.
- Verified by: `internal/domain/midnight_test.go::TestNextMidnightAcrossTransitions`,
  `internal/domain/zones_test.go::TestMidnightInAGapStartsAtTheJump` plus a
  manual observation at a real midnight.

**FR-004 Resume from sleep**
- Priority: Must
- Requirement: When the machine resumes from sleep or hibernation, the
  indicator shall show the local day of the resume instant within 1 s.
- Acceptance: Given the machine sleeps on Monday 23:50 local time and resumes
  on Tuesday 07:00, then within 1 s of resume the indicator reads `Tuesday`.
- Verified by: `internal/application/refresh_test.go::TestResumeReevaluates`
  (fake clock plus fake power event) plus a manual sleep test.

**FR-005 Clock or timezone change**
- Priority: Must
- Requirement: When the system clock is changed, the indicator shall show the
  local day of the new instant within 1 s.
- Acceptance: Given the clock is moved from Monday 12:00 to Wednesday 12:00,
  then within 1 s the indicator reads `Wednesday`; moved back, it reads
  `Monday`.
- Verified by: `internal/application/refresh_test.go::TestClockChangeReevaluates`

**FR-006 Calendar correctness**
- Priority: Must
- Requirement: The day service shall name the correct weekday for every date
  of the Gregorian calendar from 2000-01-01 to 2399-12-31, including 29
  February in leap years, the common century year 2100 and year boundaries.
- Rationale: Leap years and century rules are where a day calculation goes
  quietly wrong.
- Acceptance: 2000-02-29 is Tuesday; 2024-02-29 is Thursday; 2028-02-29 is
  Tuesday; 2100-02-28 is Sunday and the next day is 2100-03-01, Monday;
  2026-12-31 is Thursday and 2027-01-01 is Friday. (Each checked against
  Python's calendar, independent of Go.)
- Verified by: `internal/domain/day_test.go::TestEveryDayAgainstIndependentFormula`,
  which walks every date in the range and compares against a separate
  weekday formula (Sakamoto's method) written into the test, not against Go's
  own `time` package; plus `internal/domain/midnight_test.go::TestMidnightChain`,
  which follows next midnight in London from 2000-01-01 to 2399-12-31 and
  asserts every step lands on the following civil date at 00:00, through
  every leap day and every GMT/BST change; plus
  `internal/domain/zones_test.go::TestEveryZoneMidnightChain`, which does the
  same in every zone of the embedded tz data (598 on Go 1.26.3) from 2000 to
  2100 and confirms the only dates skipped are Samoa's and Tokelau's
  2011-12-30.

### 3.2 Functional requirements: the indicator

**FR-010 No window furniture**
- Priority: Must
- Requirement: The indicator shall have no title bar, frame, caption buttons
  or window menu; its whole visible area is the day name on its background.
- Acceptance: Inspection of the running indicator; its window style is
  `WS_POPUP` with no `WS_CAPTION`, `WS_THICKFRAME` or `WS_SYSMENU`.
- Verified by: inspection. `internal/ui/loop.go` creates the window with
  `WS_POPUP` alone; no automated test covers the style.

**FR-011 Rounded corners**
- Priority: Should
- Requirement: The indicator shall use the Windows 11 system rounded corners.
- Verified by: inspection; the corner attribute call returns `S_OK`
  (measured in probe round 3, Appendix A M-5).

**FR-012 Always on top**
- Priority: Must
- Requirement: The indicator shall stay above all ordinary application
  windows.
- Acceptance: Given the indicator above a maximised application, when the
  application is clicked, then the indicator remains visible.
- Verified by: manual; probe round 3 measured this (Appendix A M-3).

**FR-013 No taskbar button, no Alt+Tab entry**
- Priority: Must
- Requirement: The indicator shall not appear on the taskbar or in the
  Alt+Tab switcher.
- Verified by: inspection. The window carries `WS_EX_TOOLWINDOW`
  (`internal/ui/loop.go`); no automated test covers the style.

**FR-014 Size**
- Priority: Must
- Requirement: The indicator's height shall equal the taskbar height of the
  monitor it is on, scaled to that monitor's DPI; its width shall fit the
  widest day name plus padding. The width shall not change from day to day.
- Acceptance: On the 3440x1440 monitor at 100% with a 48 px taskbar, the
  indicator is 48 px high. On the 3840x2400 monitor at 250%, it is as high as
  that monitor's taskbar.
- Verified by: `internal/domain/geometry_test.go::TestSizeFitsWidestDay`
  plus manual on both monitors.
- Note: the probe did not rescale between monitors (Appendix A M-6), so the
  250% half of this requirement goes beyond what has been measured.

**FR-015 Background**
- Priority: Must
- Requirement: The indicator's background shall be the dark Acrylic material,
  whatever the Windows mode (Amendment 2).
- Rationale: Measured in probe round 3: Acrylic was judged right by eye;
  Mica, Mica Alt and the two solid colours were near-identical to one another
  and none matched (Appendix A M-4).
- Verified by: inspection.

**FR-016 Follow Windows mode**
- Retired by Amendment 2. The indicator is always dark; the number is not
  reused.

**FR-017 Drag to move**
- Priority: Must
- Requirement: When the user presses the left mouse button on the indicator
  and moves beyond the drag threshold, the indicator shall follow the pointer
  until the button is released.
- Verified by: `internal/domain/geometry_test.go::TestThresholdSeparatesClickFromDrag`
  plus manual.

**FR-018 Never on the taskbar**
- Priority: Must
- Requirement: While the indicator is dragged, the indicator shall keep its
  whole rectangle inside the work area of the monitor nearest its centre.
- Rationale: Measured: a topmost strip on the taskbar is covered by the
  taskbar and stays covered (Appendix A M-2). Constraining the position
  removes the problem rather than watching for it.
- Acceptance: Given a work area of (0,0)-(3440,1392) and a 175x48 indicator,
  when it is dragged to a top-left of (3400,1420), then it is placed at
  (3265,1344).
- Verified by: `internal/domain/geometry_test.go::TestClampToWorkArea`

**FR-019 A click does nothing**
- Priority: Must
- Requirement: When the left button is pressed and released on the indicator
  without passing the drag threshold, the indicator shall take no action.
- Rationale: The tray is the control surface (Q-1, decided 2026-09-19).
- Verified by: `TestThresholdSeparatesClickFromDrag`

**FR-020 Hide for fullscreen**
- Priority: Must
- Requirement: When the fullscreen signal reports a fullscreen application
  opening, the indicator shall hide; when it reports one closing, the
  indicator shall reappear on top.
- Acceptance: A fullscreen browser video hides it; leaving fullscreen restores
  it. A screenshot taken with Win+Shift+S does not hide it.
- Verified by: manual; measured in probe round 3 (Appendix A M-3).

**FR-021 Default position**
- Priority: Must
- Requirement: Where no saved position exists, the indicator shall be placed
  at the bottom-right corner of the primary monitor's work area.
- Acceptance: With work area (0,0)-(3440,1392) and a 175x48 indicator, it is
  placed at (3265,1344) (measured, Appendix A M-5).
- Verified by: `internal/domain/geometry_test.go::TestDefaultPosition`

**FR-022 Lost monitor**
- Priority: Must
- Requirement: If the centre of the indicator at its saved or current
  position lies on no attached monitor at startup or after a display change,
  then the indicator shall move to the default position (FR-021). Otherwise
  the indicator shall stay on the monitor holding its centre, clamped into
  that monitor's work area (FR-018). (Amendment 1.)
- Acceptance: Given a saved position on a monitor that is no longer attached,
  when WhatDay starts, then the indicator appears at the default position.
  Given a saved top-left of (3000,1370) on the measured monitor (centre over
  the taskbar), then the indicator appears at (3000,1344).
- Verified by: `internal/domain/geometry_test.go::TestLostMonitorFallsBack`

**FR-023 Display changes**
- Priority: Must
- Requirement: When the display configuration, a monitor's DPI or the taskbar
  size changes, the indicator shall re-measure its size and re-apply FR-014,
  FR-018 and FR-022 within 1 s.
- Verified by: manual (resolution change, DPI change, monitor unplugged).

**FR-024 Explorer restart**
- Priority: Must
- Requirement: When Windows Explorer restarts, WhatDay shall re-add the tray
  icon and re-register for the fullscreen signal within 1 s.
- Verified by: manual (restart Windows Explorer from Task Manager); the probe
  handles `TaskbarCreated` but this was not exercised.

### 3.3 Functional requirements: the tray and menu

**FR-030 Tray icon**
- Priority: Must
- Requirement: While WhatDay runs, the tray shall show WhatDay's icon, drawn
  from Oliver's artwork (A-3), with the tooltip `WhatDay`.
- Verified by: inspection.

**FR-031 Menu opens**
- Priority: Must
- Requirement: When the tray icon is clicked with the left or right mouse
  button, WhatDay shall open the menu at the pointer.
- Verified by: manual; measured in probe rounds 2 and 3.

**FR-032 Menu contents**
- Priority: Must
- Requirement: The menu shall contain exactly: `About WhatDay`,
  `Support WhatDay (opens your browser)`, a separator, one item per palette
  colour with the current colour checked, a separator, then `Quit WhatDay`
  (Amendments 5 and 6).
- Verified by: `internal/application/menu_test.go::TestMenuModel`,
  `internal/ui/ui_test.go::TestMenuEntriesFollowTheModel`

**FR-035 Quit**
- Priority: Must
- Requirement: When `Quit WhatDay` is chosen, WhatDay shall remove its tray
  icon, close the indicator and exit, releasing the single-instance lock. It
  shall start again at the next login where that option is on (FR-060).
  (Amendment 5.)
- Verified by: manual; the log records `quit from the tray` and the process
  ends.

**FR-033 Choose a colour**
- Priority: Must
- Requirement: When a colour is chosen from the menu, the indicator shall
  repaint the day name in that colour within 1 s.
- Verified by: `internal/application/colour_test.go::TestChooseColourRepaints`

**FR-034 About**
- Priority: Must
- Requirement: When `About WhatDay` is chosen, WhatDay shall show a dialog
  stating the product name, `© 2026 Oliver Ernster` and the open-source works
  WhatDay is built with, each with its licence: Go and golang.org/x/sys
  (BSD 3-Clause, © 2009 The Go Authors) and the IANA Time Zone Database
  (public domain). Nothing else. (Amendment 3.)
- Verified by: `internal/application/menu_test.go::TestAboutContent`

**FR-036 Support** (Amendment 6)
- Priority: Should
- Requirement: When `Support WhatDay (opens your browser)` is chosen, WhatDay
  shall hand `https://www.paypal.com/ncp/payment/7LC63AH9F2UYU` to the
  desktop to open in the default browser. WhatDay opens no connection itself;
  only an `https` address is handed over. If the desktop declines, WhatDay
  shall say so in a message box and in the log.
- Verified by: `internal/application/menu_test.go::TestDonateURLIsWhatDaysOwn`,
  `internal/ui/browser_test.go::TestDonateAsksForWhatDaysAddressOnly`,
  `TestDonateRefusedSaysSo`, `TestOpenExternalRefusesAnythingButHTTPS`

### 3.4 Functional requirements: colours

**FR-040 Palette**
- Priority: Must
- Requirement: The palette shall hold named colours, one shade each, chosen
  for the dark Acrylic background (Amendment 2).
- Names (Q-2, decided 2026-09-19), in menu order: Red, Amber, Green, Blue,
  Purple, Neutral (white).
- Shades, with contrast against the measured taskbar mean `#1C222F` and the
  provisional worst case `#3A3A3A` (Q-5): Red `#FF6666` 5.57 / 3.98, Amber
  `#FFB020` 8.70 / 6.22, Green `#4CD964` 8.65 / 6.18, Blue `#5AA8FF`
  6.43 / 4.59, Purple `#C58CFF` 6.50 / 4.64, Neutral `#FFFFFF` 15.92 / 11.37.
  Closest pair: Blue and Purple at CIEDE2000 26.4.
- Verified by: NFR-COL-001 and NFR-COL-002.

**FR-041 Default colour**
- Priority: Must
- Requirement: Where no colour has been saved, the indicator shall use Red
  (Q-3, decided 2026-09-19).
- Verified by: `internal/application/settings_test.go::TestDefaults`

### 3.5 Functional requirements: settings and lifecycle

**FR-050 Remember colour and position**
- Priority: Must
- Requirement: When the colour changes or a drag ends, WhatDay shall save the
  colour and the position to `%APPDATA%\WhatDay\settings.json`.
- Verified by: `internal/infrastructure/settings/store_test.go::TestRoundTrip`

**FR-051 Atomic save**
- Priority: Must
- Requirement: The settings store shall write to a temporary file and replace
  the settings file in one rename, so an interrupted save leaves the previous
  file intact.
- Verified by: `store_test.go::TestSaveIsAtomic`

**FR-052 Missing settings**
- Priority: Must
- Requirement: If the settings file does not exist, then WhatDay shall use the
  defaults (FR-021, FR-041) without reporting a fault.
- Verified by: `store_test.go::TestMissingFileGivesDefaults`

**FR-053 Unreadable settings**
- Priority: Must
- Requirement: If the settings file cannot be read or parsed, then WhatDay
  shall use the defaults, record the reason in the log and carry on.
- Verified by: `store_test.go::TestCorruptFileGivesDefaults`

**FR-054 Unwritable settings**
- Priority: Must
- Requirement: If the settings file cannot be written, then WhatDay shall keep
  the new choice for the running session, record the reason in the log and
  carry on.
- Verified by: `internal/application/colour_test.go::TestUnwritableKeepsSessionValue`,
  `store_test.go::TestUnwritableFolderIsAnError`

**FR-060 Start at login**
- Priority: Must
- Requirement: The setup program shall offer `Start WhatDay when I sign in to
  Windows` on its install, update and repair screens. Where the option is on,
  it shall register WhatDay to start at login for the installing user
  (`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`), the value being the
  executable's path in plain double quotes; where it is off, it shall remove
  that entry. On a fresh install the option opens on; otherwise it opens on
  whether an entry naming an existing file is present. On the repair screen a
  change applies at once. (Amendment 7.)
- Acceptance: After install with the option on and a reboot, the indicator is
  visible without the user starting anything. With it off, nothing starts.
- Verified by: `setup_test.go::TestQuotedWritesThePathAsWindowsDoes`,
  `TestRunTargetReadsTheEntryBack`,
  `TestStartsAtLoginOnlyForAnEntryNamingARealFile`; the setup page driven
  against a recording backend; plus a manual reboot test.

**FR-061 Single instance**
- Priority: Must
- Requirement: When WhatDay starts while another instance is running for the
  same user, the new instance shall exit without showing an indicator or a
  tray icon.
- Verified by: `internal/infrastructure/instance/lock_test.go::TestSecondInstanceRefused`

**FR-062 Log**
- Priority: Must
- Requirement: WhatDay shall write its log to `%LOCALAPPDATA%\WhatDay\WhatDay.log`
  and point the process's standard error at that file before anything else
  runs, so a crash leaves a record.
- Verified by: `internal/infrastructure/runlog/runlog_test.go`

**FR-063 Panics recorded**
- Priority: Must
- Requirement: If a panic occurs on the message loop, then WhatDay shall
  record it in the log with its stack.
- Verified by: `runlog_test.go::TestAPanicOnAnotherGoroutineIsInTheLog`,
  `TestWhereTheRunHasNoErrorOutputEverythingIsInTheLog`

### 3.6 Functional requirements: setup program

**FR-070 Per-user install**
- Priority: Must
- Requirement: The setup program shall install WhatDay to
  `%LOCALAPPDATA%\Programs\WhatDay` without requesting administrator rights.
- Verified by: manual install on the reference machine.

**FR-071 Install, update, go back, repair, uninstall** (Amendment 8)
- Priority: Must
- Requirement: The setup program shall offer install, update, going back to
  an older version, repair and uninstall, with a screen for each. It shall
  register in the Windows Apps list with two commands. Uninstall opens setup
  on its removal screen; Modify opens it on the screen for the version
  installed, where Repair is offered. The Apps list shall offer no Repair of
  its own (`NoRepair`), so repairing always goes through setup's screen.
- Verified by: manual; each route exercised once, the Apps list entry
  inspected.

**FR-072 Stop before replace**
- Priority: Must
- Requirement: When updating, repairing or uninstalling, the setup program
  shall stop a running WhatDay before touching its files.
- Verified by: manual with WhatDay running.

**FR-073 Uninstall removes everything WhatDay wrote**
- Priority: Must
- Requirement: When uninstalling, the setup program shall remove the install
  directory, the login entry, the Apps list entry and the settings (Q-4,
  decided 2026-09-19).
- Verified by: manual; registry and folders inspected afterwards.

**FR-074 House style** (Amendment 8)
- Priority: Must
- Requirement: The setup program shall follow PigeonPost's installer: 126 px
  mark, centred body, progress bar. It shall be dark only, with no theme
  toggle, matching the strip (Amendment 2).
- Verified by: inspection beside PigeonPost's setup program.

### 3.7 Non-functional requirements

**NFR-PERF-001 Idle CPU**: Averaged over 10 minutes with no user interaction,
WhatDay shall use less than 0.1% CPU on the reference machine, measured with
`Get-Process` CPU time sampled at the start and end.

**NFR-PERF-002 Memory**: WhatDay's working set shall stay under 30 MB after 24
hours of running, measured with `Get-Process`.

**NFR-PERF-003 Startup**: The indicator shall be visible within 2 s of process
start on the reference machine, measured from the log's start line to its
shown line.

**NFR-COL-001 Contrast**: Every palette shade shall reach at least 3.0:1 WCAG
contrast against the measured dark Acrylic background. The day name is
large text (at least 24 px, which is 18 pt), so 3:1 is the WCAG threshold. Verified
by a test over the palette table against the background constants recorded in
Appendix A.

**NFR-COL-002 Distinct colours**: Every pair of palette shades shall differ by a CIEDE2000 distance of at least 20, so no two choices
look alike. Verified by the same test.

**NFR-PRIV-001 No network**: WhatDay and its setup program shall make no
network connections. The Support entry (FR-036) hands an address to the
browser, which is the program that connects. Verified by
`tests/structural/boundary_test.go::TestNoNetworkImports`, which forbids `net`
and every `net/` package in the repository's own code.

**NFR-MAINT-001 Coverage**: `internal/domain` and `internal/application`
shall be held at 100% statement coverage by `test.ps1`, which `build.ps1` runs
first and cannot skip.

**NFR-MAINT-002 Structure**: A structural test shall enforce the layering
(C-3), domain purity (no `os`, `time.Now`, `syscall` or `x/sys` in the
domain), the 400-line module cap and its 381 to 399 danger band.

**NFR-MAINT-003 Checks**: `gofmt`, `go vet` and `staticcheck` shall report
nothing.

**NFR-MAINT-004 Docs**: The repository shall carry `README.md` (with who it is
for and not for), `ARCHITECTURE.md` (invariants linked to their tests) and
`VERSION`. The README's header (the `# WhatDay` title and the owner's
two-line opening beneath it) stays verbatim; everything added goes below it.

### 3.8 Won't this time

| Item | Reason |
|---|---|
| Update check | Personal tool; no network (NFR-PRIV-001). |
| Background choice | Measured: the alternatives looked alike (M-4). |
| Auto-hide taskbar support | Rests on A-2; the work area does not exclude an auto-hidden taskbar. |
| Taskbar at the top or sides | Rests on A-2. |
| Localisation | Out of scope (1.3). Day names are English whatever the zone. |
| Light mode | Amendment 2: the owner uses dark mode only. In light mode the indicator stays dark. |

## 4. Other requirements

### 4.1 Legal

GPL-3.0 for the application and the setup program. No third-party assets.

### 4.2 Internationalisation

None by decision: English day names in every zone.

### 4.3 Risk

An FMEA would be disproportionate for a single-user utility with no data of
value; judged not warranted.

## 5. Appendices

### Appendix A: Feasibility measurements (2026-09-19)

All taken on the reference machine with three throwaway probes, logged to a
file and read back. Not project code.

| ID | Question | Measured |
|---|---|---|
| M-1 | Can a taskbar button carry the day? | Abandoned before testing in favour of the indicator; the pinned-button behaviour was never measured. |
| M-2 | Does the taskbar cover a topmost strip on it? | Yes: `COVERED by window class "Shell_TrayWnd"` at 15:45:17, still covered at 15:47:12. |
| M-3 | Fullscreen detection | The notification state API reported "busy" during a Win+Shift+S screenshot and hid the strip wrongly. The shell's `ABN_FULLSCREENAPP` fired for a fullscreen video (15:52:39 to 15:52:41) and not for a screenshot. |
| M-4 | Background | Taskbar pixels sampled at `#1F222D`, `#1B2232`, `#1C2333`, `#1D232F`, `#1C242E` (mean `#1C222F`): wallpaper-tinted, so no flat colour matches. Acrylic chosen by eye; Mica, Mica Alt and the solid modes looked alike. |
| M-5 | Win32 calls on a borderless popup | Dark mode, rounded corners, frame extension and Acrylic backdrop all returned `S_OK`. Default placement (3265,1344) in work area (0,0)-(3440,1392). |
| M-6 | Other monitors | Dragged to (3442,3672) and judged working by eye. No DPI change was logged and the probe does not rescale, so FR-014's 250% case is unmeasured. |
| M-7 | Contrast of the probe colours | Dark mode against `#1C222F`: Red 3.51, Green 5.65, Blue 4.22, Amber 7.38, Purple 3.98. Light mode against an assumed `#F3F3F3`: Red 4.08, Green 2.54, Blue 3.40, Amber 1.94, Purple 3.61. |
| M-8 | Artwork | `assets/application-icon.png`: 1254x1254, 8-bit RGBA; corners alpha 0; 24.2% of pixels alpha 0; visible bounds x 47 to 1191, y 24 to 1207. The artwork body sits near alpha 252 rather than 255 (0.1% of pixels fully opaque), typical of automatic background removal; to be inspected in the generated icons. |

### Appendix B: Open questions

Q-1 to Q-4 were decided on 2026-09-19 and now live in FR-019, FR-040, FR-041
and FR-073. Q-6 was decided the same day: follow the Windows zone
(Amendment 4, FR-002).

| ID | Question | Plan | Owner | Due |
|---|---|---|---|---|
| Q-5 | What colour is the indicator's own dark Acrylic background? M-4 measured the taskbar rather than the indicator. Acrylic blurs what is behind it, so a white window behind the indicator lightens it. | Until measured, check shades against the taskbar mean `#1C222F` plus a lighter worst case. Confirm by sampling the running indicator over a white window once the application exists. | Claude | First run of the application |

### Appendix C: Build order

1. Domain: day name, next local midnight, geometry (size, clamp, default,
   lost monitor), drag threshold, palette with its contrast and distance
   tests.
2. Application: refresh on midnight, resume and clock change; menu model;
   colour choice; settings defaults.
3. Infrastructure: settings store, single-instance lock, run log, Win32
   window, tray, appbar, DWM, timers and power events.
4. UI and the composition root.
5. Setup program, ported from PigeonPost.

## 6. Amendments

| No. | Date | Requirement | Change | Reason |
|---|---|---|---|---|
| 8 | 2026-09-19 | FR-071, FR-074 | FR-071 names going back as a route and states the Apps list entry as built: Uninstall and Modify, no Repair of its own. FR-074 makes the setup program dark only, with no theme toggle. | Owner's decision after the docs pass found both unmet: amend the specification to match what is built rather than build to it. |
| 7 | 2026-09-19 | FR-060, FR-035, scope | Starting at login becomes an option in the setup program, on by default for a fresh install, applied at once on the repair screen. The Run, uninstall and modify values are written as plain quoted paths. | Owner's decision: an option to start with Windows. Measured: the values had been written with Go's %q, which doubled every backslash in the registry. |
| 6 | 2026-09-19 | FR-032, new FR-036, NFR-PRIV-001 | The tray menu gains `Support WhatDay (opens your browser)` after About, handing WhatDay's own PayPal page to the browser. | Owner's decision: a donation link in the tray, as the other apps carry one. |
| 5 | 2026-09-19 | FR-032, new FR-035, scope, Won't list | The tray menu ends with a separator and `Quit WhatDay`. | Owner's decision after the first run: without it the only way to stop WhatDay was Task Manager. |
| 4 | 2026-09-19 | FR-002, FR-003 to FR-006, new FR-007 and FR-008, scope, glossary, Q-6 | The day follows the zone Windows is set to, re-read at every refresh, instead of Europe/London. Midnight is the first instant the local date moves on, found by bisection where clocks jump at 00:00. | Owner's decision: an English speaker may travel with the laptop. Measured: the naive midnight was wrong at 746 midnights across 598 zones from 2000 to 2100; Go's time.Local is read once per process, so it cannot follow a zone change. |
| 3 | 2026-09-19 | FR-034 | About states the copyright and the open-source works used, with their licences; the version and WhatDay's own licence are no longer shown there. | Owner's decision: About "should simply" credit the open-source providers and the author. Licences read from each work's own LICENSE or README. |
| 2 | 2026-09-19 | FR-015, FR-016, FR-040, NFR-COL-001, NFR-COL-002 | Light mode dropped: the indicator is always dark Acrylic; FR-016 retired; one shade per colour. | Owner's decision: dark mode only, "this is for me not the world". |
| 1 | 2026-09-19 | FR-022 | "Lost" is judged by the monitor's whole rectangle, not its work area; a strip still on its monitor is clamped, not moved. | `TestPlaceClampsSavedPositionOverlappingTaskbar` failed against the baseline wording: a strip whose centre sat over the taskbar was sent to the default corner though its monitor was attached. |
