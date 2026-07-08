package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	chroma "github.com/zephyraoss/libchroma/v2"

	"github.com/zephyraoss/chromakopia/internal/catalog"
	"github.com/zephyraoss/chromakopia/internal/catalog/catalogtest"
	"github.com/zephyraoss/chromakopia/internal/dataset"
	"github.com/zephyraoss/chromakopia/internal/fpcalc"
)

const (
	fixtureStride       = 8
	fixtureQBits        = 2
	fixtureSkipInterval = 64
)

var (
	mbidTrack1 = uuid.MustParse(catalogtest.RecordingMBID)
	mbidTrack2 = uuid.MustParse("22222222-2222-4222-8222-222222222222")
)

type fixture struct {
	prefix string
	track1 []uint32
	track2 []uint32
	track3 []uint32
}

func lcgValues(seed uint32, n int) []uint32 {
	values := make([]uint32, n)
	for i := range values {
		seed = seed*1664525 + 1013904223
		values[i] = seed
	}
	return values
}

func sampleForPostingIndex(values []uint32) ([]uint32, []uint8) {
	var hashes []uint32
	var ordinals []uint8
	for i := 0; i < len(values) && i/fixtureStride <= 255; i += fixtureStride {
		hashes = append(hashes, values[i])
		ordinals = append(ordinals, uint8(i/fixtureStride))
	}
	return hashes, ordinals
}

func buildFixture(t *testing.T, dir string) *fixture {
	t.Helper()
	fx := &fixture{
		prefix: filepath.Join(dir, "fixture"),
		track1: lcgValues(1, 800),
		track2: lcgValues(2, 640),
		track3: lcgValues(3, 400),
	}

	ds, err := chroma.NewDataStoreBuilder(fx.prefix+".ckd", chroma.CompressPFOR)
	if err != nil {
		t.Fatal(err)
	}
	pi, err := chroma.NewPostingIndexBuilder(fx.prefix + ".cki")
	if err != nil {
		t.Fatal(err)
	}
	mm, err := chroma.NewMetadataMapBuilder(fx.prefix+".ckm", false)
	if err != nil {
		t.Fatal(err)
	}

	datasetID := uuid.MustParse("dddddddd-dddd-4ddd-8ddd-dddddddddddd")
	ds.SetDatasetID(datasetID)
	pi.SetDatasetID(datasetID)
	mm.SetDatasetID(datasetID)
	pi.SetTuningConfig(chroma.TuningConfig{
		Stride:       fixtureStride,
		QBits:        fixtureQBits,
		SkipInterval: fixtureSkipInterval,
	})

	add := func(id uint32, durationMs uint32, values []uint32, mbid uuid.UUID, trackID uint32) {
		if err := ds.Add(id, durationMs, values); err != nil {
			t.Fatal(err)
		}
		if err := mm.Add(id, mbid, trackID, nil); err != nil {
			t.Fatal(err)
		}
		if mbid != uuid.Nil {
			hashes, ordinals := sampleForPostingIndex(values)
			if err := pi.Add(id, hashes, ordinals); err != nil {
				t.Fatal(err)
			}
		}
	}

	add(1, 100_000, fx.track1, mbidTrack1, 101)
	add(2, 187_600, fx.track2, mbidTrack2, 102)
	add(3, 50_000, fx.track3, uuid.Nil, 0)
	add(4, 100_000, fx.track1, mbidTrack1, 101)

	for name, finish := range map[string]func() error{
		".ckd": ds.Finish, ".cki": pi.Finish, ".ckm": mm.Finish,
	} {
		if err := finish(); err != nil {
			t.Fatalf("finish %s: %v", name, err)
		}
	}
	return fx
}

func openMetadataStore(t *testing.T, dir string) *catalog.Store {
	t.Helper()
	path := filepath.Join(dir, "meta.db")
	catalogtest.BuildDB(t, path, false)
	store, err := catalog.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open fixture metadata db: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func stubFpcalc(t *testing.T, dir string, duration int, values []uint32) string {
	t.Helper()
	out := filepath.Join(dir, "fpcalc-output.txt")
	content := fmt.Sprintf("DURATION=%d\nFINGERPRINT=%s\n", duration, rawString(values))
	if err := os.WriteFile(out, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "fpcalc")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncat "+out+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

func rawString(values []uint32) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = strconv.FormatInt(int64(int32(v)), 10)
	}
	return strings.Join(parts, ",")
}

