package update

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/oernster/WhatDay/internal/application"
)

// These tests never reach the network: every request goes to a fake client.

type fakeClient struct {
	seen   *http.Request
	status int
	body   io.Reader
	err    error
}

func (f *fakeClient) Do(request *http.Request) (*http.Response, error) {
	f.seen = request
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{StatusCode: f.status, Body: io.NopCloser(f.body)}, nil
}

func answering(status int, body string) *fakeClient {
	return &fakeClient{status: status, body: strings.NewReader(body)}
}

const published = `{
	"tag_name": "v1.2.0",
	"html_url": "https://github.com/oernster/WhatDay/releases/tag/v1.2.0",
	"assets": [
		{"name": "WhatDaySetup.exe", "browser_download_url": "https://github.com/oernster/WhatDay/releases/download/v1.2.0/WhatDaySetup.exe"},
		{"name": "no-address.exe", "browser_download_url": ""},
		{"name": "", "browser_download_url": "https://example.com/nameless"}
	]
}`

func latest(client Doer) (application.ReleaseInfo, error) {
	return NewSourceWith(LatestReleaseURL, client).LatestRelease()
}

func TestAPublishedReleaseIsReadWithItsWholeAssets(t *testing.T) {
	t.Parallel()
	info, err := latest(answering(http.StatusOK, published))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Version != "v1.2.0" || info.PageURL != "https://github.com/oernster/WhatDay/releases/tag/v1.2.0" {
		t.Fatalf("got %+v", info)
	}
	want := application.ReleaseAsset{Name: "WhatDaySetup.exe", DownloadURL: "https://github.com/oernster/WhatDay/releases/download/v1.2.0/WhatDaySetup.exe"}
	if len(info.Assets) != 1 || info.Assets[0] != want {
		t.Fatalf("assets: %+v", info.Assets)
	}
}

// The address is asserted literally, so a slip in it fails here rather than
// checking some other project's releases.
func TestTheRequestAsksWhatDaysLatestReleaseAsJSON(t *testing.T) {
	t.Parallel()
	client := answering(http.StatusOK, published)
	if _, err := latest(client); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := client.seen.URL.String(); got != "https://api.github.com/repos/oernster/WhatDay/releases/latest" {
		t.Errorf("url: %s", got)
	}
	if got := client.seen.Header.Get("Accept"); got != "application/vnd.github+json" {
		t.Errorf("accept: %q", got)
	}
	if client.seen.Method != http.MethodGet {
		t.Errorf("method: %s", client.seen.Method)
	}
}

func TestTheProductionSourceGivesUpAfterFiveSeconds(t *testing.T) {
	t.Parallel()
	source := NewSource()
	client, ok := source.client.(*http.Client)
	if !ok || client.Timeout != requestTimeout || requestTimeout.Seconds() != 5 {
		t.Fatalf("client %+v", source.client)
	}
	if source.url != LatestReleaseURL {
		t.Fatalf("url %q", source.url)
	}
}

func TestEveryFailureIsAnError(t *testing.T) {
	t.Parallel()
	cases := map[string]*fakeClient{
		"no network":      {err: errors.New("no route to host")},
		"not found":       answering(http.StatusNotFound, "{}"),
		"rate limited":    answering(http.StatusForbidden, published),
		"not JSON":        answering(http.StatusOK, "<html>"),
		"no tag":          answering(http.StatusOK, `{"html_url": "https://x"}`),
		"no page":         answering(http.StatusOK, `{"tag_name": "v1.2.0"}`),
		"empty identity":  answering(http.StatusOK, `{"tag_name": "", "html_url": ""}`),
		"body fails":      {status: http.StatusOK, body: failingReader{}},
		"assets not list": answering(http.StatusOK, `{"tag_name": "v1.2.0", "html_url": "https://x", "assets": 3}`),
	}
	for name, client := range cases {
		if info, err := latest(client); err == nil {
			t.Errorf("%s: answered %+v with no error", name, info)
		}
	}
}

func TestAReleaseWithNoAssetsHasNone(t *testing.T) {
	t.Parallel()
	info, err := latest(answering(http.StatusOK, `{"tag_name": "v1.2.0", "html_url": "https://x"}`))
	if err != nil || len(info.Assets) != 0 {
		t.Fatalf("got %+v, %v", info, err)
	}
}

// An answer longer than the cap is cut there, which leaves it unparseable:
// the size of what is read is WhatDay's choice, never the server's.
func TestAnOversizedAnswerIsCutAtTheCap(t *testing.T) {
	t.Parallel()
	padding := strings.Repeat(" ", maxReleaseBytes)
	if _, err := latest(answering(http.StatusOK, padding+published)); err == nil {
		t.Fatal("an answer past the cap was read in full")
	}
}

func TestAnAddressThatIsNotOneIsAnError(t *testing.T) {
	t.Parallel()
	if _, err := NewSourceWith("http://\x7f", answering(http.StatusOK, published)).LatestRelease(); err == nil {
		t.Fatal("expected an error")
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("connection reset") }
