package catalogtest

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

const (
	ArtistAlphaMBID  = "a0000000-0000-4000-8000-000000000001"
	ArtistBetaMBID   = "a0000000-0000-4000-8000-000000000002"
	LabelMBID        = "b0000000-0000-4000-8000-000000000001"
	WorkMBID         = "c0000000-0000-4000-8000-000000000001"
	ReleaseGroupMBID = "d0000000-0000-4000-8000-00000000000a"
	Release2001MBID  = "d0000000-0000-4000-8000-00000000000b"
	Release2009MBID  = "d0000000-0000-4000-8000-00000000000c"
	Track2001MBID    = "e0000000-0000-4000-8000-000000000001"
	Track2009MBID    = "e0000000-0000-4000-8000-000000000002"
	RecordingMBID    = "11111111-1111-4111-8111-111111111111"
	UnknownMBID      = "99999999-9999-4999-8999-999999999999"
)

var schema = []string{
	`CREATE TABLE artists (
	    mbid           TEXT PRIMARY KEY,
	    name           TEXT NOT NULL,
	    sort_name      TEXT NOT NULL,
	    disambiguation TEXT NOT NULL DEFAULT '',
	    type           TEXT,
	    country        TEXT,
	    gender         TEXT,
	    begin_date     TEXT,
	    end_date       TEXT,
	    ended          INTEGER NOT NULL DEFAULT 0,
	    area_mbid      TEXT,
	    area_name      TEXT
	)`,
	`CREATE TABLE artist_aliases (
	    artist_mbid TEXT NOT NULL REFERENCES artists(mbid),
	    name        TEXT NOT NULL,
	    sort_name   TEXT,
	    type        TEXT,
	    locale      TEXT NOT NULL DEFAULT '',
	    is_primary  INTEGER NOT NULL DEFAULT 0,
	    PRIMARY KEY (artist_mbid, name, locale)
	)`,
	`CREATE TABLE artist_tags (
	    artist_mbid TEXT NOT NULL REFERENCES artists(mbid),
	    tag         TEXT NOT NULL,
	    count       INTEGER NOT NULL DEFAULT 0,
	    PRIMARY KEY (artist_mbid, tag)
	)`,
	`CREATE TABLE artist_genres (
	    artist_mbid TEXT NOT NULL REFERENCES artists(mbid),
	    genre_mbid  TEXT NOT NULL,
	    genre_name  TEXT NOT NULL,
	    count       INTEGER NOT NULL DEFAULT 0,
	    PRIMARY KEY (artist_mbid, genre_mbid)
	)`,
	`CREATE TABLE artist_relationships (
	    artist_mbid         TEXT NOT NULL REFERENCES artists(mbid),
	    related_artist_mbid TEXT NOT NULL,
	    related_artist_name TEXT NOT NULL DEFAULT '',
	    type                TEXT NOT NULL,
	    direction           TEXT NOT NULL DEFAULT '',
	    begin_date          TEXT NOT NULL DEFAULT '',
	    end_date            TEXT NOT NULL DEFAULT '',
	    ended               INTEGER NOT NULL DEFAULT 0,
	    attributes          TEXT NOT NULL DEFAULT '',
	    PRIMARY KEY (artist_mbid, related_artist_mbid, type, direction, begin_date, end_date, attributes)
	)`,
	`CREATE TABLE labels (
	    mbid           TEXT PRIMARY KEY,
	    name           TEXT NOT NULL,
	    sort_name      TEXT NOT NULL,
	    disambiguation TEXT NOT NULL DEFAULT '',
	    type           TEXT,
	    label_code     INTEGER,
	    country        TEXT,
	    begin_date     TEXT,
	    end_date       TEXT,
	    ended          INTEGER NOT NULL DEFAULT 0,
	    area_mbid      TEXT,
	    area_name      TEXT
	)`,
	`CREATE TABLE label_aliases (
	    label_mbid TEXT NOT NULL REFERENCES labels(mbid),
	    name       TEXT NOT NULL,
	    sort_name  TEXT,
	    type       TEXT,
	    locale     TEXT NOT NULL DEFAULT '',
	    is_primary INTEGER NOT NULL DEFAULT 0,
	    PRIMARY KEY (label_mbid, name, locale)
	)`,
	`CREATE TABLE label_tags (
	    label_mbid TEXT NOT NULL REFERENCES labels(mbid),
	    tag        TEXT NOT NULL,
	    count      INTEGER NOT NULL DEFAULT 0,
	    PRIMARY KEY (label_mbid, tag)
	)`,
	`CREATE TABLE label_genres (
	    label_mbid TEXT NOT NULL REFERENCES labels(mbid),
	    genre_mbid TEXT NOT NULL,
	    genre_name TEXT NOT NULL,
	    count      INTEGER NOT NULL DEFAULT 0,
	    PRIMARY KEY (label_mbid, genre_mbid)
	)`,
	`CREATE TABLE works (
	    mbid           TEXT PRIMARY KEY,
	    title          TEXT NOT NULL,
	    disambiguation TEXT NOT NULL DEFAULT '',
	    type           TEXT,
	    languages      TEXT NOT NULL DEFAULT ''
	)`,
	`CREATE TABLE work_aliases (
	    work_mbid  TEXT NOT NULL REFERENCES works(mbid),
	    name       TEXT NOT NULL,
	    sort_name  TEXT,
	    type       TEXT,
	    locale     TEXT NOT NULL DEFAULT '',
	    is_primary INTEGER NOT NULL DEFAULT 0,
	    PRIMARY KEY (work_mbid, name, locale)
	)`,
	`CREATE TABLE work_iswcs (
	    work_mbid TEXT NOT NULL REFERENCES works(mbid),
	    iswc      TEXT NOT NULL,
	    PRIMARY KEY (work_mbid, iswc)
	)`,
	`CREATE TABLE work_tags (
	    work_mbid TEXT NOT NULL REFERENCES works(mbid),
	    tag       TEXT NOT NULL,
	    count     INTEGER NOT NULL DEFAULT 0,
	    PRIMARY KEY (work_mbid, tag)
	)`,
	`CREATE TABLE recording_works (
	    recording_mbid TEXT NOT NULL,
	    work_mbid      TEXT NOT NULL REFERENCES works(mbid),
	    type           TEXT NOT NULL,
	    attributes     TEXT NOT NULL DEFAULT '',
	    PRIMARY KEY (recording_mbid, work_mbid, type, attributes)
	)`,
	`CREATE TABLE release_groups (
	    mbid               TEXT PRIMARY KEY,
	    title              TEXT NOT NULL,
	    primary_type       TEXT,
	    disambiguation     TEXT NOT NULL DEFAULT '',
	    first_release_date TEXT
	)`,
	`CREATE TABLE release_group_secondary_types (
	    release_group_mbid TEXT NOT NULL REFERENCES release_groups(mbid),
	    type               TEXT NOT NULL,
	    PRIMARY KEY (release_group_mbid, type)
	)`,
	`CREATE TABLE release_group_artists (
	    release_group_mbid TEXT NOT NULL REFERENCES release_groups(mbid),
	    artist_mbid        TEXT NOT NULL,
	    artist_name        TEXT NOT NULL,
	    join_phrase        TEXT NOT NULL DEFAULT '',
	    position           INTEGER NOT NULL,
	    PRIMARY KEY (release_group_mbid, position)
	)`,
	`CREATE TABLE release_group_tags (
	    release_group_mbid TEXT NOT NULL REFERENCES release_groups(mbid),
	    tag                TEXT NOT NULL,
	    count              INTEGER NOT NULL DEFAULT 0,
	    PRIMARY KEY (release_group_mbid, tag)
	)`,
	`CREATE TABLE releases (
	    mbid               TEXT PRIMARY KEY,
	    title              TEXT NOT NULL,
	    status             TEXT,
	    date               TEXT,
	    country            TEXT,
	    barcode            TEXT,
	    packaging          TEXT,
	    language           TEXT,
	    script             TEXT,
	    release_group_mbid TEXT REFERENCES release_groups(mbid)
	)`,
	`CREATE TABLE release_artists (
	    release_mbid TEXT NOT NULL REFERENCES releases(mbid),
	    artist_mbid  TEXT NOT NULL,
	    artist_name  TEXT NOT NULL,
	    join_phrase  TEXT NOT NULL DEFAULT '',
	    position     INTEGER NOT NULL,
	    PRIMARY KEY (release_mbid, position)
	)`,
	`CREATE TABLE release_labels (
	    release_mbid   TEXT NOT NULL REFERENCES releases(mbid),
	    label_mbid     TEXT NOT NULL DEFAULT '',
	    label_name     TEXT NOT NULL DEFAULT '',
	    catalog_number TEXT NOT NULL DEFAULT '',
	    PRIMARY KEY (release_mbid, label_mbid, catalog_number)
	)`,
	`CREATE TABLE release_media (
	    release_mbid TEXT NOT NULL REFERENCES releases(mbid),
	    position     INTEGER NOT NULL,
	    format       TEXT,
	    track_count  INTEGER NOT NULL DEFAULT 0,
	    PRIMARY KEY (release_mbid, position)
	)`,
	`CREATE TABLE recordings (
	    mbid               TEXT PRIMARY KEY,
	    title              TEXT NOT NULL,
	    length             INTEGER,
	    disambiguation     TEXT NOT NULL DEFAULT '',
	    video              INTEGER NOT NULL DEFAULT 0,
	    first_release_date TEXT
	)`,
	`CREATE TABLE recording_artists (
	    recording_mbid TEXT NOT NULL REFERENCES recordings(mbid),
	    artist_mbid    TEXT NOT NULL,
	    artist_name    TEXT NOT NULL,
	    join_phrase    TEXT NOT NULL DEFAULT '',
	    position       INTEGER NOT NULL,
	    PRIMARY KEY (recording_mbid, position)
	)`,
	`CREATE TABLE recording_isrcs (
	    recording_mbid TEXT NOT NULL REFERENCES recordings(mbid),
	    isrc           TEXT NOT NULL,
	    PRIMARY KEY (recording_mbid, isrc)
	)`,
	`CREATE TABLE recording_tags (
	    recording_mbid TEXT NOT NULL REFERENCES recordings(mbid),
	    tag            TEXT NOT NULL,
	    count          INTEGER NOT NULL DEFAULT 0,
	    PRIMARY KEY (recording_mbid, tag)
	)`,
	`CREATE TABLE tracks (
	    mbid           TEXT PRIMARY KEY,
	    release_mbid   TEXT NOT NULL REFERENCES releases(mbid),
	    recording_mbid TEXT NOT NULL REFERENCES recordings(mbid),
	    media_position INTEGER NOT NULL,
	    position       INTEGER NOT NULL,
	    number         TEXT NOT NULL,
	    title          TEXT NOT NULL,
	    length         INTEGER
	)`,
	`CREATE TABLE external_links (
	    entity_type TEXT NOT NULL,
	    entity_mbid TEXT NOT NULL,
	    rel_type    TEXT NOT NULL,
	    url         TEXT NOT NULL,
	    PRIMARY KEY (entity_mbid, rel_type, url)
	)`,
}