type matchResponse struct {
	Status  string `json:"status"`
	Error   string `json:"error"`
	Matches []struct {
		ID        string  `json:"id"`
		Score     float64 `json:"score"`
		Recording struct {
			Title    string `json:"title"`
			Artist   string `json:"artist"`
			Album    string `json:"album"`
			Duration int    `json:"duration"`
		} `json:"recording"`
	} `json:"matches"`
}

func postJSON(t *testing.T, url string, body any) (*http.Response, matchResponse) {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var mr matchResponse
	if err := json.NewDecoder(resp.Body).Decode(&mr); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp, mr
}

func newTestServer(t *testing.T, withMeta bool) (*httptest.Server, *fixture) {
	t.Helper()
	dir := t.TempDir()
	fx := buildFixture(t, dir)

	ds, err := dataset.Open(fx.prefix)
	if err != nil {
		t.Fatalf("open fixture dataset: %v", err)
	}
	t.Cleanup(func() { ds.Close() })

	srv := &Server{
		Dataset: ds,
		Fpcalc:  &fpcalc.Runner{Path: stubFpcalc(t, dir, 100, fx.track1)},
		TempDir: dir,
	}
	if withMeta {
		srv.Metadata = openMetadataStore(t, dir)
	}

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, fx
}

func TestIdentifyFingerprint(t *testing.T) {
	ts, fx := newTestServer(t, true)

	t.Run("exact match with metadata join", func(t *testing.T) {
		resp, mr := postJSON(t, ts.URL+"/identify/fingerprint",
			map[string]any{"fingerprint": rawString(fx.track1), "duration": 100})
		if resp.StatusCode != http.StatusOK || mr.Status != "ok" {
			t.Fatalf("status %d / %q (%s)", resp.StatusCode, mr.Status, mr.Error)
		}
		if len(mr.Matches) != 1 {
			t.Fatalf("matches = %d, want 1 (duplicate fingerprints must collapse per MBID): %+v", len(mr.Matches), mr.Matches)
		}
		m := mr.Matches[0]
		if m.ID != mbidTrack1.String() {
			t.Errorf("id = %s, want %s", m.ID, mbidTrack1)
		}
		if m.Score < 0.99 {
			t.Errorf("score = %v, want ~1.0", m.Score)
		}
		if m.Recording.Title != "Test Song" || m.Recording.Artist != "Alpha feat. Beta" ||
			m.Recording.Album != "Test Album" || m.Recording.Duration != 100 {
			t.Errorf("recording = %+v", m.Recording)
		}
	})

	t.Run("offset query still matches", func(t *testing.T) {
		_, mr := postJSON(t, ts.URL+"/identify/fingerprint",
			map[string]any{"fingerprint": rawString(fx.track1[80:])})
		if len(mr.Matches) == 0 || mr.Matches[0].ID != mbidTrack1.String() {
			t.Fatalf("matches = %+v", mr.Matches)
		}
		if mr.Matches[0].Score < 0.99 {
			t.Errorf("score = %v, want ~1.0", mr.Matches[0].Score)
		}
	})

	t.Run("degrades to MBID when recording unknown to metadata db", func(t *testing.T) {
		_, mr := postJSON(t, ts.URL+"/identify/fingerprint",
			map[string]any{"fingerprint": rawString(fx.track2)})
		if len(mr.Matches) != 1 {
			t.Fatalf("matches = %+v", mr.Matches)
		}
		m := mr.Matches[0]
		if m.ID != mbidTrack2.String() {
			t.Errorf("id = %s, want %s", m.ID, mbidTrack2)
		}
		if m.Recording.Title != "Unknown Title" || m.Recording.Artist != "Unknown Artist" {
			t.Errorf("recording = %+v", m.Recording)
		}
		if m.Recording.Duration != 188 {
			t.Errorf("duration = %d, want 188", m.Recording.Duration)
		}
	})

	t.Run("unmapped fingerprint is not reported", func(t *testing.T) {
		_, mr := postJSON(t, ts.URL+"/identify/fingerprint",
			map[string]any{"fingerprint": rawString(fx.track3)})
		if mr.Status != "ok" || len(mr.Matches) != 0 {
			t.Fatalf("status %q matches %+v, want ok with none", mr.Status, mr.Matches)
		}
	})

	t.Run("rejects missing fingerprint", func(t *testing.T) {
		resp, mr := postJSON(t, ts.URL+"/identify/fingerprint", map[string]any{})
		if resp.StatusCode != http.StatusBadRequest || mr.Status != "ERROR" {
			t.Errorf("status %d / %q", resp.StatusCode, mr.Status)
		}
	})
}

