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

- A borderless, always-on-top strip showing the current day name in
  Europe/London.
- A notification-area (tray) icon whose menu is the only control surface.
- A choice of colour for the day name, remembered across restarts.
- The strip's position, chosen by dragging, remembered across restarts.
- Start at login.
- A setup program in the house style, ported from PigeonPost's `installer/`.

Out of scope (decided; see also 3.5 Won't this time):

- Any language other than English.
- Any timezone other than Europe/London, including following the Windows
  timezone setting.
- Showing the date, the time or a week number.
- A taskbar button, a pinned taskbar button or anything drawn inside the
  taskbar itself (feasibility measured and rejected: Appendix A, M-1).
- Any background choice. The background is fixed (FR-015).
- Windows 10, macOS and Linux.
- Any network access, including an update check.
- An Exit item in the menu.
- Correcting the Windows clock. WhatDay trusts the system clock; keeping it
  right is Windows' job (time synchronisation), as is any leap second.

### 1.4 Definitions

| Term | Meaning |
|---|---|
| Indicator | The borderless strip that shows the day name. |
| Tray icon | WhatDay's icon in the Windows notification area. |
| Menu | The popup menu opened from the tray icon. |
| London day | The day of the week of the current instant in the IANA zone Europe/London (GMT in winter, BST in summer). |
| London midnight | The instant at which the Europe/London civil date changes. |
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
| A-1 | UK daylight saving rules do not change during the product's life. If they do, a rebuild with a newer Go toolchain carries the new rules. | Oliver | Confirmed 2026-09-19 |
| A-2 | The taskbar stays at the bottom edge and is not set to auto-hide. The work-area rule (FR-018) relies on a taskbar that reserves space. | Oliver | Confirmed 2026-09-19 |
| A-3 | Oliver supplies the tray, application and installer artwork as a master PNG with a transparent background. | Oliver | Confirmed and met 2026-09-19: `assets/application-icon.png`, 1254x1254 RGBA, all four corners alpha 0 (measured, Appendix A M-8). |

## 3. Requirements

Priorities use MoSCoW. Every requirement names the test that will verify it;
test names are planned, not yet written.

### 3.1 Functional requirements: the day

**FR-001 Day name**
- Priority: Must
- Requirement: The indicator shall display the London day as one of
  `Monday`, `Tuesday`, `Wednesday`, `Thursday`, `Friday`, `Saturday` or
  `Sunday`: English, full word, initial capital.
- Acceptance: Given the instant 2026-09-19T15:44:49+01:00, the indicator reads
  `Saturday`.
- Verified by: `internal/domain/day_test.go::TestDayNameAtInstant`

**FR-002 Europe/London regardless of Windows timezone**
- Priority: Must
- Requirement: The day service shall derive the London day from Europe/London
  civil time whatever timezone Windows is set to.
- Acceptance: Given Windows set to UTC-10:00 and the instant
  2026-09-19T23:30:00+01:00 (London Saturday, Hawaii Saturday 12:30), then at
  2026-09-20T00:30:00+01:00 the indicator reads `Sunday` while Hawaii is still
  on Saturday.
- Verified by: `internal/domain/day_test.go::TestDayIgnoresLocalZone`

**FR-003 Change at London midnight**
- Priority: Must
- Requirement: When London midnight passes, the indicator shall show the new
  day name within 1 s.
- Rationale: "It will always accurately change at midnight."
- Acceptance: Given the scheduler computes the next London midnight after
  2026-03-29T12:00:00+01:00 (the first BST day, 23 hours long), then the
  answer is 2026-03-29T23:00:00Z. Given 2026-10-25T12:00:00Z (the first GMT
  day, 25 hours long), then the answer is 2026-10-26T00:00:00Z.
- Verified by: `internal/domain/midnight_test.go::TestNextMidnightAcrossTransitions`
  plus a manual observation at a real midnight (Appendix B).

**FR-004 Resume from sleep**
- Priority: Must
- Requirement: When the machine resumes from sleep or hibernation, the
  indicator shall show the London day of the resume instant within 1 s.
- Acceptance: Given the machine sleeps on Monday 23:50 London time and resumes
  on Tuesday 07:00, then within 1 s of resume the indicator reads `Tuesday`.
- Verified by: `internal/application/refresh_test.go::TestResumeReevaluates`
  (fake clock plus fake power event) plus a manual sleep test.

**FR-005 Clock or timezone change**
- Priority: Must
- Requirement: When the system clock is changed, the indicator shall show the
  London day of the new instant within 1 s.
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
  which follows next London midnight from 2000-01-01 to 2399-12-31 and asserts
  every step lands on the following civil date at 00:00 London time, through
  every leap day and every GMT/BST change.

### 3.2 Functional requirements: the indicator

**FR-010 No window furniture**
- Priority: Must
- Requirement: The indicator shall have no title bar, frame, caption buttons
  or window menu; its whole visible area is the day name on its background.
- Acceptance: Inspection of the running indicator; its window style is
  `WS_POPUP` with no `WS_CAPTION`, `WS_THICKFRAME` or `WS_SYSMENU`.
- Verified by: `internal/ui/window_test.go::TestIndicatorStyleHasNoFurniture`

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
- Verified by: `TestIndicatorStyleHasNoFurniture` (tool-window style) plus
  inspection.

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
- Requirement: The indicator's background shall be the Acrylic material in the
  current Windows mode.
- Rationale: Measured in probe round 3: Acrylic was judged right by eye;
  Mica, Mica Alt and the two solid colours were near-identical to one another
  and none matched (Appendix A M-4).
- Verified by: inspection.

**FR-016 Follow Windows mode**
- Priority: Must
- Requirement: When the Windows mode changes between light and dark, the
  indicator shall repaint in the new mode within 1 s.
- Verified by: manual; the change arrives as `WM_SETTINGCHANGE` with
  `ImmersiveColorSet`.

**FR-017 Drag to move**
- Priority: Must
- Requirement: When the user presses the left mouse button on the indicator
  and moves beyond the drag threshold, the indicator shall follow the pointer
  until the button is released.
- Verified by: `internal/domain/drag_test.go::TestThresholdSeparatesClickFromDrag`
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
- Requirement: The menu shall contain exactly: `About WhatDay`, a separator,
  then one item per palette colour with the current colour checked.
- Verified by: `internal/application/menu_test.go::TestMenuModel`

**FR-033 Choose a colour**
- Priority: Must
- Requirement: When a colour is chosen from the menu, the indicator shall
  repaint the day name in that colour within 1 s.
- Verified by: `internal/application/colour_test.go::TestChooseColourRepaints`

**FR-034 About**
- Priority: Must
- Requirement: When `About WhatDay` is chosen, WhatDay shall show a dialog
  stating the product name, the version read from `VERSION`, the author and
  the licence (GPL-3.0).
- Verified by: `internal/application/about_test.go::TestAboutContent`

### 3.4 Functional requirements: colours

**FR-040 Palette**
- Priority: Must
- Requirement: The palette shall hold named colours, each with one shade for
  dark mode and one for light mode. The indicator shall paint the shade that
  matches the current Windows mode.
- Rationale: Measured: a single shade cannot serve both modes. Green `#30B050`
  reaches 5.65:1 on dark but 2.54:1 on light; Amber `#F0A010` 7.38:1 on dark
  but 1.94:1 on light (Appendix A M-7).
- Names (Q-2, decided 2026-09-19), in menu order: Red, Amber, Green, Blue,
  Purple, Neutral (white on dark, black on light).
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
- Verified by: `store_test.go::TestUnwritableKeepsSessionValue`

**FR-060 Start at login**
- Priority: Must
- Requirement: The setup program shall register WhatDay to start at login for
  the installing user (`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`).
- Acceptance: After install and a reboot, the indicator is visible without the
  user starting anything.
- Verified by: manual reboot test.

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
- Verified by: `runlog_test.go::TestPanicIsRecorded`

### 3.6 Functional requirements: setup program

**FR-070 Per-user install**
- Priority: Must
- Requirement: The setup program shall install WhatDay to
  `%LOCALAPPDATA%\Programs\WhatDay` without requesting administrator rights.
- Verified by: manual install on the reference machine.

**FR-071 Install, update, repair, uninstall**
- Priority: Must
- Requirement: The setup program shall offer install, update, repair and
  uninstall with a screen for each; it shall register in the Windows Apps
  list with Modify and Repair.
- Verified by: manual; each route exercised once.

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

**FR-074 House style**
- Priority: Must
- Requirement: The setup program shall follow PigeonPost's installer: 126 px
  mark, centred body, progress bar, light and dark themes.
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
contrast against the measured Acrylic background of its mode. The day name is
large text (at least 24 px, which is 18 pt), so 3:1 is the WCAG threshold. Verified
by a test over the palette table against the background constants recorded in
Appendix A.

**NFR-COL-002 Distinct colours**: Within each mode, every pair of palette
shades shall differ by a CIEDE2000 distance of at least 20, so no two choices
look alike. Verified by the same test.

**NFR-PRIV-001 No network**: WhatDay and its setup program shall make no
network connections. Verified by a structural test forbidding `net` and
`net/http` imports outside the setup program's Wails runtime.

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
`VERSION`.

### 3.8 Won't this time

| Item | Reason |
|---|---|
| Update check | Personal tool; no network (NFR-PRIV-001). |
| Exit menu item | "That is all"; uninstall or Task Manager stops it. |
| Background choice | Measured: the alternatives looked alike (M-4). |
| Auto-hide taskbar support | Rests on A-2; the work area does not exclude an auto-hidden taskbar. |
| Taskbar at the top or sides | Rests on A-2. |
| Localisation, other zones | Out of scope (1.3). |

## 4. Other requirements

### 4.1 Legal

GPL-3.0 for the application and the setup program. No third-party assets.

### 4.2 Internationalisation

None by decision: English day names and Europe/London only.

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
and FR-073.

| ID | Question | Plan | Owner | Due |
|---|---|---|---|---|
| Q-5 | What colour is the indicator's own Acrylic background in each mode? M-4 measured the taskbar rather than the indicator; light mode was not measured at all. NFR-COL-001 needs both. | Sample the running indicator's background in dark and light mode over a few different windows behind it; record the range. Choose shades against the worst case. | Claude, with Oliver switching the mode | Before the palette shades are fixed |

### Appendix C: Build order

1. Domain: day name, next London midnight, geometry (size, clamp, default,
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
| 1 | 2026-09-19 | FR-022 | "Lost" is judged by the monitor's whole rectangle, not its work area; a strip still on its monitor is clamped, not moved. | `TestPlaceClampsSavedPositionOverlappingTaskbar` failed against the baseline wording: a strip whose centre sat over the taskbar was sent to the default corner though its monitor was attached. |
