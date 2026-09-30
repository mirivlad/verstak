// Package updates answers one question: is there a newer Verstak release than
// the one running?
//
// It only looks. Nothing is downloaded or installed; the user gets the release
// page. The check reaches the network, which a local-first application must not
// do behind the user's back, so it runs only when asked — the setting is off by
// default and a manual check is always a click.
package updates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DefaultEndpoint is the GitHub API for the newest published, non-prerelease
// desktop release.
const DefaultEndpoint = "https://api.github.com/repos/mirivlad/verstak/releases/latest"

// Result is what a check found.
type Result struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	URL       string `json:"url"`
	Newer     bool   `json:"newer"`
	CheckedAt string `json:"checkedAt"`
}

// Checker queries a release endpoint.
type Checker struct {
	Client   *http.Client
	Endpoint string
	Now      func() time.Time
}

// Check asks the endpoint for the latest release and compares it with current.
func (c Checker) Check(ctx context.Context, current string) (Result, error) {
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	now := time.Now
	if c.Now != nil {
		now = c.Now
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Result{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "Verstak/"+current)

	response, err := client.Do(request)
	if err != nil {
		return Result{}, fmt.Errorf("release server unreachable: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("release server answered %s", response.Status)
	}

	var release struct {
		TagName    string `json:"tag_name"`
		HTMLURL    string `json:"html_url"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&release); err != nil {
		return Result{}, fmt.Errorf("release answer is not readable: %w", err)
	}
	if release.TagName == "" || release.Draft {
		return Result{}, errors.New("release server returned no published release")
	}
	if !strings.HasPrefix(release.HTMLURL, "https://github.com/") {
		return Result{}, errors.New("release page is not on github.com")
	}

	newer, err := IsNewer(release.TagName, current)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Current:   current,
		Latest:    release.TagName,
		URL:       release.HTMLURL,
		Newer:     newer && !release.Prerelease,
		CheckedAt: now().UTC().Format(time.RFC3339),
	}, nil
}

// IsNewer reports whether latest is a later release than current. A current
// build without a release version ("dev", "", or a bare commit) is treated as
// older than any release, so a developer build is always offered the release.
// A pre-release suffix sorts before the release it precedes.
func IsNewer(latest, current string) (bool, error) {
	l, err := parse(latest)
	if err != nil {
		return false, fmt.Errorf("latest release %q: %w", latest, err)
	}
	c, err := parse(current)
	if err != nil {
		return true, nil
	}
	for i := 0; i < 3; i++ {
		if l.core[i] != c.core[i] {
			return l.core[i] > c.core[i], nil
		}
	}
	switch {
	case l.pre == c.pre:
		return false, nil
	case l.pre == "":
		return true, nil // v1.0.0 is newer than v1.0.0-beta
	case c.pre == "":
		return false, nil
	default:
		return l.pre > c.pre, nil
	}
}

var gitDescribeSuffix = regexp.MustCompile(`^[0-9]+-g[0-9a-f]+$`)

type version struct {
	core [3]int
	pre  string
}

func parse(value string) (version, error) {
	var out version
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	// A dirty or locally built binary still carries its release; the suffix
	// says nothing about which release is newer.
	value = strings.TrimSuffix(value, "-dirty")
	if i := strings.IndexByte(value, '+'); i >= 0 {
		value = value[:i]
	}
	if i := strings.IndexByte(value, '-'); i >= 0 {
		out.pre = value[i+1:]
		value = value[:i]
		// "v0.2.8-5-g1ff1c5a" is git describe for five commits after v0.2.8:
		// later than that release, not a pre-release before it.
		if gitDescribeSuffix.MatchString(out.pre) {
			out.pre = ""
		}
	}
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return out, errors.New("not a MAJOR.MINOR.PATCH version")
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return out, errors.New("not a MAJOR.MINOR.PATCH version")
		}
		out.core[i] = n
	}
	return out, nil
}
