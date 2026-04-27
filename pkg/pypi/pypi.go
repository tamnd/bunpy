// Package pypi is the PEP 691 simple-index client. It fetches a
// project page (JSON), parses every release file, and surfaces
// versions plus hashes for the resolver and wheel installer that
// land in later v0.1.x rungs.
//
// The client is offline-first by design: the only piece that ever
// touches the network is httpkit.RoundTripper, and tests pass a
// fixture transport so CI never reaches the live PyPI index. A
// disk cache is optional; when present, ETag-revalidation turns
// 304 responses into cache hits.
package pypi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/tamnd/bunpy/v1/internal/httpkit"
	"github.com/tamnd/bunpy/v1/pkg/cache"
)

// DefaultBaseURL is PyPI's PEP 691 simple index.
const DefaultBaseURL = "https://pypi.org/simple/"

// Accept header for the JSON variant of the simple index.
const acceptHeader = "application/vnd.pypi.simple.v1+json"

// Client fetches PEP 691 project pages.
type Client struct {
	BaseURL   string
	HTTP      httpkit.RoundTripper
	Cache     *cache.Index
	UserAgent string
}

// New returns a Client with sane defaults.
func New() *Client {
	return &Client{
		BaseURL:   DefaultBaseURL,
		HTTP:      httpkit.Default(4),
		UserAgent: "bunpy/0.2.1",
	}
}

// Project is a parsed PEP 691 project page.
type Project struct {
	Name     string      `json:"name"`
	Files    []File      `json:"files"`
	Versions []string    `json:"versions"`
	Meta     ProjectMeta `json:"meta"`
}

// File is one release artefact: a wheel or sdist (or an unknown
// kind we still record for fidelity).
type File struct {
	Filename       string            `json:"filename"`
	URL            string            `json:"url"`
	Hashes         map[string]string `json:"hashes,omitempty"`
	RequiresPython string            `json:"requires_python,omitempty"`
	Yanked         bool              `json:"yanked,omitempty"`
	YankedReason   string            `json:"yanked_reason,omitempty"`
	Version        string            `json:"version,omitempty"`
	Kind           string            `json:"kind"`
	// CoreMetadataAvailable mirrors PEP 658: the wheel's METADATA
	// is served at <url>.metadata. Both legacy
	// data-dist-info-metadata and current core-metadata spellings
	// surface here.
	CoreMetadataAvailable bool              `json:"core_metadata_available,omitempty"`
	CoreMetadataHashes    map[string]string `json:"core_metadata_hashes,omitempty"`
}

// ProjectMeta is the small `meta` object PEP 691 carries.
type ProjectMeta struct {
	APIVersion string `json:"api_version,omitempty"`
	LastSerial int64  `json:"last_serial,omitempty"`
	ETag       string `json:"etag,omitempty"`
}

