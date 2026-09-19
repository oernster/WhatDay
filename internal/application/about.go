package application

// Product facts, each with this one home.
const (
	ProductName = "WhatDay"
	Author      = "Oliver Ernster"
	Licence     = "GPL-3.0"
)

// About is what the About dialog states (FR-034).
type About struct {
	Name, Version, Author, Licence string
}

// NewAbout answers the About facts for version, which the composition root
// reads from the build (VERSION, C-7).
func NewAbout(version string) About {
	return About{Name: ProductName, Version: version, Author: Author, Licence: Licence}
}
