# Development

How to build WhatDay from source, what each script does and how a release is
cut. Every command is PowerShell, run from the repository root.

## Tools

| Tool | Needed for | Get it |
|---|---|---|
| Go, at the version `go.mod` names or later | everything | [go.dev/dl](https://go.dev/dl/) |
| Wails CLI v2.12.0 | the setup program | `go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0` |
| WebView2 runtime | running the setup program | ships with Windows 11 |
| Python 3 | stamping the site's version, which `build.ps1` does first | [python.org](https://www.python.org/downloads/) |
| Pillow | regenerating the icon only | `python -m pip install pillow` |

staticcheck and goversioninfo are not installed: `test.ps1` and `build.ps1`
run each at a pinned version through `go run`, so a new release of either
cannot change the result of building unchanged code. The pins live in those
two scripts.

cgo is not used. No C compiler is needed.

## Building

```powershell
./build.ps1
```

In order, it:

1. Runs `stamp_version.py`, so the site shows the version being built.
2. Runs [`test.ps1`](TESTING.md). A failure stops the build; there is no
   switch to skip it.
3. Reads `VERSION`, which must be `major.minor.patch`.
4. Writes `cmd/whatday/resource.syso` with goversioninfo: the icon plus the
   executable's properties (description, version, copyright).
5. Builds `build/bin/WhatDay.exe` with `-H windowsgui`, stripped and with the
   version passed in through `-ldflags`.
6. Zips it into `installer/payload.zip`.
7. Writes `installer/build/windows/icon.ico` and `info.json` (the setup
   program's icon and properties) from the same values.
8. Runs `wails build` in `installer/`.
9. Copies the result to `dist-installer/WhatDaySetup.exe`, the one file that
   ships.
10. Puts the 22-byte empty zip back in `installer/payload.zip`, whether or not
   the build succeeded, so `go build ./...` keeps working without a full
   build.

To build only the application:

```powershell
./build.ps1 -SkipInstaller
```

Everything generated is ignored by git: `build/`, `installer/build/`,
`dist-installer/`, `cmd/whatday/resource.syso` and
`installer/frontend/wailsjs/`.

### Reading the executable's properties

PowerShell's `(Get-Item <exe>).VersionInfo` reads the setup program's
properties as blank. They are there: Explorer's Properties dialog and the
Win32 version API both read them. Wails records them under the neutral
language (`0000`); .NET's reader comes back empty for every Wails build
checked, PigeonPost's included. Check the setup program in Explorer.

## Running from source

```powershell
go run ./cmd/whatday
```

Only one copy runs per user session. If the installed WhatDay is running, the
new one writes `already running; this copy exits` to the log and stops, so
choose `Quit WhatDay` from the tray first.

Everything WhatDay does is written to `%LOCALAPPDATA%\WhatDay\WhatDay.log`:
the version, the timezone, the day it shows, each layout, each drag, each
resume and clock change. Read it first when something looks wrong.

## The icon

`assets/application-icon.png` is the master. From it,
`tools/genicons.py` makes:

- `assets/application-icon.ico`, at 16, 24, 32, 48, 64, 128 and 256 pixels,
  which `build.ps1` puts on both executables. The tray icon, the Start Menu
  shortcut and the Apps list all take theirs from the executable.
- `installer/frontend/dist/icon.png`, the 256-pixel mark in the setup
  window's header.

```powershell
python tools/genicons.py
```

It is not part of the build. Both outputs are committed, so a clone builds
without Pillow. Run it when the master changes, then commit the results.

`docs/donate.png` is the donation artwork shared by every project site; it is
copied, not generated.

## Versioning

`VERSION` holds the only version string. Change it there and nowhere else.
The build carries it into both programs and both executables' properties.
Nothing in the source or the documents holds a version; the site shows the
version through a stamped token (below).

## The site

`docs/` is the GitHub Pages site, published from the `main` branch's `/docs`
folder. It is plain HTML and CSS with no build step.

The version on the site sits between `<!--VERSION-->` and `<!--/VERSION-->`.
`build.ps1` stamps it; to stamp without building:

```powershell
python stamp_version.py
```

It rewrites every token under `docs/` from `VERSION`, prints the files it
changed and changes nothing on a second run.

To preview locally, serve the folder:

```powershell
python -m http.server 8000 --directory docs
```

## Cutting a release

1. Set `VERSION`.
2. Run `./build.ps1`. It stamps the site and will not build from a failing
   tree.
3. Commit, tag `v<version>` and push.
4. Create a GitHub release for the tag and attach
   `dist-installer/WhatDaySetup.exe`.

The README's install instructions point at the Releases page, so step 4 is
what users see.

## Standing rules

- Domain and application stay at 100% coverage; the gate fails otherwise.
- No Go file over 400 lines; none between 381 and 399.
- No `net` import anywhere.
- Every exported type has a doc comment.
- No version string outside `VERSION`.

The reasons are in [ARCHITECTURE.md](ARCHITECTURE.md); the checks are in
[TESTING.md](TESTING.md).
