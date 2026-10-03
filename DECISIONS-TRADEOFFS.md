# Decisions and trade-offs

The deliberate choices WhatDay rests on: what was chosen, what was given up
for it and why. Each entry is the decision as the product makes it today.
The detail behind each one, with the tests that hold it, lives in
[ARCHITECTURE.md](ARCHITECTURE.md) and the specification
([REQUIREMENTS.md](REQUIREMENTS.md)), whose amendments record every change
from the baseline with its reason.

## The product as a whole

### One answer and nothing else

WhatDay shows the name of the day. It does not show the date, the time or a
week number; Windows already shows those.

- **Rather than:** a clock or calendar widget that also names the day.
- **Gains:** a small surface that can be held to a high bar; nothing on
  screen repeats what the taskbar already says.
- **Costs:** anyone who wants more needs another tool.

### Built for one person

The specification names one user: the owner, on his own machine. Choices
that would serve other people (other languages, a light strip, a taskbar
anywhere but the bottom) are ruled out rather than offered.

- **Rather than:** a general utility with settings for every taste.
- **Gains:** every requirement can be checked on the machine it was written
  for; nothing is built for a user who may not exist.
- **Costs:** some people cannot use it as it stands; the README says so.

### Go on Win32, with no toolkit

The application is pure Go, written directly against the Windows API with
the standard library and Go's own Windows extension. No C compiler is
needed.

- **Rather than:** a UI toolkit. One popup window, one menu, a message box
  and a task dialog need nothing more.
- **Gains:** a small dependency list; a single executable with nothing
  beside it.
- **Costs:** the window, the tray, the appbar and the dialogs are WhatDay's
  own code. Most of it can only be verified by hand.

### Windows only, permanently

WhatDay is a Windows application and will stay one. It is built and tested
on Windows 11; earlier versions are not refused, only untested. Where the
Acrylic backdrop is refused, the refusal is logged and the strip carries on.

- **Rather than:** a cross-platform design; refusing to start on anything
  older than the tested version.
- **Gains:** the strip can lean on the taskbar, the work area and the
  shell's own signals without an abstraction over them.
- **Costs:** no other operating system; behaviour on older Windows is
  whatever it turns out to be.

### Specification before code

Every behaviour is written as a numbered requirement naming the test or the
hand check that verifies it. A change after the baseline arrives as a
numbered amendment with its reason, including the changes made to match
what was built rather than build to the text.

- **Rather than:** building first and describing afterwards.
- **Gains:** a decision reversed (light mode, a fixed timezone, no network
  at all) is recorded with why; a claim in the docs is one a test or a check
  holds.
- **Costs:** the specification is a document of its own to keep true.

## Getting the day right

### The zone Windows is set to, read at every refresh

The day follows whatever timezone Windows is set to now. WhatDay reads the
Windows zone at every refresh, so a laptop that travels shows the new day
within a minute. Day names stay English in every zone.

- **Rather than:** a fixed zone, which was the first design; Go's own idea
  of the local zone, which is read once when the process starts and so
  cannot follow a change.
- **Gains:** the strip always agrees with the Windows clock beside it.
- **Costs:** a refresh reads the zone each time, rather than once.

### Windows names the zone; the rules travel with the program

Windows' own ICU translates the Windows zone into its IANA name. The rules
for that zone come from the timezone database compiled into the binary.
Where daylight saving is switched off in Windows, the zone's standard offset
is kept all year, as the Windows clock does.

- **Rather than:** a translation table of WhatDay's own; the rules the
  machine happens to have installed.
- **Gains:** no table to maintain; the answer never depends on what is
  installed.
- **Costs:** when a country changes its clocks, WhatDay is right again only
  once it is rebuilt with a newer Go toolchain.

### A zone that cannot be read keeps the last good one

If the zone cannot be read or translated, WhatDay keeps the zone it last
read (else the one the process started with) and logs the fault once until
it changes.

- **Rather than:** stopping; falling back to UTC; logging every minute.
- **Gains:** a passing fault never makes the day wrong or the log long.
- **Costs:** a zone change made during the fault is missed until it clears.

### Midnight is when the date moves on

The new day begins at the first instant the local date changes. Usually
that is 00:00; where a zone's clocks jump forward at midnight, it is the
moment of the jump.

- **Rather than:** asking for 00:00 on the next date, which was measured
  wrong in every zone whose clocks skip midnight.
- **Gains:** the day changes at the right instant in every zone, including
  the ones whose midnight never happens.
- **Costs:** a little more code than the obvious version.

### One response to everything that could change the day

