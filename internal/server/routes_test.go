package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zephyraoss/chromakopia/internal/catalog"
	"github.com/zephyraoss/chromakopia/internal/catalog/catalogtest"
	"github.com/zephyraoss/chromakopia/internal/dataset"
	"github.com/zephyraoss/chromakopia/internal/fpcalc"
)

func newIdentifyServer(t *testing.T, dir string) (*Server, *fixture) {
	t.Helper()
	fx := buildFixture(t, dir)
	ds, err := dataset.Open(fx.prefix)
	if err != nil {
		t.Fatalf("open fixture dataset: %v", err)
	}
	t.Cleanup(func() { ds.Close() })
	return &Server{
		Dataset: ds,
		Fpcalc:  &fpcalc.Runner{Path: stubFpcalc(t, dir, 100, fx.track1)},
		TempDir: dir,
	}, fx
}

func TestIdentifyRemoteMetadata(t *testing.T) {
	dir := t.TempDir()
	srv, fx := newIdentifyServer(t, dir)

	store := openMetadataStore(t, dir)
	metaTS := httptest.NewServer(Routes(nil, store))
	defer metaTS.Close()
	srv.Metadata = catalog.NewClient(metaTS.URL)

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	_, mr := postJSON(t, ts.URL+"/identify/fingerprint",
		map[string]any{"fingerprint": rawString(fx.track1), "duration": 100})
	if len(mr.Matches) != 1 {
		t.Fatalf("matches = %+v", mr.Matches)
	}
	rec := mr.Matches[0].Recording
	if rec.Title != "Test Song" || rec.Artist != "Alpha feat. Beta" ||
		rec.Album != "Test Album" || rec.Duration != 100 {
		t.Errorf("recording = %+v, want remote-joined metadata", rec)
	}

	t.Run("unknown recording degrades to MBID", func(t *testing.T) {
		_, mr := postJSON(t, ts.URL+"/identify/fingerprint",
			map[string]any{"fingerprint": rawString(fx.track2)})
		if len(mr.Matches) != 1 || mr.Matches[0].Recording.Title != "Unknown Title" {
			t.Errorf("matches = %+v, want placeholder metadata", mr.Matches)
		}
	})

	t.Run("unreachable remote degrades to MBID", func(t *testing.T) {
		metaTS.Close()
		resp, mr := postJSON(t, ts.URL+"/identify/fingerprint",
			map[string]any{"fingerprint": rawString(fx.track1)})
		if resp.StatusCode != http.StatusOK || mr.Status != "ok" {
			t.Fatalf("status %d / %q, want identification to survive metadata outage", resp.StatusCode, mr.Status)
		}
		if len(mr.Matches) != 1 || mr.Matches[0].ID != mbidTrack1.String() {
			t.Fatalf("matches = %+v", mr.Matches)
		}
		if mr.Matches[0].Recording.Title != "Unknown Title" {
			t.Errorf("recording = %+v, want placeholder metadata", mr.Matches[0].Recording)
		}
	})
}

func TestRoutesModes(t *testing.T) {
	dir := t.TempDir()
	srv, _ := newIdentifyServer(t, dir)
	store := openMetadataStore(t, dir)
	srv.Metadata = store

	status := func(t *testing.T, ts *httptest.Server, method, path string) int {
		t.Helper()
		req, err := http.NewRequest(method, ts.URL+path, strings.NewReader("{}"))
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	health := func(t *testing.T, ts *httptest.Server) map[string]any {
		t.Helper()
		resp, err := http.Get(ts.URL + "/health")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var body map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		return body
	}
	catalogPath := "/catalog/recording/" + catalogtest.RecordingMBID

	t.Run("identify only", func(t *testing.T) {
		ts := httptest.NewServer(Routes(srv, nil))
		defer ts.Close()
		if got := status(t, ts, http.MethodGet, catalogPath); got != http.StatusNotFound {
			t.Errorf("catalog status = %d, want 404 (not registered)", got)
		}
		if got := status(t, ts, http.MethodPost, "/identify/fingerprint"); got != http.StatusBadRequest {
			t.Errorf("identify status = %d, want 400", got)
		}
		body := health(t, ts)
		if modes, _ := body["modes"].([]any); len(modes) != 1 || modes[0] != "identify" {
			t.Errorf("health modes = %v", body["modes"])
		}
		if body["metadataDb"] != "ok" || body["catalog"] != nil {
			t.Errorf("health = %v", body)
		}
	})

	t.Run("metadata only", func(t *testing.T) {
		ts := httptest.NewServer(Routes(nil, store))
		defer ts.Close()
		if got := status(t, ts, http.MethodGet, catalogPath); got != http.StatusOK {
			t.Errorf("catalog status = %d, want 200", got)
		}
		if got := status(t, ts, http.MethodPost, "/identify/fingerprint"); got != http.StatusNotFound {
			t.Errorf("identify status = %d, want 404 (not registered)", got)
		}
		body := health(t, ts)
		if modes, _ := body["modes"].([]any); len(modes) != 1 || modes[0] != "metadata" {
			t.Errorf("health modes = %v", body["modes"])
		}
		if body["catalog"] != "ok" || body["records"] != nil {
			t.Errorf("health = %v", body)
		}
	})

	t.Run("both", func(t *testing.T) {
		ts := httptest.NewServer(Routes(srv, store))
		defer ts.Close()
		if got := status(t, ts, http.MethodGet, catalogPath); got != http.StatusOK {
			t.Errorf("catalog status = %d, want 200", got)
		}
		if got := status(t, ts, http.MethodPost, "/identify/fingerprint"); got != http.StatusBadRequest {
			t.Errorf("identify status = %d, want 400", got)
		}
		body := health(t, ts)
		if modes, _ := body["modes"].([]any); len(modes) != 2 {
			t.Errorf("health modes = %v", body["modes"])
		}
		if body["catalog"] != "ok" || body["metadataDb"] != "ok" || body["records"] == nil {
			t.Errorf("health = %v", body)
		}
	})
}
