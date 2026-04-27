// Package audit queries the OSV (Open Source Vulnerabilities) database
// for known vulnerabilities in a set of pinned packages.
package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/tamnd/bunpy/v1/pkg/lockfile"
)

const (
	defaultBaseURL = "https://api.osv.dev/v1"
	batchSize      = 1000
)

// cvssScore is one entry in the OSV severity array.
type cvssScore struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

// Finding describes one vulnerability against one pinned package.
type Finding struct {
	Package  string
	Version  string
	ID       string  // GHSA- or CVE- identifier
	Summary  string
	Severity string  // CRITICAL / HIGH / MEDIUM / LOW / UNKNOWN
	URL      string  // https://osv.dev/vulnerability/<ID>
}

// OSVClient calls the OSV querybatch API.
type OSVClient struct {
	HTTPClient *http.Client
	BaseURL    string
}

// NewOSVClient returns a client with default settings.
func NewOSVClient() *OSVClient {
	return &OSVClient{
		HTTPClient: &http.Client{},
		BaseURL:    defaultBaseURL,
	}
}

// QueryBatch sends all pins to OSV in chunks of 1000 and returns all
// findings. Order is not guaranteed.
func (c *OSVClient) QueryBatch(pins []lockfile.Package) ([]Finding, error) {
	if len(pins) == 0 {
		return nil, nil
	}
	var all []Finding
	for start := 0; start < len(pins); start += batchSize {
		end := min(start+batchSize, len(pins))
		chunk := pins[start:end]
		found, err := c.queryChunk(chunk)
		if err != nil {
			return nil, err
		}
		all = append(all, found...)
	}
	return all, nil
}

func (c *OSVClient) queryChunk(pins []lockfile.Package) ([]Finding, error) {
	type pkgSpec struct {
		Name      string `json:"name"`
		Ecosystem string `json:"ecosystem"`
	}
	type query struct {
		Version string  `json:"version"`
		Package pkgSpec `json:"package"`
	}
	type request struct {
		Queries []query `json:"queries"`
	}
	qs := make([]query, len(pins))
	for i, p := range pins {
		qs[i] = query{
			Version: p.Version,
			Package: pkgSpec{Name: p.Name, Ecosystem: "PyPI"},
		}
	}
	body, err := json.Marshal(request{Queries: qs})
	if err != nil {
		return nil, fmt.Errorf("audit: marshal request: %w", err)
	}
	url := c.BaseURL + "/querybatch"
	resp, err := c.HTTPClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("audit: post %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("audit: OSV returned %s", resp.Status)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("audit: read response: %w", err)
	}

	type dbSpecific struct {
		Severity string `json:"severity"`
	}
	type vuln struct {
		ID         string      `json:"id"`
		Summary    string      `json:"summary"`
		Severity   []cvssScore `json:"severity"`
		DBSpecific dbSpecific  `json:"database_specific"`
	}
	type result struct {
		Vulns []vuln `json:"vulns"`
	}
	type response struct {
		Results []result `json:"results"`
	}
	var parsed response
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("audit: parse response: %w", err)
	}
	var findings []Finding
	for i, r := range parsed.Results {
		if i >= len(pins) {
			break
		}
		p := pins[i]
		for _, v := range r.Vulns {
			sev := parseSeverity(v.DBSpecific.Severity, v.Severity)
			findings = append(findings, Finding{
				Package:  p.Name,
				Version:  p.Version,
				ID:       v.ID,
				Summary:  v.Summary,
				Severity: sev,
				URL:      "https://osv.dev/vulnerability/" + v.ID,
			})
		}
	}
	return findings, nil
}

func parseSeverity(dbSev string, cvss []cvssScore) string {
	if dbSev != "" {
		upper := strings.ToUpper(dbSev)
		switch upper {
		case "CRITICAL", "HIGH", "MEDIUM", "LOW":
			return upper
		}
	}
	for _, s := range cvss {
		score := parseScore(s.Score)
		switch {
		case score >= 9.0:
			return "CRITICAL"
		case score >= 7.0:
			return "HIGH"
		case score >= 4.0:
			return "MEDIUM"
		case score >= 0.1:
			return "LOW"
		}
	}
	return "UNKNOWN"
}

func parseScore(s string) float64 {
	// CVSS score is a decimal like "9.8" or "CVSS:3.1/AV:N/.../7.5".
	// Extract the last numeric token after the final slash or space.
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '/' || r == ' '
	})
	if len(parts) == 0 {
		return 0
	}
	last := parts[len(parts)-1]
	var f float64
	fmt.Sscanf(last, "%f", &f)
	return f
}

// Filter removes findings whose ID appears in the ignore list.
// Comparison is case-insensitive.
func Filter(findings []Finding, ignore []string) []Finding {
	if len(ignore) == 0 {
		return findings
	}
	lc := make([]string, len(ignore))
	for i, id := range ignore {
		lc[i] = strings.ToLower(id)
	}
	out := findings[:0:0]
	for _, f := range findings {
		if !containsCI(lc, strings.ToLower(f.ID)) {
			out = append(out, f)
		}
	}
	return out
}

func containsCI(haystack []string, needle string) bool {
	return slices.Contains(haystack, needle)
}

// SortFindings sorts findings by severity (CRITICAL first) then by
// package name for stable output.
func SortFindings(findings []Finding) {
	order := map[string]int{
		"CRITICAL": 0,
		"HIGH":     1,
		"MEDIUM":   2,
		"LOW":      3,
		"UNKNOWN":  4,
	}
	sort.SliceStable(findings, func(i, j int) bool {
		oi := order[findings[i].Severity]
		oj := order[findings[j].Severity]
		if oi != oj {
			return oi < oj
		}
		return findings[i].Package < findings[j].Package
	})
}
