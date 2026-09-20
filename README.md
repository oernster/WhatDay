# <img width="128" height="128" alt="application-icon" src="https://github.com/user-attachments/assets/41075177-9f79-45df-84b3-8f23aa9f66a9" /> WhatDay
What bloody day is it?  Everything tells me the time and date; nothing tells me the day.  This does

> **Commercial licences available.** WhatDay is free and open source under
> GPL-3.0. If those terms do not suit what you are building, such as a
> closed-source product, a commercial licence can be bought from me
> separately. It covers my own code; third-party libraries keep their own
> licences. See
> [commercial licensing](https://ernster.dev/commercial-licensing.html).

---

WhatDay puts the name of the day in a small dark strip just above the Windows
taskbar. `Saturday`. That is the whole product. It changes at midnight, in
whatever timezone Windows is set to.

Site: [oernster.github.io/WhatDay](https://oernster.github.io/WhatDay/)

## Who it is for

- Anyone on Windows whose clock shows `14:32  19/09/2026` and who still has
  to work out what day that is.
- People who work from home, keep odd hours or have just come back from
  leave, for whom the days have stopped having names.

## Who it is not for

- Anyone who wants the date, the time or a week number: Windows already shows
  those and WhatDay does not repeat them.
- Anyone who wants the day in a language other than English, a timezone other
  than the one Windows is set to or a light strip. None of those exist.
- Anyone with the taskbar at the top, at the side or set to auto-hide. WhatDay
  keeps clear of the taskbar by staying inside the space it reserves.

## What it does

- **Shows the day** in English, in full, in a borderless always-on-top strip
  as tall as the taskbar. It has no taskbar button and no Alt+Tab entry.
- **Changes at midnight**, in the timezone Windows is set to now. Take the
  laptop to Sydney and the day follows Windows there within a minute. Where a
  zone's clocks jump forward at midnight, the new day starts at the jump.
- **Catches up** after sleep, after the clock is changed and after the
  timezone changes.
- **Moves where you drag it** and stays there across restarts. It will not sit
  on the taskbar. If its monitor is unplugged, it comes back to the bottom
  right of the main monitor.
- **Gets out of the way** when a fullscreen application opens, as the taskbar
  does, then comes back.
- **Comes in seven colours**, chosen from the tray menu: Red (the default),
  Amber, Yellow, Green, Blue, Purple and Neutral (white).
- **Starts when you sign in**, unless you untick that option in setup.
  `Quit WhatDay` in the tray menu stops it until the next sign-in or until it
  is started from the Start Menu.

The tray menu holds everything: `About WhatDay`, `Support WhatDay (opens your
browser)`, the seven colours and `Quit WhatDay`. Clicking the strip itself does
nothing, on purpose.

## What it does not do

- **No network.** WhatDay makes no connection of any kind: no update check, no
  telemetry, no accounts. A structural test fails the build if any of its code
  imports a network package. The Support entry hands a web address to your
  browser, which is the program that connects.
- **No administrator rights.** Setup installs for your account only.
- **No correcting the clock.** WhatDay trusts the Windows clock; keeping that
  right is Windows' job.

## Install

WhatDay runs on Windows. It is built and tested on Windows 11, where the strip
gets its translucent Acrylic backdrop (22H2 and later).

1. Download `WhatDaySetup.exe` from the
   [Releases page](https://github.com/oernster/WhatDay/releases).
2. Run it and choose Install.

Setup puts WhatDay in `%LOCALAPPDATA%\Programs\WhatDay`, adds a Start Menu
shortcut and lists it under Settings > Apps. `Start WhatDay when I sign in to
Windows` is ticked to begin with; untick it to start WhatDay only when you
choose.

Running setup again offers Update, Go back or Repair, depending on the version
already installed, with Uninstall beside each. The sign-in option shows how it
stands; on the Repair screen a change to it applies at once. Modify in the
Apps list opens the same program.

Uninstalling removes the program, the shortcut, the sign-in entry, the Apps
list entry and everything WhatDay wrote: its settings and its log.

## Where it keeps things

| What | Where |
|---|---|
| The program | `%LOCALAPPDATA%\Programs\WhatDay` |
| The colour and position | `%APPDATA%\WhatDay\settings.json` |
| The log | `%LOCALAPPDATA%\WhatDay\WhatDay.log`, started afresh past 1 MB |
| The setup log | `%TEMP%\WhatDaySetup.log` |

## Stack

| Part | What |
|---|---|
| Language | Go, cgo disabled |
| The application | The Go standard library plus `golang.org/x/sys/windows`: Win32 directly, no UI toolkit |
| Timezone rules | The IANA database embedded in the binary (`time/tzdata`); Windows' own ICU names the zone |
| The setup program | Wails v2, with a hand-written page |
| Tests | Go's `testing`, gofmt, go vet, staticcheck |
| Build | PowerShell: `build.ps1` and `test.ps1` |

## Build and test

```powershell
./build.ps1
```

That stamps the site's version, runs the whole test gate, then builds
`dist-installer/WhatDaySetup.exe`. The gate alone is `./test.ps1`.
[DEVELOPMENT.md](DEVELOPMENT.md) covers the tools and every build option;
[TESTING.md](TESTING.md) covers what the gate checks and why.

## Documents

- [ARCHITECTURE.md](ARCHITECTURE.md): the layers, the rules the tests
  enforce and why the design is the shape it is.
- [DEVELOPMENT.md](DEVELOPMENT.md): tools, building, the icon and the release
  steps.
- [TESTING.md](TESTING.md): the gate, the coverage floors and the checks made
  by hand.
- [REQUIREMENTS.md](REQUIREMENTS.md): the specification, every requirement
  with the test or check that verifies it.

## Supporting the project

The tray menu carries `Support WhatDay (opens your browser)`. It hands a PayPal
page to your browser; WhatDay itself sends nothing, so the no-network promise
above is unchanged by the entry existing.

WhatDay is free and stays free. Donations support maintenance and continued
development. Nothing is withheld behind one: there is no paid tier, no licence
key and no feature a donation unlocks.

<a href="https://www.paypal.com/ncp/payment/7LC63AH9F2UYU"><img src="docs/donate.png" alt="Donate to WhatDay" width="120"></a>

## Licence

Distributed under the GNU General Public License v3.0; see [LICENSE](LICENSE).

A commercial licence for my own code is also available, separately from the
open-source licence: see
[commercial licensing](https://ernster.dev/commercial-licensing.html).

### Open source credits

Compiled into WhatDay, as its About dialog states:

- **Go** and **golang.org/x/sys**: BSD 3-Clause, © 2009 The Go Authors.
- **IANA Time Zone Database**: public domain.

The setup program is built with [Wails](https://wails.io) (MIT).
