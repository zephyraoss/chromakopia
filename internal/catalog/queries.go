package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type Area struct {
	MBID string `json:"mbid,omitempty"`
	Name string `json:"name"`
}

type Alias struct {
	Name     string `json:"name"`
	SortName string `json:"sortName,omitempty"`
	Type     string `json:"type,omitempty"`
	Locale   string `json:"locale,omitempty"`
	Primary  bool   `json:"primary,omitempty"`
}

type Tag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type Genre struct {
	MBID  string `json:"mbid"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type CreditedArtist struct {
	MBID       string `json:"mbid"`
	Name       string `json:"name"`
	JoinPhrase string `json:"joinPhrase,omitempty"`
}

type ArtistRelationship struct {
	ArtistMBID string `json:"artistMbid"`
	ArtistName string `json:"artistName"`
	Type       string `json:"type"`
	Direction  string `json:"direction,omitempty"`
	BeginDate  string `json:"beginDate,omitempty"`
	EndDate    string `json:"endDate,omitempty"`
	Ended      bool   `json:"ended,omitempty"`
	Attributes string `json:"attributes,omitempty"`
}

type ReleaseGroupSummary struct {
	MBID             string `json:"mbid"`
	Title            string `json:"title"`
	PrimaryType      string `json:"primaryType,omitempty"`
	FirstReleaseDate string `json:"firstReleaseDate,omitempty"`
	ArtistCredit     string `json:"artistCredit,omitempty"`
}

type Artist struct {
	MBID           string                `json:"mbid"`
	Name           string                `json:"name"`
	SortName       string                `json:"sortName"`
	Disambiguation string                `json:"disambiguation,omitempty"`
	Type           string                `json:"type,omitempty"`
	Country        string                `json:"country,omitempty"`
	Gender         string                `json:"gender,omitempty"`
	BeginDate      string                `json:"beginDate,omitempty"`
	EndDate        string                `json:"endDate,omitempty"`
	Ended          bool                  `json:"ended"`
	Area           *Area                 `json:"area,omitempty"`
	Aliases        []Alias               `json:"aliases"`
	Tags           []Tag                 `json:"tags"`
	Genres         []Genre               `json:"genres"`
	Relationships  []ArtistRelationship  `json:"relationships"`
	ReleaseGroups  []ReleaseGroupSummary `json:"releaseGroups"`
}

type ReleaseSummary struct {
	MBID       string `json:"mbid"`
	Title      string `json:"title"`
	Status     string `json:"status,omitempty"`
	Date       string `json:"date,omitempty"`
	Country    string `json:"country,omitempty"`
	TrackCount int    `json:"trackCount"`
}

type ReleaseGroup struct {
	MBID             string           `json:"mbid"`
	Title            string           `json:"title"`
	PrimaryType      string           `json:"primaryType,omitempty"`
	SecondaryTypes   []string         `json:"secondaryTypes"`
	Disambiguation   string           `json:"disambiguation,omitempty"`
	FirstReleaseDate string           `json:"firstReleaseDate,omitempty"`
	Artists          []CreditedArtist `json:"artists"`
	ArtistCredit     string           `json:"artistCredit,omitempty"`
	Releases         []ReleaseSummary `json:"releases"`
}

type ReleaseGroupRef struct {
	MBID        string `json:"mbid"`
	Title       string `json:"title"`
	PrimaryType string `json:"primaryType,omitempty"`
}

type ReleaseLabel struct {
	MBID          string `json:"mbid,omitempty"`
	Name          string `json:"name"`
	CatalogNumber string `json:"catalogNumber,omitempty"`
}

type Medium struct {
	Position   int    `json:"position"`
	Format     string `json:"format,omitempty"`
	TrackCount int    `json:"trackCount"`
}

type Track struct {
	MBID          string `json:"mbid"`
	MediaPosition int    `json:"mediaPosition"`
	Position      int    `json:"position"`
	Number        string `json:"number"`
	Title         string `json:"title"`
	LengthMs      int64  `json:"lengthMs,omitempty"`
	RecordingMBID string `json:"recordingMbid"`
}

type Release struct {
	MBID         string           `json:"mbid"`
	Title        string           `json:"title"`
	Status       string           `json:"status,omitempty"`
	Date         string           `json:"date,omitempty"`
	Country      string           `json:"country,omitempty"`
	Barcode      string           `json:"barcode,omitempty"`
	Packaging    string           `json:"packaging,omitempty"`
	Language     string           `json:"language,omitempty"`
	Script       string           `json:"script,omitempty"`
	Artists      []CreditedArtist `json:"artists"`
	ArtistCredit string           `json:"artistCredit,omitempty"`
	ReleaseGroup *ReleaseGroupRef `json:"releaseGroup,omitempty"`
	Labels       []ReleaseLabel   `json:"labels"`
	Media        []Medium         `json:"media"`
	Tracks       []Track          `json:"tracks"`
}

type RecordingWork struct {
	MBID         string `json:"mbid"`
	Title        string `json:"title"`
	Type         string `json:"type,omitempty"`
	Relationship string `json:"relationship"`
	Attributes   string `json:"attributes,omitempty"`
}

type RecordingRelease struct {
	MBID         string           `json:"mbid"`
	Title        string           `json:"title"`
	Status       string           `json:"status,omitempty"`
	Date         string           `json:"date,omitempty"`
	Country      string           `json:"country,omitempty"`
	ReleaseGroup *ReleaseGroupRef `json:"releaseGroup,omitempty"`
}

type Recording struct {
	MBID             string             `json:"mbid"`
	Title            string             `json:"title"`
	LengthMs         int64              `json:"lengthMs,omitempty"`
	Disambiguation   string             `json:"disambiguation,omitempty"`
	Video            bool               `json:"video,omitempty"`
	FirstReleaseDate string             `json:"firstReleaseDate,omitempty"`
	Artists          []CreditedArtist   `json:"artists"`
	ArtistCredit     string             `json:"artistCredit,omitempty"`
	ISRCs            []string           `json:"isrcs"`
	Works            []RecordingWork    `json:"works"`
	Releases         []RecordingRelease `json:"releases"`
}

type LabelRelease struct {
	MBID          string `json:"mbid"`
	Title         string `json:"title"`
	Status        string `json:"status,omitempty"`
	Date          string `json:"date,omitempty"`
	Country       string `json:"country,omitempty"`
	CatalogNumber string `json:"catalogNumber,omitempty"`
	ArtistCredit  string `json:"artistCredit,omitempty"`
}

type Label struct {
	MBID           string         `json:"mbid"`
	Name           string         `json:"name"`
	SortName       string         `json:"sortName"`
	Disambiguation string         `json:"disambiguation,omitempty"`
	Type           string         `json:"type,omitempty"`
	LabelCode      int            `json:"labelCode,omitempty"`
	Country        string         `json:"country,omitempty"`
	BeginDate      string         `json:"beginDate,omitempty"`
	EndDate        string         `json:"endDate,omitempty"`
	Ended          bool           `json:"ended"`
	Area           *Area          `json:"area,omitempty"`
	Aliases        []Alias        `json:"aliases"`
	Releases       []LabelRelease `json:"releases"`
}

type WorkRecording struct {
	MBID         string `json:"mbid"`
	Title        string `json:"title"`
	ArtistCredit string `json:"artistCredit,omitempty"`
	Relationship string `json:"relationship"`
	Attributes   string `json:"attributes,omitempty"`
}

type Work struct {
	MBID           string          `json:"mbid"`
	Title          string          `json:"title"`
	Disambiguation string          `json:"disambiguation,omitempty"`
	Type           string          `json:"type,omitempty"`
	Languages      string          `json:"languages,omitempty"`
	ISWCs          []string        `json:"iswcs"`
	Recordings     []WorkRecording `json:"recordings"`
}

func creditSubquery(table, fkColumn, outerColumn string) string {
	return fmt.Sprintf(`COALESCE((
        SELECT group_concat(piece, '')
        FROM (
            SELECT c.artist_name || c.join_phrase AS piece
            FROM %s c
            WHERE c.%s = %s
            ORDER BY c.position
        )
    ), '')`, table, fkColumn, outerColumn)
}

func creditString(artists []CreditedArtist) string {
	var b strings.Builder
	for _, a := range artists {
		b.WriteString(a.Name)
		b.WriteString(a.JoinPhrase)
	}
	return strings.TrimSpace(b.String())
}

func queryCredits(ctx context.Context, db *sql.DB, table, fkColumn, mbid string) ([]CreditedArtist, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf(
		`SELECT artist_mbid, artist_name, join_phrase FROM %s WHERE %s = ? ORDER BY position`,
		table, fkColumn), mbid)
	if err != nil {
		return nil, fmt.Errorf("%s for %s: %w", table, mbid, err)
	}
	defer rows.Close()

	credits := make([]CreditedArtist, 0, 2)
	for rows.Next() {
		var c CreditedArtist
		if err := rows.Scan(&c.MBID, &c.Name, &c.JoinPhrase); err != nil {
			return nil, err
		}
		credits = append(credits, c)
	}
	return credits, rows.Err()
}

func queryAliases(ctx context.Context, db *sql.DB, table, fkColumn, mbid string) ([]Alias, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf(
		`SELECT name, COALESCE(sort_name, ''), COALESCE(type, ''), locale, is_primary
		 FROM %s WHERE %s = ? ORDER BY is_primary DESC, name`,
		table, fkColumn), mbid)
	if err != nil {
		return nil, fmt.Errorf("%s for %s: %w", table, mbid, err)
	}
	defer rows.Close()

	aliases := make([]Alias, 0, 4)
	for rows.Next() {
		var a Alias
		var primary int
		if err := rows.Scan(&a.Name, &a.SortName, &a.Type, &a.Locale, &primary); err != nil {
			return nil, err
		}
		a.Primary = primary != 0
		aliases = append(aliases, a)
	}
	return aliases, rows.Err()
}

func scanArea(mbid, name sql.NullString) *Area {
	if mbid.String == "" && name.String == "" {
		return nil
	}
	return &Area{MBID: mbid.String, Name: name.String}
}

func (s *Store) Artist(ctx context.Context, mbid string) (*Artist, error) {
	db := s.DB()
	a := &Artist{MBID: mbid}
	var typ, country, gender, begin, end, areaMBID, areaName sql.NullString
	var ended int
	err := db.QueryRowContext(ctx,
		`SELECT name, sort_name, disambiguation, type, country, gender,
		        begin_date, end_date, ended, area_mbid, area_name
		 FROM artists WHERE mbid = ?`, mbid,
	).Scan(&a.Name, &a.SortName, &a.Disambiguation, &typ, &country, &gender,
		&begin, &end, &ended, &areaMBID, &areaName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("artist %s: %w", mbid, err)
	}
	a.Type, a.Country, a.Gender = typ.String, country.String, gender.String
	a.BeginDate, a.EndDate = begin.String, end.String
	a.Ended = ended != 0
	a.Area = scanArea(areaMBID, areaName)

	if a.Aliases, err = queryAliases(ctx, db, "artist_aliases", "artist_mbid", mbid); err != nil {
		return nil, err
	}
	if a.Tags, err = s.tags(ctx, db, "artist_tags", "artist_mbid", mbid); err != nil {
		return nil, err
	}
	if a.Genres, err = s.genres(ctx, db, "artist_genres", "artist_mbid", mbid); err != nil {
		return nil, err
	}
	if a.Relationships, err = s.artistRelationships(ctx, db, mbid); err != nil {
		return nil, err
	}
	if a.ReleaseGroups, err = s.artistReleaseGroups(ctx, db, mbid); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Store) tags(ctx context.Context, db *sql.DB, table, fkColumn, mbid string) ([]Tag, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf(
		`SELECT tag, count FROM %s WHERE %s = ? ORDER BY count DESC, tag`, table, fkColumn), mbid)
	if err != nil {
		return nil, fmt.Errorf("%s for %s: %w", table, mbid, err)
	}
	defer rows.Close()

	tags := make([]Tag, 0, 8)
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.Name, &t.Count); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (s *Store) genres(ctx context.Context, db *sql.DB, table, fkColumn, mbid string) ([]Genre, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf(
		`SELECT genre_mbid, genre_name, count FROM %s WHERE %s = ? ORDER BY count DESC, genre_name`,
		table, fkColumn), mbid)
	if err != nil {
		return nil, fmt.Errorf("%s for %s: %w", table, mbid, err)
	}
	defer rows.Close()

	genres := make([]Genre, 0, 8)
	for rows.Next() {
		var g Genre
		if err := rows.Scan(&g.MBID, &g.Name, &g.Count); err != nil {
			return nil, err
		}
		genres = append(genres, g)
	}
	return genres, rows.Err()
}

func (s *Store) artistRelationships(ctx context.Context, db *sql.DB, mbid string) ([]ArtistRelationship, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT related_artist_mbid, related_artist_name, type, direction,
		        begin_date, end_date, ended, attributes
		 FROM artist_relationships WHERE artist_mbid = ?
		 ORDER BY type, related_artist_name`, mbid)
	if err != nil {
		return nil, fmt.Errorf("artist relationships %s: %w", mbid, err)
	}
	defer rows.Close()

	rels := make([]ArtistRelationship, 0, 8)
	for rows.Next() {
		var rel ArtistRelationship
		var ended int
		if err := rows.Scan(&rel.ArtistMBID, &rel.ArtistName, &rel.Type, &rel.Direction,
			&rel.BeginDate, &rel.EndDate, &ended, &rel.Attributes); err != nil {
			return nil, err
		}
		rel.Ended = ended != 0
		rels = append(rels, rel)
	}
	return rels, rows.Err()
}