func TestIdentifyFile(t *testing.T) {
	ts, _ := newTestServer(t, true)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "song.mp3")
	if err != nil {
		t.Fatal(err)
	}
	fw.Write([]byte("fake audio bytes; the stub fpcalc never reads them"))
	mw.Close()

	resp, err := http.Post(ts.URL+"/identify/file", mw.FormDataContentType(), &buf)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var mr matchResponse
	if err := json.NewDecoder(resp.Body).Decode(&mr); err != nil {
		t.Fatal(err)
	}
	if mr.Status != "ok" || len(mr.Matches) == 0 || mr.Matches[0].ID != mbidTrack1.String() {
		t.Fatalf("response = %+v", mr)
	}

	t.Run("rejects request without file", func(t *testing.T) {
		var empty bytes.Buffer
		mw := multipart.NewWriter(&empty)
		mw.Close()
		resp, err := http.Post(ts.URL+"/identify/file", mw.FormDataContentType(), &empty)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", resp.StatusCode)
		}
	})
}

func TestIdentifyURL(t *testing.T) {
	ts, _ := newTestServer(t, true)

	audio := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte("fake audio bytes"))
	}))
	defer audio.Close()

	resp, mr := postJSON(t, ts.URL+"/identify/url", map[string]any{"url": audio.URL + "/song.mp3"})
	if resp.StatusCode != http.StatusOK || mr.Status != "ok" || len(mr.Matches) == 0 {
		t.Fatalf("status %d, response %+v", resp.StatusCode, mr)
	}
	if mr.Matches[0].ID != mbidTrack1.String() {
		t.Errorf("id = %s, want %s", mr.Matches[0].ID, mbidTrack1)
	}

	t.Run("rejects missing url", func(t *testing.T) {
		resp, mr := postJSON(t, ts.URL+"/identify/url", map[string]any{})
		if resp.StatusCode != http.StatusBadRequest || mr.Status != "ERROR" {
			t.Errorf("status %d / %q", resp.StatusCode, mr.Status)
		}
	})

	t.Run("reports download failures", func(t *testing.T) {
		resp, mr := postJSON(t, ts.URL+"/identify/url", map[string]any{"url": audio.URL + "/missing"})
		if resp.StatusCode != http.StatusBadGateway || mr.Status != "ERROR" {
			t.Errorf("status %d / %q", resp.StatusCode, mr.Status)
		}
	})
}

func TestNoMetadataDB(t *testing.T) {
	ts, fx := newTestServer(t, false)

	_, mr := postJSON(t, ts.URL+"/identify/fingerprint",
		map[string]any{"fingerprint": rawString(fx.track1)})
	if len(mr.Matches) != 1 {
		t.Fatalf("matches = %+v", mr.Matches)
	}
	m := mr.Matches[0]
	if m.ID != mbidTrack1.String() {
		t.Errorf("id = %s, want %s", m.ID, mbidTrack1)
	}
	if m.Recording.Title != "Unknown Title" || m.Recording.Artist != "Unknown Artist" {
		t.Errorf("recording = %+v, want unknown placeholders", m.Recording)
	}
	if m.Recording.Duration != 100 {
		t.Errorf("duration = %d, want 100 (from .ckd)", m.Recording.Duration)
	}
}

func TestHealth(t *testing.T) {
	ts, _ := newTestServer(t, true)
	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body struct {
		Status     string `json:"status"`
		Records    uint64 `json:"records"`
		MetadataDB string `json:"metadataDb"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "ok" || body.Records != 4 || body.MetadataDB != "ok" {
		t.Errorf("health = %+v", body)
	}
}