const createSearchFTS = `
CREATE VIRTUAL TABLE search_fts USING fts5(
    entity_type UNINDEXED,
    entity_mbid UNINDEXED,
    heading,
    subtitle,
    meta,
    aux,
    tokenize = 'unicode61 remove_diacritics 2'
)`

var fixtureRows = []string{
	`INSERT INTO artists VALUES ('` + ArtistAlphaMBID + `', 'Alpha', 'Alpha', 'UK group', 'Group', 'GB', NULL,
	    '1990', NULL, 0, 'f0000000-0000-4000-8000-0000000000aa', 'United Kingdom')`,
	`INSERT INTO artists VALUES ('` + ArtistBetaMBID + `', 'Beta', 'Beta', '', 'Person', 'GB', 'male',
	    NULL, NULL, 0, NULL, NULL)`,
	`INSERT INTO artist_aliases VALUES ('` + ArtistAlphaMBID + `', 'The Alphas', 'Alphas, The', 'Artist name', '', 1)`,
	`INSERT INTO artist_tags VALUES ('` + ArtistAlphaMBID + `', 'rock', 5)`,
	`INSERT INTO artist_tags VALUES ('` + ArtistAlphaMBID + `', 'electronic', 2)`,
	`INSERT INTO artist_genres VALUES ('` + ArtistAlphaMBID + `', 'f0000000-0000-4000-8000-0000000000bb', 'rock', 5)`,
	`INSERT INTO artist_relationships VALUES ('` + ArtistAlphaMBID + `', '` + ArtistBetaMBID + `', 'Beta',
	    'member of band', 'backward', '1990', '', 0, '')`,
	`INSERT INTO artist_relationships VALUES ('` + ArtistBetaMBID + `', '` + ArtistAlphaMBID + `', 'Alpha',
	    'member of band', 'forward', '1990', '', 0, '')`,

	`INSERT INTO labels VALUES ('` + LabelMBID + `', 'Fern Records', 'Fern Records', 'UK indie', 'Original Production',
	    123, 'GB', '1995', NULL, 0, NULL, NULL)`,
	`INSERT INTO label_aliases VALUES ('` + LabelMBID + `', 'Fern', 'Fern', NULL, '', 0)`,

	`INSERT INTO works VALUES ('` + WorkMBID + `', 'Test Song', '', 'Song', 'eng')`,
	`INSERT INTO work_iswcs VALUES ('` + WorkMBID + `', 'T-123.456.789-0')`,
	`INSERT INTO recording_works VALUES ('` + RecordingMBID + `', '` + WorkMBID + `', 'performance', '')`,

	`INSERT INTO release_groups VALUES ('` + ReleaseGroupMBID + `', 'Test Album', 'Album', '', '2001-01-01')`,
	`INSERT INTO release_group_secondary_types VALUES ('` + ReleaseGroupMBID + `', 'Compilation')`,
	`INSERT INTO release_group_artists VALUES ('` + ReleaseGroupMBID + `', '` + ArtistAlphaMBID + `', 'Alpha', '', 0)`,

	`INSERT INTO releases VALUES ('` + Release2001MBID + `', 'Test Album (2001 press)', 'Official', '2001-01-01',
	    'GB', '1234567890123', 'Jewel Case', 'eng', 'Latn', '` + ReleaseGroupMBID + `')`,
	`INSERT INTO releases VALUES ('` + Release2009MBID + `', 'Test Album (2009 remaster)', 'Official', '2009-05-01',
	    'GB', NULL, NULL, 'eng', 'Latn', '` + ReleaseGroupMBID + `')`,
	`INSERT INTO release_artists VALUES ('` + Release2001MBID + `', '` + ArtistAlphaMBID + `', 'Alpha', '', 0)`,
	`INSERT INTO release_artists VALUES ('` + Release2009MBID + `', '` + ArtistAlphaMBID + `', 'Alpha', '', 0)`,
	`INSERT INTO release_labels VALUES ('` + Release2001MBID + `', '` + LabelMBID + `', 'Fern Records', 'FERN-001')`,
	`INSERT INTO release_media VALUES ('` + Release2001MBID + `', 1, 'CD', 1)`,
	`INSERT INTO release_media VALUES ('` + Release2009MBID + `', 1, 'CD', 1)`,

	`INSERT INTO recordings (mbid, title, length, first_release_date)
	    VALUES ('` + RecordingMBID + `', 'Test Song', 100000, '2001-01-01')`,
	`INSERT INTO recording_artists VALUES ('` + RecordingMBID + `', '` + ArtistAlphaMBID + `', 'Alpha', ' feat. ', 0)`,
	`INSERT INTO recording_artists VALUES ('` + RecordingMBID + `', '` + ArtistBetaMBID + `', 'Beta', '', 1)`,
	`INSERT INTO recording_isrcs VALUES ('` + RecordingMBID + `', 'GBAAA0100001')`,

	`INSERT INTO tracks VALUES ('` + Track2001MBID + `', '` + Release2001MBID + `', '` + RecordingMBID + `',
	    1, 1, '1', 'Test Song', 100000)`,
	`INSERT INTO tracks VALUES ('` + Track2009MBID + `', '` + Release2009MBID + `', '` + RecordingMBID + `',
	    1, 1, '1', 'Test Song', 100000)`,
}

