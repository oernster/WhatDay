package application

import "github.com/oernster/WhatDay/internal/domain"

// MenuKind says what a menu entry is.
type MenuKind int

// The kinds of menu entry (FR-032).
const (
	MenuAbout MenuKind = iota
	MenuSeparator
	MenuColour
)

// MenuItem is one entry of the tray menu. For a colour entry, Label is the
// colour's name and Checked marks the colour in use.
type MenuItem struct {
	Kind    MenuKind
	Label   string
	Checked bool
}

// AboutLabel is the menu wording for the About entry.
const AboutLabel = "About " + ProductName

// Menu answers the whole tray menu (FR-032): About, a separator, then every
// palette colour in order with the current one checked.
func (ind *Indicator) Menu() []MenuItem {
	items := []MenuItem{{Kind: MenuAbout, Label: AboutLabel}, {Kind: MenuSeparator}}
	for _, c := range domain.Palette() {
		items = append(items, MenuItem{Kind: MenuColour, Label: c.Name, Checked: c.Name == ind.settings.ColourName})
	}
	return items
}