func (s *Store) artistReleaseGroups(ctx context.Context, db *sql.DB, mbid string) ([]ReleaseGroupSummary, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT rg.mbid, rg.title, COALESCE(rg.primary_type, ''),
		        COALESCE(rg.first_release_date, ''),
		        `+creditSubquery("release_group_artists", "release_group_mbid", "rg.mbid")+`
		 FROM release_group_artists rga
		 JOIN release_groups rg ON rg.mbid = rga.release_group_mbid
		 WHERE rga.artist_mbid = ?
		   AND rga.artist_name IN (
		       SELECT name FROM artists WHERE mbid = ?
		       UNION
		       SELECT name FROM artist_aliases WHERE artist_mbid = ?)
		 GROUP BY rg.mbid
		 ORDER BY (rg.first_release_date IS NULL OR rg.first_release_date = ''),
		          rg.first_release_date, rg.title`, mbid, mbid, mbid)
	if err != nil {
		return nil, fmt.Errorf("artist release groups %s: %w", mbid, err)
	}
	defer rows.Close()

	groups := make([]ReleaseGroupSummary, 0, 16)
	for rows.Next() {
		var g ReleaseGroupSummary
		if err := rows.Scan(&g.MBID, &g.Title, &g.PrimaryType, &g.FirstReleaseDate, &g.ArtistCredit); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (s *Store) ReleaseGroup(ctx context.Context, mbid string) (*ReleaseGroup, error) {
	db := s.DB()
	rg := &ReleaseGroup{MBID: mbid}
	var primaryType, firstDate sql.NullString
	err := db.QueryRowContext(ctx,
		`SELECT title, primary_type, disambiguation, first_release_date
		 FROM release_groups WHERE mbid = ?`, mbid,
	).Scan(&rg.Title, &primaryType, &rg.Disambiguation, &firstDate)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("release group %s: %w", mbid, err)
	}
	rg.PrimaryType, rg.FirstReleaseDate = primaryType.String, firstDate.String

	rg.SecondaryTypes = make([]string, 0, 2)
	rows, err := db.QueryContext(ctx,
		`SELECT type FROM release_group_secondary_types WHERE release_group_mbid = ? ORDER BY type`, mbid)
	if err != nil {
		return nil, fmt.Errorf("release group secondary types %s: %w", mbid, err)
	}
	defer rows.Close()
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		rg.SecondaryTypes = append(rg.SecondaryTypes, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if rg.Artists, err = queryCredits(ctx, db, "release_group_artists", "release_group_mbid", mbid); err != nil {
		return nil, err
	}
	rg.ArtistCredit = creditString(rg.Artists)

	if rg.Releases, err = s.releaseSummaries(ctx, db, mbid); err != nil {
		return nil, err
	}
	return rg, nil
}

func (s *Store) releaseSummaries(ctx context.Context, db *sql.DB, rgMBID string) ([]ReleaseSummary, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT r.mbid, r.title, COALESCE(r.status, ''), COALESCE(r.date, ''), COALESCE(r.country, ''),
		        (SELECT COALESCE(SUM(rm.track_count), 0) FROM release_media rm WHERE rm.release_mbid = r.mbid)
		 FROM releases r WHERE r.release_group_mbid = ?
		 ORDER BY (r.date IS NULL OR r.date = ''), r.date, r.title`, rgMBID)
	if err != nil {
		return nil, fmt.Errorf("releases of group %s: %w", rgMBID, err)
	}
	defer rows.Close()

	releases := make([]ReleaseSummary, 0, 4)
	for rows.Next() {
		var r ReleaseSummary
		if err := rows.Scan(&r.MBID, &r.Title, &r.Status, &r.Date, &r.Country, &r.TrackCount); err != nil {
			return nil, err
		}
		releases = append(releases, r)
	}
	return releases, rows.Err()
}