The midnight wake-up, resume from sleep, a clock change and a timezone
change all lead to the same refresh: read the zone, paint the day, arm the
next wake-up.

- **Rather than:** a handler of its own for each event.
- **Gains:** one place to be right.
- **Costs:** none recorded.

### Never wait more than a minute

The window's timer fires at the next midnight or one minute from now,
whichever comes first.

- **Rather than:** one timer set for midnight. A Windows timer counts
  elapsed time rather than wall-clock time and pauses while the machine
  sleeps, so a long wait would drift.
- **Gains:** any drift, a missed resume or clock-change message and an
  unannounced zone change are all put right within a minute.
- **Costs:** a refresh every minute; the idle cost has not yet been
  measured.

### The calendar is checked independently

Every date across four centuries is checked against a separate weekday
formula written into the test. Midnight is followed through every zone of
the embedded database.

- **Rather than:** trusting Go's own time package to check itself.
- **Gains:** leap years, century years and clock changes are proved rather
  than assumed.
- **Costs:** the domain tests take several seconds.

### The Windows clock is trusted

WhatDay reads the system clock and never corrects it.

- **Rather than:** a time source of its own.
- **Gains:** the strip never disagrees with the clock beside it.
- **Costs:** a wrong Windows clock gives a wrong day.

## The strip

### Above the taskbar, never on it

The strip sits just above the taskbar. While it is dragged it is kept wholly
inside the work area of its monitor, so it can never overlap the taskbar.

- **Rather than:** drawing on the taskbar, which was measured: the taskbar
  covers a topmost window on it and keeps it covered; a taskbar button,
  abandoned in favour of the strip.
- **Gains:** the problem is removed rather than watched for.
- **Costs:** a taskbar at the top, at the side or set to auto-hide is not
  supported.

### A strip with no furniture

The strip is a borderless always-on-top window with no taskbar button and
no Alt+Tab entry. Clicking it does nothing; dragging moves it.

- **Rather than:** an ordinary window; a click that opens the menu.
- **Gains:** the strip is the day and nothing else; the tray is the one
  place controls live.
- **Costs:** a click on the strip gives no sign that the tray holds the
  controls.

### Always dark Acrylic

The strip's background is the dark Acrylic material, whatever mode Windows
is in. There is no background choice and no light strip.

- **Rather than:** Mica, Mica Alt or a solid colour, tried by eye on the
  reference machine: they looked alike and none matched the taskbar, which
  the wallpaper tints; following the Windows mode, dropped because the
  owner uses dark mode only.
- **Gains:** one shade per colour; nothing to set.
- **Costs:** on a light desktop the strip stays dark.

### As tall as its own taskbar, as wide as the longest day

The strip's height is the taskbar height of the monitor it is on, already in
that monitor's pixels; a monitor with no taskbar borrows the main monitor's,
scaled to match. Its width fits the widest day name, so it never
changes from day to day.

- **Rather than:** one size everywhere; a width that follows the day.
- **Gains:** the strip matches the taskbar on monitors of different scales;
  it never jumps in width at midnight.
- **Costs:** shorter day names sit in spare space.

### Hidden for fullscreen by the taskbar's own signal

The strip registers as an appbar and hides when the shell reports a
fullscreen application, the same signal the taskbar hides on.

- **Rather than:** the notification-state API, which was measured
  reporting busy during a screenshot and hid the strip wrongly.
- **Gains:** the strip hides exactly when the taskbar does.
- **Costs:** none recorded.

### A drag across scales is sized by the drag

While the mouse button is down, a change of scale from crossing onto
another monitor is ignored; the drag sizes the strip itself.

- **Rather than:** laying out at once, which snaps the strip back to its
  saved position mid-drag.
- **Gains:** a drag between monitors of different scales goes where it is
  taken.
- **Costs:** none recorded.

### A lost monitor sends it home

If the strip's centre lies on no attached monitor, it moves to the bottom
right of the main monitor. A strip still on its monitor is clamped into
that monitor's work area instead.

- **Rather than:** judging "lost" by the work area, which would send a
  strip whose centre sat over the taskbar to the default corner though its
  monitor was still attached.
- **Gains:** unplugging a monitor never strands the strip off screen; a
  strip near the taskbar stays where it was put.
- **Costs:** none recorded.

## Colours

### Seven colours, measured rather than picked by eye

The palette holds seven named shades. A test holds every one to the WCAG
contrast minimum for large text and every pair to a minimum perceptual
distance, so no two look alike. The colour formulas are themselves checked
against published reference values. Yellow was chosen from several
candidates as the one furthest from Amber.