// Get fetches name from the index. If a cache is configured and
// has a stored ETag, the request goes out with If-None-Match and
// a 304 turns into a cache hit.
func (c *Client) Get(ctx context.Context, name string) (*Project, error) {
	norm := Normalize(name)
	url := strings.TrimRight(c.baseURL(), "/") + "/" + norm + "/"

	var cachedBody []byte
	var cachedETag string
	if c.Cache != nil {
		if body, etag, ok := c.Cache.Get(norm); ok {
			cachedBody = body
			cachedETag = etag
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", acceptHeader)
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	if cachedETag != "" {
		req.Header.Set("If-None-Match", cachedETag)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		if cachedBody != nil {
			return parseProject(norm, cachedBody, cachedETag)
		}
		return nil, fmt.Errorf("pypi: get %s: %w", url, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified:
		if cachedBody != nil {
			return parseProject(norm, cachedBody, cachedETag)
		}
		return nil, fmt.Errorf("pypi: 304 with no cached body for %s", norm)
	case http.StatusNotFound:
		return nil, &NotFoundError{Name: norm}
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("pypi: %s: %s", url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("pypi: read %s: %w", url, err)
	}
	etag := resp.Header.Get("ETag")
	if c.Cache != nil {
		_ = c.Cache.Put(norm, body, etag)
	}
	return parseProject(norm, body, etag)
}

// FetchMetadata returns the wheel's dist-info METADATA bytes. When
// f.CoreMetadataAvailable is true it goes after <url>.metadata via
// httpkit; otherwise it falls back to fetching the wheel body and
// asking the caller-provided extractor to find METADATA inside the
// archive. Hash verification is best-effort: when the index supplied
// a sha256, a mismatch returns an error.
func (c *Client) FetchMetadata(ctx context.Context, f File, fetchWheel func() ([]byte, error), extract func([]byte) ([]byte, error)) ([]byte, error) {
	if f.CoreMetadataAvailable {
		req, err := http.NewRequestWithContext(ctx, "GET", f.URL+".metadata", nil)
		if err != nil {
			return nil, err
		}
		if c.UserAgent != "" {
			req.Header.Set("User-Agent", c.UserAgent)
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return nil, fmt.Errorf("pypi: get %s.metadata: %w", f.URL, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			return nil, fmt.Errorf("pypi: %s.metadata: %s", f.URL, resp.Status)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("pypi: read %s.metadata: %w", f.URL, err)
		}
		if want := f.CoreMetadataHashes["sha256"]; want != "" {
			if got := sha256Hex(body); got != want {
				return nil, fmt.Errorf("pypi: metadata hash mismatch for %s: got %s want %s", f.Filename, got, want)
			}
		}
		return body, nil
	}
	body, err := fetchWheel()
	if err != nil {
		return nil, err
	}
	return extract(body)
}

// NotFoundError is returned when the index serves a 404 for the
// requested project. Callers can use errors.As to detect it and
// decide whether to surface a typo or a missing private mirror.
type NotFoundError struct{ Name string }

func (e *NotFoundError) Error() string { return "pypi: project not found: " + e.Name }

func (c *Client) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return DefaultBaseURL
}

// rawPage models exactly the fields we read off the wire. Unknown
// fields are silently ignored (PEP 691 grows over time).
type rawPage struct {
	Name  string `json:"name"`
	Files []struct {
		Filename             string            `json:"filename"`
		URL                  string            `json:"url"`
		Hashes               map[string]string `json:"hashes,omitempty"`
		RequiresPython       string            `json:"requires-python,omitempty"`
		Yanked               json.RawMessage   `json:"yanked,omitempty"`
		CoreMetadata         json.RawMessage   `json:"core-metadata,omitempty"`
		DataDistInfoMetadata json.RawMessage   `json:"data-dist-info-metadata,omitempty"`
	} `json:"files"`
	Meta struct {
		APIVersion string `json:"api-version,omitempty"`
		LastSerial int64  `json:"_last-serial,omitempty"`
	} `json:"meta"`
}

func parseProject(name string, body []byte, etag string) (*Project, error) {
	var raw rawPage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("pypi: parse %s: %w", name, err)
	}
	canonical := raw.Name
	if canonical == "" {
		canonical = name
	}
	canonical = Normalize(canonical)
	p := &Project{
		Name: canonical,
		Meta: ProjectMeta{
			APIVersion: raw.Meta.APIVersion,
			LastSerial: raw.Meta.LastSerial,
			ETag:       etag,
		},
	}
	versions := map[string]struct{}{}
	for _, rf := range raw.Files {
		f := File{
			Filename:       rf.Filename,
			URL:            rf.URL,
			Hashes:         rf.Hashes,
			RequiresPython: rf.RequiresPython,
		}
		f.Yanked, f.YankedReason = parseYanked(rf.Yanked)
		raw := rf.CoreMetadata
		if len(raw) == 0 {
			raw = rf.DataDistInfoMetadata
		}
		f.CoreMetadataAvailable, f.CoreMetadataHashes = parseCoreMetadata(raw)
		f.Kind = kindOfFilename(rf.Filename).String()
		f.Version = versionFromFilename(canonical, rf.Filename)
		if f.Version != "" {
			versions[f.Version] = struct{}{}
		}
		p.Files = append(p.Files, f)
	}
	for v := range versions {
		p.Versions = append(p.Versions, v)
	}
	sort.Strings(p.Versions)
	return p, nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func parseCoreMetadata(raw json.RawMessage) (bool, map[string]string) {
	if len(raw) == 0 {
		return false, nil
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		return b, nil
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err == nil {
		return true, m
	}
	return false, nil
}

func parseYanked(raw json.RawMessage) (bool, string) {
	if len(raw) == 0 {
		return false, ""
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		return b, ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s != "", s
	}
	return false, ""
}

// FileKind classifies a release file by extension.
type FileKind int

const (
	FileUnknown FileKind = iota
	FileWheel
	FileSdist
)

func (k FileKind) String() string {
	switch k {
	case FileWheel:
		return "wheel"
	case FileSdist:
		return "sdist"
	default:
		return "unknown"
	}
}

func kindOfFilename(name string) FileKind {
	switch {
	case strings.HasSuffix(name, ".whl"):
		return FileWheel
	case strings.HasSuffix(name, ".tar.gz"), strings.HasSuffix(name, ".zip"):
		return FileSdist
	default:
		return FileUnknown
	}
}

// versionFromFilename extracts a release version from the wheel
// filename per PEP 427 (name-version-...whl) or the sdist
// convention (name-version.tar.gz / .zip). Returns "" when the
// filename does not match the expected shape.
func versionFromFilename(projectName, filename string) string {
	base := filename
	switch {
	case strings.HasSuffix(base, ".whl"):
		base = strings.TrimSuffix(base, ".whl")
		parts := strings.SplitN(base, "-", 3)
		if len(parts) < 2 {
			return ""
		}
		return parts[1]
	case strings.HasSuffix(base, ".tar.gz"):
		base = strings.TrimSuffix(base, ".tar.gz")
	case strings.HasSuffix(base, ".zip"):
		base = strings.TrimSuffix(base, ".zip")
	default:
		return ""
	}
	prefix := normalizeForFilename(projectName) + "-"
	if strings.HasPrefix(strings.ToLower(base), prefix) {
		return base[len(prefix):]
	}
	if i := strings.LastIndex(base, "-"); i > 0 {
		return base[i+1:]
	}
	return ""
}

func normalizeForFilename(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, "-", "_"))
}

// Normalize returns the PEP 503 normalised form of name: lowercase
// with runs of [-_.]+ collapsed into a single dash.
func Normalize(name string) string {
	name = strings.ToLower(name)
	var b strings.Builder
	prevDash := false
	for _, r := range name {
		switch {
		case r == '-' || r == '_' || r == '.':
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		default:
			b.WriteRune(r)
			prevDash = false
		}
	}
	return b.String()
}

// PathFor returns the index-relative path for a project under the
// PEP 691 layout. Useful for fixture authoring.
func PathFor(name string) string {
	return Normalize(name) + "/"
}