func (s *Store) Release(ctx context.Context, mbid string) (*Release, error) {
	db := s.DB()
	rel := &Release{MBID: mbid}
	var status, date, country, barcode, packaging, language, script, rgMBID sql.NullString
	err := db.QueryRowContext(ctx,
		`SELECT title, status, date, country, barcode, packaging, language, script, release_group_mbid
		 FROM releases WHERE mbid = ?`, mbid,
	).Scan(&rel.Title, &status, &date, &country, &barcode, &packaging, &language, &script, &rgMBID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("release %s: %w", mbid, err)
	}
	rel.Status, rel.Date, rel.Country = status.String, date.String, country.String
	rel.Barcode, rel.Packaging = barcode.String, packaging.String
	rel.Language, rel.Script = language.String, script.String

	if rgMBID.String != "" {
		ref := &ReleaseGroupRef{MBID: rgMBID.String}
		var primaryType sql.NullString
		err := db.QueryRowContext(ctx,
			`SELECT title, primary_type FROM release_groups WHERE mbid = ?`, rgMBID.String,
		).Scan(&ref.Title, &primaryType)
		switch {
		case errors.Is(err, sql.ErrNoRows):
		case err != nil:
			return nil, fmt.Errorf("release group of %s: %w", mbid, err)
		default:
			ref.PrimaryType = primaryType.String
		}
		rel.ReleaseGroup = ref
	}

	if rel.Artists, err = queryCredits(ctx, db, "release_artists", "release_mbid", mbid); err != nil {
		return nil, err
	}
	rel.ArtistCredit = creditString(rel.Artists)

	if rel.Labels, err = s.releaseLabels(ctx, db, mbid); err != nil {
		return nil, err
	}
	if rel.Media, err = s.releaseMedia(ctx, db, mbid); err != nil {
		return nil, err
	}
	if rel.Tracks, err = s.releaseTracks(ctx, db, mbid); err != nil {
		return nil, err
	}
	return rel, nil
}