- **Rather than:** shades chosen by eye; an open colour picker.
- **Gains:** every choice is readable and no two look alike; a new shade
  that fails either limit fails the suite.
- **Costs:** the strip's own background has not been measured. Shades are
  checked against the taskbar's measured colour and a lighter stand-in until
  it is.

### A colour name it does not know paints in Red

A saved colour that is missing or no longer in the palette falls back to
the default, Red.

- **Rather than:** refusing to start; painting nothing.
- **Gains:** a stale or corrupt setting still shows the day.
- **Costs:** none recorded.

## Privacy and the network

### One connection, held by a structural test

The update check is the only connection WhatDay makes: an anonymous read of
its own latest release on GitHub. A structural test forbids every network
package in the repository except the one the update check uses, in that
package alone. A second test fails once that exemption is no longer needed.
The setup program makes no connection.

- **Rather than:** no network at all, the first design; a promise that the
  network is used sparingly.
- **Gains:** the change was disclosed in the specification rather than
  made silently; a new way out is an edit somebody has to make and defend.
- **Costs:** WhatDay is no longer fully offline.

### Nothing about the user in the request

The request names WhatDay's releases and asks for JSON. Beyond that it
carries only the standard headers Go sends with any request: no version, no
identifier and no telemetry. It gives up quickly and reads only a bounded
amount of the answer.

- **Rather than:** telling the server which version is asking.
- **Gains:** nothing about the user or the machine leaves it; a slow or
  oversized answer cannot hold the check.
- **Costs:** there is no way to know which versions are in use.

### Update checks: daily, quiet unless there is news

An automatic check runs a few seconds after the strip appears, then once a
day, off the window's thread. It speaks only to offer a newer release that
has not been skipped. A check the user asks for ignores the skip and always
answers.

- **Rather than:** no check at all; one that reports every outcome.
- **Gains:** updates are found without nagging; starting up is never slowed
  by the network.
- **Costs:** one unprompted request a day.

### Never claim the latest version without knowing

A version on either side that is not dotted numbers, a source build's
included, reads as unreachable.

- **Rather than:** guessing a comparison.
- **Gains:** WhatDay never tells somebody they are up to date when it
  cannot tell.
- **Costs:** a build from source always reports that GitHub could not be
  reached.

### A prompt with named buttons, with a fallback

The update prompt is Windows' task dialog with Download, Skip This Version
and Later, enabled by a manifest the build embeds. Where the dialog is
unavailable, a Yes, No, Cancel box explains which button does what. A
skipped release is saved by its tag and not offered again.

- **Rather than:** a plain message box alone, whose buttons cannot be
  renamed.
- **Gains:** the choices say what they do.
- **Costs:** the dialog's record is laid out by hand, as measured on the
  reference machine.

### Donations and downloads go through the browser

The Support entry and the update prompt's Download hand an address to the
default browser. Only a secure web address is handed over. A refusal is
logged and shown, so the entry never appears to do nothing.

- **Rather than:** making those requests itself.
- **Gains:** no connection of WhatDay's own is added.
- **Costs:** WhatDay never learns what happened next.

## Running and failing

### The log opens first

The log is opened before anything else. Crash reports are sent to it, so a
crash lands in the file even though a windowed program has no console. A log past a size limit is started afresh
when the next run begins. Without a log folder, lines go to standard error
rather than stopping the program.

- **Rather than:** a log opened once the window is up.
- **Gains:** a windowed program with no console never dies in silence.
- **Costs:** a plain-text file on disk, with an older history lost when it
  is started afresh.

### One copy per session; two beat none

A named lock scoped to the user's session lets one copy run; a second steps
aside and says so in the log. If the check itself fails, WhatDay starts
anyway.

- **Rather than:** refusing to start when the check cannot be made.
- **Gains:** a fault in the check never leaves the user with no strip.
- **Costs:** in that rare case two strips may show.

### Settings saved whole or not at all

The colour, the position and any skipped release are saved to a temporary
file, then one rename replaces the settings. A missing file means defaults;
an unreadable one means defaults and a log line. A save that fails keeps
the choice for the running session.

- **Rather than:** writing the file in place; stopping on a bad file.
- **Gains:** an interrupted save leaves the previous file intact; a bad
  file never stops the strip.
- **Costs:** a choice that cannot be saved is lost at the next start.

## Building and installing

### A setup program ported, not designed afresh

The setup program is the house installer, ported from PigeonPost: a Wails
application with a hand-written page and no front-end build step. Install,
update, going back to an older version, repair and uninstall are routes
decided once from what the machine holds. It is dark only, like the strip.

