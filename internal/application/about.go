package application

import (
	"fmt"
	"strings"
)

// Product facts, each with this one home.
const (
	ProductName   = "WhatDay"
	Author        = "Oliver Ernster"
	CopyrightYear = 2026
	// DonateURL is where the Support entry sends a browser (FR-036). It is the
	// only address WhatDay knows. It is handed to the desktop, which opens the
	// page, so WhatDay itself never opens a connection (NFR-PRIV-001).
	DonateURL = "https://www.paypal.com/ncp/payment/7LC63AH9F2UYU"
)

// Credit names one open-source work WhatDay is built from.
type Credit struct {
	Work, Licence, Holder string
}

// credits are the works compiled into WhatDay, each licence read from the
// work's own LICENSE file (Go and golang.org/x/sys) or its README (the time
// zone data Go embeds).
var credits = [...]Credit{
	{Work: "Go", Licence: "BSD 3-Clause", Holder: "© 2009 The Go Authors"},
	{Work: "golang.org/x/sys", Licence: "BSD 3-Clause", Holder: "© 2009 The Go Authors"},
	{Work: "IANA Time Zone Database", Licence: "public domain"},
}

// About is what the About dialog states (FR-034, Amendment 3): the author's
// copyright and the open-source works used, nothing more.
type About struct {
	Title     string
	Copyright string
	Credits   []Credit
}

// NewAbout answers the About facts.
func NewAbout() About {
	return About{
		Title:     AboutLabel,
		Copyright: fmt.Sprintf("© %d %s", CopyrightYear, Author),
		Credits:   append([]Credit(nil), credits[:]...),
	}
}

// Text answers the dialog's body, one line per fact.
func (a About) Text() string {
	lines := []string{ProductName, a.Copyright, "", "Built with:"}
	for _, c := range a.Credits {
		line := c.Work + ", " + c.Licence
		if c.Holder != "" {
			line += ", " + c.Holder
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