var ftsRows = []string{
	`INSERT INTO search_fts VALUES ('artist', '` + ArtistAlphaMBID + `', 'Alpha', 'Alpha', 'Group GB', 'The Alphas')`,
	`INSERT INTO search_fts VALUES ('artist', '` + ArtistBetaMBID + `', 'Beta', 'Beta', 'Person GB', '')`,
	`INSERT INTO search_fts VALUES ('label', '` + LabelMBID + `', 'Fern Records', 'Fern Records', 'Original Production GB', 'Fern')`,
	`INSERT INTO search_fts VALUES ('work', '` + WorkMBID + `', 'Test Song', '', 'Song', '')`,
	`INSERT INTO search_fts VALUES ('release_group', '` + ReleaseGroupMBID + `', 'Test Album', 'Alpha', 'Album 2001-01-01', '')`,
	`INSERT INTO search_fts VALUES ('release', '` + Release2001MBID + `', 'Test Album (2001 press)', 'Alpha', '2001-01-01 GB', 'Fern Records')`,
	`INSERT INTO search_fts VALUES ('release', '` + Release2009MBID + `', 'Test Album (2009 remaster)', 'Alpha', '2009-05-01 GB', '')`,
	`INSERT INTO search_fts VALUES ('recording', '` + RecordingMBID + `', 'Test Song', 'Alpha feat. Beta', '2001-01-01', '')`,
	`INSERT INTO search_fts VALUES ('track', '` + Track2001MBID + `', 'Test Song', 'Alpha', '1 Test Album (2001 press)', '')`,
}

func BuildDB(t testing.TB, path string, withFTS bool) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("open fixture db: %v", err)
	}
	defer db.Close()

	stmts := append([]string{}, schema...)
	stmts = append(stmts, fixtureRows...)
	if withFTS {
		stmts = append(stmts, createSearchFTS)
		stmts = append(stmts, ftsRows...)
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("fixture db: %v\n%s", err, stmt)
		}
	}
}

func Exec(t testing.TB, path string, stmts ...string) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("open fixture db: %v", err)
	}
	defer db.Close()
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("fixture db: %v\n%s", err, stmt)
		}
	}
}
