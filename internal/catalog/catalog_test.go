package catalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/zephyraoss/chromakopia/internal/catalog/catalogtest"
)

func newCatalogServer(t *testing.T, withFTS bool) (*httptest.Server, *Store) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "meta.db")
	catalogtest.BuildDB(t, path, withFTS)
	store, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open fixture db: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	mux := http.NewServeMux()
	store.Register(mux)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts, store
}

func getJSON(t *testing.T, url string, out any) int {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
	return resp.StatusCode
}

func TestArtistEndpoint(t *testing.T) {
	ts, _ := newCatalogServer(t, false)

	var a Artist
	if code := getJSON(t, ts.URL+"/catalog/artist/"+catalogtest.ArtistAlphaMBID, &a); code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if a.Name != "Alpha" || a.Type != "Group" || a.Country != "GB" {
		t.Errorf("artist = %+v", a)
	}
	if a.Area == nil || a.Area.Name != "United Kingdom" {
		t.Errorf("area = %+v", a.Area)
	}
	if len(a.Aliases) != 1 || a.Aliases[0].Name != "The Alphas" || !a.Aliases[0].Primary {
		t.Errorf("aliases = %+v", a.Aliases)
	}
	if len(a.Tags) != 2 || a.Tags[0].Name != "rock" || a.Tags[0].Count != 5 {
		t.Errorf("tags = %+v (want rock first by count)", a.Tags)
	}
	if len(a.Genres) != 1 || a.Genres[0].Name != "rock" {
		t.Errorf("genres = %+v", a.Genres)
	}
	if len(a.Relationships) != 1 || a.Relationships[0].ArtistMBID != catalogtest.ArtistBetaMBID ||
		a.Relationships[0].Type != "member of band" {
		t.Errorf("relationships = %+v", a.Relationships)
	}
	if len(a.ReleaseGroups) != 1 || a.ReleaseGroups[0].MBID != catalogtest.ReleaseGroupMBID ||
		a.ReleaseGroups[0].Title != "Test Album" || a.ReleaseGroups[0].ArtistCredit != "Alpha" {
		t.Errorf("release groups = %+v", a.ReleaseGroups)
	}

	var b Artist
	if code := getJSON(t, ts.URL+"/catalog/artist/"+catalogtest.ArtistBetaMBID, &b); code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if len(b.ReleaseGroups) != 0 || len(b.Relationships) != 1 {
		t.Errorf("beta = %+v", b)
	}
}

func TestReleaseGroupEndpoint(t *testing.T) {
	ts, _ := newCatalogServer(t, false)

	var rg ReleaseGroup
	if code := getJSON(t, ts.URL+"/catalog/release-group/"+catalogtest.ReleaseGroupMBID, &rg); code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if rg.Title != "Test Album" || rg.PrimaryType != "Album" || rg.FirstReleaseDate != "2001-01-01" {
		t.Errorf("release group = %+v", rg)
	}
	if len(rg.SecondaryTypes) != 1 || rg.SecondaryTypes[0] != "Compilation" {
		t.Errorf("secondary types = %+v", rg.SecondaryTypes)
	}
	if rg.ArtistCredit != "Alpha" {
		t.Errorf("artist credit = %q", rg.ArtistCredit)
	}
	if len(rg.Releases) != 2 || rg.Releases[0].MBID != catalogtest.Release2001MBID ||
		rg.Releases[1].MBID != catalogtest.Release2009MBID {
		t.Fatalf("releases = %+v (want 2001 press first)", rg.Releases)
	}
	if rg.Releases[0].TrackCount != 1 {
		t.Errorf("track count = %d", rg.Releases[0].TrackCount)
	}
}

func TestReleaseEndpoint(t *testing.T) {
	ts, _ := newCatalogServer(t, false)

	var rel Release
	if code := getJSON(t, ts.URL+"/catalog/release/"+catalogtest.Release2001MBID, &rel); code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if rel.Title != "Test Album (2001 press)" || rel.Date != "2001-01-01" || rel.Barcode != "1234567890123" {
		t.Errorf("release = %+v", rel)
	}
	if rel.ReleaseGroup == nil || rel.ReleaseGroup.MBID != catalogtest.ReleaseGroupMBID ||
		rel.ReleaseGroup.Title != "Test Album" {
		t.Errorf("release group ref = %+v", rel.ReleaseGroup)
	}
	if len(rel.Labels) != 1 || rel.Labels[0].MBID != catalogtest.LabelMBID ||
		rel.Labels[0].CatalogNumber != "FERN-001" {
		t.Errorf("labels = %+v", rel.Labels)
	}
	if len(rel.Media) != 1 || rel.Media[0].Format != "CD" || rel.Media[0].TrackCount != 1 {
		t.Errorf("media = %+v", rel.Media)
	}
	if len(rel.Tracks) != 1 || rel.Tracks[0].RecordingMBID != catalogtest.RecordingMBID ||
		rel.Tracks[0].Number != "1" {
		t.Errorf("tracks = %+v", rel.Tracks)
	}
}