func (s *Store) releaseLabels(ctx context.Context, db *sql.DB, mbid string) ([]ReleaseLabel, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT label_mbid, label_name, catalog_number
		 FROM release_labels WHERE release_mbid = ? ORDER BY label_name, catalog_number`, mbid)
	if err != nil {
		return nil, fmt.Errorf("release labels %s: %w", mbid, err)
	}
	defer rows.Close()

	labels := make([]ReleaseLabel, 0, 2)
	for rows.Next() {
		var l ReleaseLabel
		if err := rows.Scan(&l.MBID, &l.Name, &l.CatalogNumber); err != nil {
			return nil, err
		}
		labels = append(labels, l)
	}
	return labels, rows.Err()
}

func (s *Store) releaseMedia(ctx context.Context, db *sql.DB, mbid string) ([]Medium, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT position, COALESCE(format, ''), track_count
		 FROM release_media WHERE release_mbid = ? ORDER BY position`, mbid)
	if err != nil {
		return nil, fmt.Errorf("release media %s: %w", mbid, err)
	}
	defer rows.Close()

	media := make([]Medium, 0, 2)
	for rows.Next() {
		var m Medium
		if err := rows.Scan(&m.Position, &m.Format, &m.TrackCount); err != nil {
			return nil, err
		}
		media = append(media, m)
	}
	return media, rows.Err()
}

