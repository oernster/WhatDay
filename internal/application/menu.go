package application

import "github.com/oernster/WhatDay/internal/domain"

// MenuKind says what a menu entry is.
type MenuKind int

// The kinds of menu entry (FR-032).
const (
	MenuAbout MenuKind = iota
	MenuSeparator
	MenuColour
	MenuQuit
	MenuDonate
)

// MenuItem is one entry of the tray menu. For a colour entry, Label is the
// colour's name and Checked marks the colour in use.
type MenuItem struct {
	Kind    MenuKind
	Label   string
	Checked bool
}

// Menu wording.
const (
	AboutLabel = "About " + ProductName
	QuitLabel  = "Quit " + ProductName
	// DonateLabel says the entry leaves WhatDay: a menu has no tooltip, so
	// the label itself carries it.
	DonateLabel = "Support " + ProductName + " (opens your browser)"
)

// Menu answers the whole tray menu (FR-032, Amendments 5 and 6): About, the
// Support entry, a separator, every palette colour in order with the current
// one checked, another separator, then Quit.
func (ind *Indicator) Menu() []MenuItem {
	items := []MenuItem{{Kind: MenuAbout, Label: AboutLabel}, {Kind: MenuDonate, Label: DonateLabel}, {Kind: MenuSeparator}}
	for _, c := range domain.Palette() {
		items = append(items, MenuItem{Kind: MenuColour, Label: c.Name, Checked: c.Name == ind.settings.ColourName})
	}
	return append(items, MenuItem{Kind: MenuSeparator}, MenuItem{Kind: MenuQuit, Label: QuitLabel})
}