func TestRecordingEndpoint(t *testing.T) {
	ts, _ := newCatalogServer(t, false)

	var rec Recording
	if code := getJSON(t, ts.URL+"/catalog/recording/"+catalogtest.RecordingMBID, &rec); code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if rec.Title != "Test Song" || rec.LengthMs != 100000 {
		t.Errorf("recording = %+v", rec)
	}
	if rec.ArtistCredit != "Alpha feat. Beta" || len(rec.Artists) != 2 || rec.Artists[1].Name != "Beta" {
		t.Errorf("artists = %+v credit = %q", rec.Artists, rec.ArtistCredit)
	}
	if len(rec.ISRCs) != 1 || rec.ISRCs[0] != "GBAAA0100001" {
		t.Errorf("isrcs = %+v", rec.ISRCs)
	}
	if len(rec.Works) != 1 || rec.Works[0].MBID != catalogtest.WorkMBID ||
		rec.Works[0].Relationship != "performance" {
		t.Errorf("works = %+v", rec.Works)
	}
	if len(rec.Releases) != 2 || rec.Releases[0].MBID != catalogtest.Release2001MBID {
		t.Fatalf("releases = %+v (want earliest first)", rec.Releases)
	}
	if rec.Releases[0].ReleaseGroup == nil || rec.Releases[0].ReleaseGroup.Title != "Test Album" {
		t.Errorf("release group ref = %+v", rec.Releases[0].ReleaseGroup)
	}
}

func TestLabelEndpoint(t *testing.T) {
	ts, _ := newCatalogServer(t, false)

	var l Label
	if code := getJSON(t, ts.URL+"/catalog/label/"+catalogtest.LabelMBID, &l); code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if l.Name != "Fern Records" || l.LabelCode != 123 || l.Country != "GB" {
		t.Errorf("label = %+v", l)
	}
	if len(l.Aliases) != 1 || l.Aliases[0].Name != "Fern" {
		t.Errorf("aliases = %+v", l.Aliases)
	}
	if len(l.Releases) != 1 || l.Releases[0].MBID != catalogtest.Release2001MBID ||
		l.Releases[0].CatalogNumber != "FERN-001" || l.Releases[0].ArtistCredit != "Alpha" {
		t.Errorf("releases = %+v", l.Releases)
	}
}

func TestWorkEndpoint(t *testing.T) {
	ts, _ := newCatalogServer(t, false)

	var w Work
	if code := getJSON(t, ts.URL+"/catalog/work/"+catalogtest.WorkMBID, &w); code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if w.Title != "Test Song" || w.Type != "Song" || w.Languages != "eng" {
		t.Errorf("work = %+v", w)
	}
	if len(w.ISWCs) != 1 || w.ISWCs[0] != "T-123.456.789-0" {
		t.Errorf("iswcs = %+v", w.ISWCs)
	}
	if len(w.Recordings) != 1 || w.Recordings[0].MBID != catalogtest.RecordingMBID ||
		w.Recordings[0].ArtistCredit != "Alpha feat. Beta" {
		t.Errorf("recordings = %+v", w.Recordings)
	}
}

func TestUnknownMBIDsReturn404(t *testing.T) {
	ts, _ := newCatalogServer(t, false)

	for _, entity := range []string{"artist", "release-group", "release", "recording", "label", "work"} {
		var body struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		}
		code := getJSON(t, ts.URL+"/catalog/"+entity+"/"+catalogtest.UnknownMBID, &body)
		if code != http.StatusNotFound || body.Status != "ERROR" || body.Error == "" {
			t.Errorf("%s: status %d body %+v, want 404 error JSON", entity, code, body)
		}
	}
}

func TestSearchFTS(t *testing.T) {
	ts, _ := newCatalogServer(t, true)

	var body searchResponse
	if code := getJSON(t, ts.URL+"/catalog/search?q=alpha", &body); code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if !body.Indexed {
		t.Fatal("indexed = false, want the FTS fast path")
	}
	found := false
	for _, r := range body.Results {
		if r.Type == "artist" && r.MBID == catalogtest.ArtistAlphaMBID {
			found = true
		}
	}
	if !found {
		t.Errorf("results = %+v, want artist Alpha", body.Results)
	}

	t.Run("prefix tokens match", func(t *testing.T) {
		var body searchResponse
		getJSON(t, ts.URL+"/catalog/search?q=test+so", &body)
		var types []string
		for _, r := range body.Results {
			types = append(types, r.Type)
		}
		if len(body.Results) < 2 {
			t.Errorf("results = %v, want recording and work at least", types)
		}
	})

	t.Run("type filter", func(t *testing.T) {
		var body searchResponse
		getJSON(t, ts.URL+"/catalog/search?q=test&type=recording", &body)
		if len(body.Results) != 1 || body.Results[0].Type != "recording" ||
			body.Results[0].MBID != catalogtest.RecordingMBID {
			t.Errorf("results = %+v, want only the recording", body.Results)
		}
	})

	t.Run("hyphenated type alias", func(t *testing.T) {
		var body searchResponse
		getJSON(t, ts.URL+"/catalog/search?q=test&type=release-group", &body)
		if len(body.Results) != 1 || body.Results[0].Type != "release_group" {
			t.Errorf("results = %+v, want only the release group", body.Results)
		}
	})
}