func (s *Store) releaseTracks(ctx context.Context, db *sql.DB, mbid string) ([]Track, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT mbid, media_position, position, number, title, COALESCE(length, 0), recording_mbid
		 FROM tracks WHERE release_mbid = ? ORDER BY media_position, position`, mbid)
	if err != nil {
		return nil, fmt.Errorf("release tracks %s: %w", mbid, err)
	}
	defer rows.Close()

	tracks := make([]Track, 0, 16)
	for rows.Next() {
		var t Track
		if err := rows.Scan(&t.MBID, &t.MediaPosition, &t.Position, &t.Number, &t.Title, &t.LengthMs, &t.RecordingMBID); err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func (s *Store) Recording(ctx context.Context, mbid string) (*Recording, error) {
	db := s.DB()
	rec := &Recording{MBID: mbid}
	var length sql.NullInt64
	var firstDate sql.NullString
	var video int
	err := db.QueryRowContext(ctx,
		`SELECT title, length, disambiguation, video, first_release_date
		 FROM recordings WHERE mbid = ?`, mbid,
	).Scan(&rec.Title, &length, &rec.Disambiguation, &video, &firstDate)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("recording %s: %w", mbid, err)
	}
	rec.LengthMs = length.Int64
	rec.Video = video != 0
	rec.FirstReleaseDate = firstDate.String

	if rec.Artists, err = queryCredits(ctx, db, "recording_artists", "recording_mbid", mbid); err != nil {
		return nil, err
	}
	rec.ArtistCredit = creditString(rec.Artists)

	rec.ISRCs = make([]string, 0, 2)
	rows, err := db.QueryContext(ctx,
		`SELECT isrc FROM recording_isrcs WHERE recording_mbid = ? ORDER BY isrc`, mbid)
	if err != nil {
		return nil, fmt.Errorf("recording isrcs %s: %w", mbid, err)
	}
	defer rows.Close()
	for rows.Next() {
		var isrc string
		if err := rows.Scan(&isrc); err != nil {
			return nil, err
		}
		rec.ISRCs = append(rec.ISRCs, isrc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if rec.Works, err = s.recordingWorks(ctx, db, mbid); err != nil {
		return nil, err
	}
	if rec.Releases, err = s.recordingReleases(ctx, db, mbid); err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *Store) recordingWorks(ctx context.Context, db *sql.DB, mbid string) ([]RecordingWork, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT w.mbid, w.title, COALESCE(w.type, ''), rw.type, rw.attributes
		 FROM recording_works rw
		 JOIN works w ON w.mbid = rw.work_mbid
		 WHERE rw.recording_mbid = ?
		 ORDER BY w.title, rw.type`, mbid)
	if err != nil {
		return nil, fmt.Errorf("recording works %s: %w", mbid, err)
	}
	defer rows.Close()

	works := make([]RecordingWork, 0, 2)
	for rows.Next() {
		var w RecordingWork
		if err := rows.Scan(&w.MBID, &w.Title, &w.Type, &w.Relationship, &w.Attributes); err != nil {
			return nil, err
		}
		works = append(works, w)
	}
	return works, rows.Err()
}

