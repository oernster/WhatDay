// Package update answers WhatDay's latest published release from GitHub
// (FR-037, FR-038). It is the one package in WhatDay that reaches the
// network. It makes one kind of request: an anonymous read of the public
// releases list, sending nothing about the user (NFR-PRIV-001, Amendment 11).
// Ported from PigeonPost's update source.
//
// GitHub's latest-release endpoint only ever answers a published release that
// is neither a draft nor a prerelease, so a tag pushed mid-development is
// structurally invisible: the guard is the endpoint's own contract.
package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/oernster/WhatDay/internal/application"
)

// LatestReleaseURL is GitHub's latest-release endpoint for WhatDay.
const LatestReleaseURL = "https://api.github.com/repos/oernster/WhatDay/releases/latest"

// acceptHeader asks the GitHub API for its JSON.
const acceptHeader = "application/vnd.github+json"

// requestTimeout bounds the one request: a check must never hang.
const requestTimeout = 5 * time.Second

// maxReleaseBytes caps what is read of an answer. A release description runs
// to a few kilobytes; nothing from outside decides how much is read.
const maxReleaseBytes = 1 << 20

// Doer sends one request; *http.Client is one and tests inject a fake.
type Doer interface {
	Do(request *http.Request) (*http.Response, error)
}

// releasePayload is the part of GitHub's answer the check reads.
type releasePayload struct {
	TagName string         `json:"tag_name"`
	HTMLURL string         `json:"html_url"`
	Assets  []assetPayload `json:"assets"`
}

type assetPayload struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

// errIncomplete says an answer lacked the tag or the page.
var errIncomplete = errors.New("the release names no tag or no page")

// Source is an application.ReleaseSource backed by GitHub.
type Source struct {
	url    string
	client Doer
}

// NewSource asks WhatDay's own releases, giving up after requestTimeout.
func NewSource() *Source {
	return NewSourceWith(LatestReleaseURL, &http.Client{Timeout: requestTimeout})
}

// NewSourceWith asks url through client.
func NewSourceWith(url string, client Doer) *Source {
	return &Source{url: url, client: client}
}

// LatestRelease answers the latest published release; failing that, an error
// saying why it could not be read. Assets lacking a name or an address are dropped.
func (s *Source) LatestRelease() (application.ReleaseInfo, error) {
	request, err := http.NewRequest(http.MethodGet, s.url, nil)
	if err != nil {
		return application.ReleaseInfo{}, fmt.Errorf("building the release request: %w", err)
	}
	request.Header.Set("Accept", acceptHeader)
	response, err := s.client.Do(request)
	if err != nil {
		return application.ReleaseInfo{}, fmt.Errorf("asking GitHub for the latest release: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return application.ReleaseInfo{}, fmt.Errorf("GitHub answered status %d", response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxReleaseBytes))
	if err != nil {
		return application.ReleaseInfo{}, fmt.Errorf("reading the release: %w", err)
	}
	var payload releasePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return application.ReleaseInfo{}, fmt.Errorf("parsing the release: %w", err)
	}
	if payload.TagName == "" || payload.HTMLURL == "" {
		return application.ReleaseInfo{}, errIncomplete
	}
	info := application.ReleaseInfo{Version: payload.TagName, PageURL: payload.HTMLURL}
	for _, asset := range payload.Assets {
		if asset.Name != "" && asset.DownloadURL != "" {
			info.Assets = append(info.Assets, application.ReleaseAsset{Name: asset.Name, DownloadURL: asset.DownloadURL})
		}
	}
	return info, nil
}