func TestSearchSlowPath(t *testing.T) {
	ts, _ := newCatalogServer(t, false)

	var body searchResponse
	if code := getJSON(t, ts.URL+"/catalog/search?q=Test+Song", &body); code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if body.Indexed {
		t.Fatal("indexed = true, want the slow path (no search_fts in fixture)")
	}
	got := map[string]bool{}
	for _, r := range body.Results {
		got[r.Type] = true
	}
	for _, want := range []string{"work", "recording", "track"} {
		if !got[want] {
			t.Errorf("results %v missing %s titled 'Test Song'", got, want)
		}
	}

	t.Run("substring tier", func(t *testing.T) {
		var body searchResponse
		getJSON(t, ts.URL+"/catalog/search?q=ern+Reco&type=label", &body)
		if len(body.Results) != 1 || body.Results[0].MBID != catalogtest.LabelMBID {
			t.Errorf("results = %+v, want Fern Records via substring match", body.Results)
		}
	})

	t.Run("alias match", func(t *testing.T) {
		var body searchResponse
		getJSON(t, ts.URL+"/catalog/search?q=The+Alphas&type=artist", &body)
		if len(body.Results) != 1 || body.Results[0].MBID != catalogtest.ArtistAlphaMBID {
			t.Errorf("results = %+v, want Alpha via alias", body.Results)
		}
	})

	t.Run("isrc exact match", func(t *testing.T) {
		var body searchResponse
		getJSON(t, ts.URL+"/catalog/search?q=GBAAA0100001&type=recording", &body)
		if len(body.Results) != 1 || body.Results[0].MBID != catalogtest.RecordingMBID {
			t.Errorf("results = %+v, want recording via ISRC", body.Results)
		}
	})
}

func TestSearchValidation(t *testing.T) {
	ts, _ := newCatalogServer(t, false)

	for name, path := range map[string]string{
		"missing q":     "/catalog/search",
		"unknown type":  "/catalog/search?q=x&type=banana",
		"invalid limit": "/catalog/search?q=x&limit=zero",
	} {
		var body struct {
			Status string `json:"status"`
		}
		if code := getJSON(t, ts.URL+path, &body); code != http.StatusBadRequest || body.Status != "ERROR" {
			t.Errorf("%s: status %d body %+v, want 400", name, code, body)
		}
	}
}

func TestRecordingSummary(t *testing.T) {
	_, store := newCatalogServer(t, false)
	ctx := context.Background()

	sum, err := store.RecordingSummary(ctx, catalogtest.RecordingMBID)
	if err != nil {
		t.Fatal(err)
	}
	want := RecordingSummary{Title: "Test Song", Artist: "Alpha feat. Beta", Album: "Test Album", DurationMs: 100000}
	if sum == nil || *sum != want {
		t.Errorf("summary = %+v, want %+v", sum, want)
	}

	missing, err := store.RecordingSummary(ctx, catalogtest.UnknownMBID)
	if err != nil || missing != nil {
		t.Errorf("unknown = (%+v, %v), want (nil, nil)", missing, err)
	}
}

func TestStoreReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "meta.db")
	catalogtest.BuildDB(t, path, false)

	store, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	rec, err := store.Recording(context.Background(), catalogtest.RecordingMBID)
	if err != nil || rec == nil || rec.Title != "Test Song" {
		t.Fatalf("before reopen: (%+v, %v)", rec, err)
	}

	rebuilt := filepath.Join(dir, "meta.db.new")
	catalogtest.BuildDB(t, rebuilt, false)
	catalogtest.Exec(t, rebuilt, `UPDATE recordings SET title = 'Test Song (Remastered)' WHERE mbid = '`+catalogtest.RecordingMBID+`'`)
	if err := os.Rename(rebuilt, path); err != nil {
		t.Fatal(err)
	}

	if err := store.Reopen(context.Background()); err != nil {
		t.Fatalf("reopen: %v", err)
	}
	rec, err = store.Recording(context.Background(), catalogtest.RecordingMBID)
	if err != nil || rec == nil || rec.Title != "Test Song (Remastered)" {
		t.Fatalf("after reopen: (%+v, %v), want the rebuilt title", rec, err)
	}
}