func (s *Store) recordingReleases(ctx context.Context, db *sql.DB, mbid string) ([]RecordingRelease, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT r.mbid, r.title, COALESCE(r.status, ''), COALESCE(r.date, ''), COALESCE(r.country, ''),
		        COALESCE(r.release_group_mbid, ''), COALESCE(rg.title, ''), COALESCE(rg.primary_type, '')
		 FROM tracks t
		 JOIN releases r ON r.mbid = t.release_mbid
		 LEFT JOIN release_groups rg ON rg.mbid = r.release_group_mbid
		 WHERE t.recording_mbid = ?
		 GROUP BY r.mbid
		 ORDER BY (r.date IS NULL OR r.date = ''), r.date, r.mbid`, mbid)
	if err != nil {
		return nil, fmt.Errorf("recording releases %s: %w", mbid, err)
	}
	defer rows.Close()

	releases := make([]RecordingRelease, 0, 4)
	for rows.Next() {
		var r RecordingRelease
		var rgMBID, rgTitle, rgType string
		if err := rows.Scan(&r.MBID, &r.Title, &r.Status, &r.Date, &r.Country, &rgMBID, &rgTitle, &rgType); err != nil {
			return nil, err
		}
		if rgMBID != "" {
			r.ReleaseGroup = &ReleaseGroupRef{MBID: rgMBID, Title: rgTitle, PrimaryType: rgType}
		}
		releases = append(releases, r)
	}
	return releases, rows.Err()
}

func (s *Store) Label(ctx context.Context, mbid string) (*Label, error) {
	db := s.DB()
	l := &Label{MBID: mbid}
	var typ, country, begin, end, areaMBID, areaName sql.NullString
	var labelCode sql.NullInt64
	var ended int
	err := db.QueryRowContext(ctx,
		`SELECT name, sort_name, disambiguation, type, label_code, country,
		        begin_date, end_date, ended, area_mbid, area_name
		 FROM labels WHERE mbid = ?`, mbid,
	).Scan(&l.Name, &l.SortName, &l.Disambiguation, &typ, &labelCode, &country,
		&begin, &end, &ended, &areaMBID, &areaName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("label %s: %w", mbid, err)
	}
	l.Type, l.Country = typ.String, country.String
	l.LabelCode = int(labelCode.Int64)
	l.BeginDate, l.EndDate = begin.String, end.String
	l.Ended = ended != 0
	l.Area = scanArea(areaMBID, areaName)

	if l.Aliases, err = queryAliases(ctx, db, "label_aliases", "label_mbid", mbid); err != nil {
		return nil, err
	}
	if l.Releases, err = s.labelReleases(ctx, db, mbid); err != nil {
		return nil, err
	}
	return l, nil
}

func (s *Store) labelReleases(ctx context.Context, db *sql.DB, mbid string) ([]LabelRelease, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT r.mbid, r.title, COALESCE(r.status, ''), COALESCE(r.date, ''), COALESCE(r.country, ''),
		        rl.catalog_number,
		        `+creditSubquery("release_artists", "release_mbid", "r.mbid")+`
		 FROM release_labels rl
		 JOIN releases r ON r.mbid = rl.release_mbid
		 WHERE rl.label_mbid = ?
		 ORDER BY (r.date IS NULL OR r.date = ''), r.date, r.title`, mbid)
	if err != nil {
		return nil, fmt.Errorf("label releases %s: %w", mbid, err)
	}
	defer rows.Close()

	releases := make([]LabelRelease, 0, 16)
	for rows.Next() {
		var r LabelRelease
		if err := rows.Scan(&r.MBID, &r.Title, &r.Status, &r.Date, &r.Country, &r.CatalogNumber, &r.ArtistCredit); err != nil {
			return nil, err
		}
		releases = append(releases, r)
	}
	return releases, rows.Err()
}

