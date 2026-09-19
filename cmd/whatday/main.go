// Command whatday tells you the day of the week. That is all.
//
// This is the composition root: the one place that builds the adapters and
// wires them to the application (C-3).
package main

import (
	"os"
	"time"

	"github.com/oernster/WhatDay/internal/application"
	"github.com/oernster/WhatDay/internal/infrastructure/clock"
	"github.com/oernster/WhatDay/internal/infrastructure/instance"
	"github.com/oernster/WhatDay/internal/infrastructure/runlog"
	"github.com/oernster/WhatDay/internal/infrastructure/settings"
	"github.com/oernster/WhatDay/internal/infrastructure/zone"
	"github.com/oernster/WhatDay/internal/ui"
)

// appVersion is set from VERSION by build.ps1 through -ldflags; it is a var
// because -X cannot reach a const.
var appVersion = "0.0.0-dev"

func main() {
	log := openLog()
	log.Printf("version %s", appVersion)

	lock, held, err := instance.Acquire(instance.Name)
	switch {
	case err != nil:
		// Better two copies than none: carry on and say so.
		log.Printf("single-instance check failed, starting anyway: %v", err)
	case !held:
		log.Printf("already running; this copy exits")
		return
	default:
		defer func() { _ = lock.Release() }()
	}

	window := ui.NewWindow(log, time.Now)
	indicator := application.NewIndicator(clock.System{}, zone.NewWindows(time.Local), window, window, settings.NewStore(), log)
	if err := window.Run(indicator); err != nil {
		log.Printf("cannot open the indicator: %v", err)
		os.Exit(1)
	}
}

// openLog opens the run log and points crash reports at it before anything
// else runs (FR-062, FR-063). Without a log, lines go to standard error,
// which is all that is left.
func openLog() application.Log {
	path, err := runlog.Path()
	if err == nil {
		var file *os.File
		if file, err = runlog.Open(path, time.Now()); err == nil {
			logger := runlog.NewLogger(file, time.Now)
			if keepErr := runlog.Keep(file); keepErr != nil {
				logger.Printf("crash reports will not reach the log: %v", keepErr)
			}
			return logger
		}
	}
	logger := runlog.NewLogger(os.Stderr, time.Now)
	logger.Printf("no log file: %v", err)
	return logger
}