- **Rather than:** a generic installer; a new design.
- **Gains:** a setup program already proved elsewhere; the install policy
  lives in WhatDay's own tested code, with the page as a facade over it.
- **Costs:** the setup program needs Wails and WebView2, unlike the
  application itself.

### Installed for one user, without administrator rights

Files go under the user's own local folder; the Apps list entry and the
sign-in entry go under the user's own registry.

- **Rather than:** a machine-wide install.
- **Gains:** Windows never asks for administrator rights.
- **Costs:** each account on a machine installs separately.

### Starting at sign-in is an option

Setup offers to start WhatDay at sign-in, ticked on a fresh install and
otherwise showing what the machine holds. On the repair screen a change
applies at once. The entry is written in the plain quoted form Windows
itself uses.

- **Rather than:** always starting at sign-in.
- **Gains:** the user decides; the option never claims a state the machine
  is not in.
- **Costs:** none recorded.

### Setup asks before it closes WhatDay

Before any file is touched, setup checks whether WhatDay is running and asks
to close it. It ends it by the executable's name; the install and uninstall
steps also refuse outright while it runs.

- **Rather than:** replacing files under a running program.
- **Gains:** an install never fails half way over a file in use.
- **Costs:** a running strip is ended without its own shutdown.

### The payload is fenced

Every entry in the embedded archive is checked against the install folder
before any is written.

- **Rather than:** unpacking as it reads.
- **Gains:** a crafted archive cannot write outside the install folder; a
  bad one writes nothing.
- **Costs:** none recorded.

### Uninstall removes everything WhatDay wrote

Uninstall removes the program, the shortcut, the sign-in entry, the Apps
list entry, the settings and the log. The Apps list offers Uninstall and
Modify but no Repair of its own; repairing always goes through setup's own
screen. Setup keeps its own step log in the temporary folder. The program
folder goes last: a hidden shell waits for setup to exit, then removes it.

- **Rather than:** leaving settings behind for a later install; removing the
  program folder after a fixed pause, which fails silently while setup is
  still open on its last screen.
- **Gains:** nothing is left over; the step log survives the folders it
  removes.
- **Costs:** a reinstall starts from the defaults. The wait is bounded at
  30 minutes; setup left open longer than that leaves its folder behind.

### One version, one gate, pinned tools

The version lives in one file and reaches both programs and their file
properties at build time. The build runs the whole test gate first and has
no switch to skip it. The checker and the resource tool run at pinned
versions rather than being installed.

- **Rather than:** a version written in several places; a skippable gate;
  whatever version of each tool is installed.
- **Gains:** the version cannot drift; no release comes from a failing tree;
  a new release of a tool cannot change the build of unchanged code.
- **Costs:** every tool upgrade is a deliberate edit to the scripts.

### A website written for the user

The site explains what WhatDay does and offers the setup program. Its hero
shows a live strip naming today rather than a screenshot. The download
button points at the latest release, so it never needs editing.

- **Rather than:** a captured screenshot; a link per version.
- **Gains:** the site shows the day better than a picture of one; the
  download never goes stale.
- **Costs:** the site's version is stamped from the version file by the
  build.

## Engineering

### Layers with one place where they meet

The code is split into domain, application, infrastructure and interface,
each allowed to depend only inward, with one composition root wiring them.
The domain and application never read the clock or touch the disk; the
instant arrives through a port. Structural tests hold all of it, each
proved by planting a violation.

- **Rather than:** convention alone.
- **Gains:** the rules about the day can be tested with no clock, disk or
  window.
- **Costs:** more packages and more explicit wiring.

### Complete coverage where it means something

The domain and application are held at complete statement coverage. Every
other package is held at the figure it actually reaches, raised when cover
rises.

- **Rather than:** one figure over the whole program; floors set as
  targets.
- **Gains:** a floor fails the moment cover is lost.
- **Costs:** the window, the dialogs and the registry work rely on checks
  by hand.

### Small files

Every source file stays under a line limit. None may sit in the band just
below it either.

- **Rather than:** letting files grow.
- **Gains:** files split at real seams before they are forced to.
- **Costs:** more small files.

### Tests with real parts that never touch the user's copy

Tests use a real temporary folder, a real lock held by a second process, a
child process that really crashes and the real ICU. They never reach the
network, open a browser or a dialog, read the real settings or touch a
running WhatDay.

- **Rather than:** mocks for everything; tests that end the owner's own
  copy.
- **Gains:** a passing test means the real thing works; the suite measures
  the same whether WhatDay is running or not.
- **Costs:** fakes are written by hand; some failures cannot be caused on
  demand and stay untested.