func (s *Store) Work(ctx context.Context, mbid string) (*Work, error) {
	db := s.DB()
	w := &Work{MBID: mbid}
	var typ sql.NullString
	err := db.QueryRowContext(ctx,
		`SELECT title, disambiguation, type, languages FROM works WHERE mbid = ?`, mbid,
	).Scan(&w.Title, &w.Disambiguation, &typ, &w.Languages)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("work %s: %w", mbid, err)
	}
	w.Type = typ.String

	w.ISWCs = make([]string, 0, 2)
	rows, err := db.QueryContext(ctx,
		`SELECT iswc FROM work_iswcs WHERE work_mbid = ? ORDER BY iswc`, mbid)
	if err != nil {
		return nil, fmt.Errorf("work iswcs %s: %w", mbid, err)
	}
	defer rows.Close()
	for rows.Next() {
		var iswc string
		if err := rows.Scan(&iswc); err != nil {
			return nil, err
		}
		w.ISWCs = append(w.ISWCs, iswc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if w.Recordings, err = s.workRecordings(ctx, db, mbid); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Store) workRecordings(ctx context.Context, db *sql.DB, mbid string) ([]WorkRecording, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT rec.mbid, rec.title, rw.type, rw.attributes,
		        `+creditSubquery("recording_artists", "recording_mbid", "rec.mbid")+`
		 FROM recording_works rw
		 JOIN recordings rec ON rec.mbid = rw.recording_mbid
		 WHERE rw.work_mbid = ?
		 ORDER BY rec.title, rec.mbid`, mbid)
	if err != nil {
		return nil, fmt.Errorf("work recordings %s: %w", mbid, err)
	}
	defer rows.Close()

	recordings := make([]WorkRecording, 0, 8)
	for rows.Next() {
		var r WorkRecording
		if err := rows.Scan(&r.MBID, &r.Title, &r.Relationship, &r.Attributes, &r.ArtistCredit); err != nil {
			return nil, err
		}
		recordings = append(recordings, r)
	}
	return recordings, rows.Err()
}
