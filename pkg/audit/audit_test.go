package audit_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamnd/bunpy/v1/pkg/audit"
	"github.com/tamnd/bunpy/v1/pkg/lockfile"
)

func makeClient(ts *httptest.Server) *audit.OSVClient {
	c := audit.NewOSVClient()
	c.HTTPClient = ts.Client()
	c.BaseURL = ts.URL
	return c
}

func makeOSVResponse(results []any) string {
	body, _ := json.Marshal(map[string]any{"results": results})
	return string(body)
}

func vulnResult(id, summary, severity string) map[string]any {
	return map[string]any{
		"vulns": []map[string]any{
			{
				"id":      id,
				"summary": summary,
				"database_specific": map[string]any{
					"severity": severity,
				},
			},
		},
	}
}

func emptyResult() map[string]any {
	return map[string]any{"vulns": []any{}}
}

func TestQueryBatchVulnFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := makeOSVResponse([]any{
			vulnResult("GHSA-test-0001-xxxx", "ReDoS in cookie parsing", "HIGH"),
			emptyResult(),
		})
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	}))
	defer ts.Close()

	pins := []lockfile.Package{
		{Name: "requests", Version: "2.28.2"},
		{Name: "flask", Version: "3.0.0"},
	}
	c := makeClient(ts)
	findings, err := c.QueryBatch(pins)
	if err != nil {
		t.Fatalf("QueryBatch: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(findings))
	}
	f := findings[0]
	if f.Package != "requests" {
		t.Errorf("want requests, got %s", f.Package)
	}
	if f.ID != "GHSA-test-0001-xxxx" {
		t.Errorf("want GHSA-test-0001-xxxx, got %s", f.ID)
	}
	if f.Severity != "HIGH" {
		t.Errorf("want HIGH, got %s", f.Severity)
	}
}

func TestQueryBatchClean(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := makeOSVResponse([]any{emptyResult(), emptyResult()})
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	}))
	defer ts.Close()

	pins := []lockfile.Package{
		{Name: "requests", Version: "2.31.0"},
		{Name: "flask", Version: "3.0.3"},
	}
	c := makeClient(ts)
	findings, err := c.QueryBatch(pins)
	if err != nil {
		t.Fatalf("QueryBatch: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("want 0 findings, got %d", len(findings))
	}
}

func TestQueryBatchChunking(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		// Decode request to count queries.
		var req struct {
			Queries []any `json:"queries"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		results := make([]any, len(req.Queries))
		for i := range results {
			results[i] = emptyResult()
		}
		resp := makeOSVResponse(results)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	}))
	defer ts.Close()

	// 1500 pins should split into two requests (1000 + 500).
	pins := make([]lockfile.Package, 1500)
	for i := range pins {
		pins[i] = lockfile.Package{Name: "pkg", Version: "1.0.0"}
	}
	c := makeClient(ts)
	_, err := c.QueryBatch(pins)
	if err != nil {
		t.Fatalf("QueryBatch: %v", err)
	}
	if calls != 2 {
		t.Errorf("want 2 HTTP calls, got %d", calls)
	}
}

func TestFilterByGHSA(t *testing.T) {
	findings := []audit.Finding{
		{ID: "GHSA-test-0001-xxxx", Package: "requests"},
		{ID: "GHSA-test-0002-yyyy", Package: "flask"},
	}
	filtered := audit.Filter(findings, []string{"GHSA-test-0001-xxxx"})
	if len(filtered) != 1 {
		t.Fatalf("want 1, got %d", len(filtered))
	}
	if filtered[0].ID != "GHSA-test-0002-yyyy" {
		t.Errorf("wrong finding remaining: %s", filtered[0].ID)
	}
}

func TestFilterByCVE(t *testing.T) {
	findings := []audit.Finding{
		{ID: "CVE-2024-12345", Package: "urllib3"},
		{ID: "GHSA-test-0003-zzzz", Package: "certifi"},
	}
	// Case-insensitive match.
	filtered := audit.Filter(findings, []string{"cve-2024-12345"})
	if len(filtered) != 1 {
		t.Fatalf("want 1, got %d", len(filtered))
	}
	if filtered[0].ID != "GHSA-test-0003-zzzz" {
		t.Errorf("wrong finding remaining: %s", filtered[0].ID)
	}
}

func TestSeverityParsing(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := makeOSVResponse([]any{
			// CVSS score 9.5 -> CRITICAL
			map[string]any{
				"vulns": []map[string]any{
					{
						"id":      "GHSA-crit",
						"summary": "critical",
						"severity": []map[string]any{
							{"type": "CVSS_V3", "score": "9.5"},
						},
					},
				},
			},
			// CVSS score 3.0 -> LOW
			map[string]any{
				"vulns": []map[string]any{
					{
						"id":      "GHSA-low",
						"summary": "low",
						"severity": []map[string]any{
							{"type": "CVSS_V3", "score": "3.0"},
						},
					},
				},
			},
			// No score -> UNKNOWN
			map[string]any{
				"vulns": []map[string]any{
					{
						"id":      "GHSA-unk",
						"summary": "unknown",
					},
				},
			},
		})
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	}))
	defer ts.Close()

	pins := []lockfile.Package{
		{Name: "a", Version: "1.0"},
		{Name: "b", Version: "1.0"},
		{Name: "c", Version: "1.0"},
	}
	c := makeClient(ts)
	findings, err := c.QueryBatch(pins)
	if err != nil {
		t.Fatalf("QueryBatch: %v", err)
	}
	if len(findings) != 3 {
		t.Fatalf("want 3, got %d", len(findings))
	}
	want := map[string]string{"GHSA-crit": "CRITICAL", "GHSA-low": "LOW", "GHSA-unk": "UNKNOWN"}
	for _, f := range findings {
		if got := f.Severity; got != want[f.ID] {
			t.Errorf("%s: want %s, got %s", f.ID, want[f.ID], got)
		}
	}
	_ = strings.ToLower // imported above
}
