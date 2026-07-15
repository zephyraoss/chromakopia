package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func (s *Store) Search(ctx context.Context, query, entityType string, limit int) (results []SearchResult, indexed bool, err error) {
	db := s.DB()
	hasIndex, err := searchIndexExists(ctx, db)
	if err != nil {
		return nil, false, err
	}
	if hasIndex {
		results, err = searchFast(ctx, db, query, entityType, limit)
	} else {
		results, err = searchSlow(ctx, db, query, entityType, limit)
	}
	if err != nil {
		return nil, hasIndex, err
	}
	if err := enrichResults(ctx, db, results); err != nil {
		return nil, hasIndex, err
	}
	return results, hasIndex, nil
}

type SearchResult struct {
	Type           string  `json:"type"`
	MBID           string  `json:"mbid"`
	Name           string  `json:"name"`
	Artist         string  `json:"artist,omitempty"`
	Year           int     `json:"year,omitempty"`
	Disambiguation string  `json:"disambiguation,omitempty"`
	ArtistType     string  `json:"artistType,omitempty"`
	WorkType       string  `json:"workType,omitempty"`
	Country        string  `json:"country,omitempty"`
	Score          float64 `json:"score,omitempty"`
}

var searchEntityTypes = []string{"artist", "label", "work", "release_group", "release", "recording", "track"}

func ValidSearchType(t string) bool {
	for _, known := range searchEntityTypes {
		if t == known {
			return true
		}
	}
	return false
}

type enrichSpec struct {
	sql  string
	scan func(*sql.Row, *SearchResult) error
}

func scanArtistYear(row *sql.Row, item *SearchResult) error {
	var artist, date string
	if err := row.Scan(&artist, &date); err != nil {
		return err
	}
	item.Artist = strings.TrimSpace(artist)
	item.Year = dateYear(date)
	return nil
}

func dateYear(date string) int {
	if len(date) < 4 {
		return 0
	}
	year, err := strconv.Atoi(date[:4])
	if err != nil {
		return 0
	}
	return year
}

var enrichSpecs = map[string]enrichSpec{
	"artist": {
		sql: `SELECT disambiguation, COALESCE(type, '') FROM artists WHERE mbid = ?`,
		scan: func(row *sql.Row, item *SearchResult) error {
			return row.Scan(&item.Disambiguation, &item.ArtistType)
		},
	},
	"label": {
		sql: `SELECT disambiguation, COALESCE(country, '') FROM labels WHERE mbid = ?`,
		scan: func(row *sql.Row, item *SearchResult) error {
			return row.Scan(&item.Disambiguation, &item.Country)
		},
	},
	"work": {
		sql: `SELECT COALESCE(type, '') FROM works WHERE mbid = ?`,
		scan: func(row *sql.Row, item *SearchResult) error {
			return row.Scan(&item.WorkType)
		},
	},
	"release_group": {
		sql: `SELECT ` + creditSubquery("release_group_artists", "release_group_mbid", "rg.mbid") + `,
		       COALESCE(rg.first_release_date, '')
		 FROM release_groups rg WHERE rg.mbid = ?`,
		scan: scanArtistYear,
	},
	"release": {
		sql: `SELECT ` + creditSubquery("release_artists", "release_mbid", "r.mbid") + `,
		       COALESCE(r.date, '')
		 FROM releases r WHERE r.mbid = ?`,
		scan: scanArtistYear,
	},
	"recording": {
		sql: `SELECT ` + creditSubquery("recording_artists", "recording_mbid", "r.mbid") + `,
		       COALESCE(r.first_release_date, '')
		 FROM recordings r WHERE r.mbid = ?`,
		scan: scanArtistYear,
	},
	"track": {
		sql: `SELECT ` + creditSubquery("release_artists", "release_mbid", "t.release_mbid") + `,
		       COALESCE(r.date, '')
		 FROM tracks t JOIN releases r ON r.mbid = t.release_mbid WHERE t.mbid = ?`,
		scan: scanArtistYear,
	},
}

func enrichResults(ctx context.Context, db *sql.DB, results []SearchResult) error {
	stmts := make(map[string]*sql.Stmt, len(enrichSpecs))
	defer func() {
		for _, stmt := range stmts {
			stmt.Close()
		}
	}()

	for i := range results {
		item := &results[i]
		spec, ok := enrichSpecs[item.Type]
		if !ok {
			continue
		}
		stmt := stmts[item.Type]
		if stmt == nil {
			var err error
			if stmt, err = db.PrepareContext(ctx, spec.sql); err != nil {
				return fmt.Errorf("enrich %s: %w", item.Type, err)
			}
			stmts[item.Type] = stmt
		}
		err := spec.scan(stmt.QueryRowContext(ctx, item.MBID), item)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("enrich %s %s: %w", item.Type, item.MBID, err)
		}
	}
	return nil
}

func searchIndexExists(ctx context.Context, db *sql.DB) (bool, error) {
	var found string
	err := db.QueryRowContext(ctx,
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'search_fts'`).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return found == "search_fts", nil
}

var searchTokenRE = regexp.MustCompile(`[\p{L}\p{N}]+`)

func buildFTSQuery(query string) (string, bool) {
	tokens := searchTokenRE.FindAllString(strings.ToLower(query), -1)
	if len(tokens) == 0 {
		return "", false
	}
	parts := make([]string, 0, len(tokens))
	for _, token := range tokens {
		parts = append(parts, token+"*")
	}
	return strings.Join(parts, " AND "), true
}

func searchFast(ctx context.Context, db *sql.DB, query, entityType string, limit int) ([]SearchResult, error) {
	matchQuery, ok := buildFTSQuery(query)
	if !ok {
		return []SearchResult{}, nil
	}

	sql := `
SELECT entity_type, entity_mbid, heading,
       bm25(search_fts, 8.0, 4.0, 2.0, 1.0) AS score
FROM search_fts
WHERE search_fts MATCH ?`
	args := []any{matchQuery}
	fetchLimit := limit
	if entityType != "" {
		sql += ` AND entity_type = ?`
		args = append(args, entityType)
	} else {
		fetchLimit = limit * 8
		if fetchLimit < 50 {
			fetchLimit = 50
		}
	}
	sql += `
ORDER BY score, entity_type, heading
LIMIT ?`
	args = append(args, fetchLimit)

	rows, err := db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("search index query: %w", err)
	}
	defer rows.Close()

	typeCounts := make(map[string]int, len(searchEntityTypes))
	out := make([]SearchResult, 0, limit)
	for rows.Next() {
		var item SearchResult
		if err := rows.Scan(&item.Type, &item.MBID, &item.Name, &item.Score); err != nil {
			return nil, err
		}
		if typeCounts[item.Type] >= limit {
			continue
		}
		typeCounts[item.Type]++
		out = append(out, item)
	}
	return out, rows.Err()
}

type slowEntity struct {
	typ       string
	selectSQL string
	exact     string
	exactArgs int
	like      string
	likeArgs  int
}

func searchSlow(ctx context.Context, db *sql.DB, query, entityType string, limit int) ([]SearchResult, error) {
	out := make([]SearchResult, 0, limit)
	for _, e := range slowEntities {
		if entityType != "" && e.typ != entityType {
			continue
		}
		results, err := searchSlowEntity(ctx, db, e, query, limit)
		if err != nil {
			return nil, err
		}
		out = append(out, results...)
	}
	return out, nil
}

func searchSlowEntity(ctx context.Context, db *sql.DB, e slowEntity, query string, limit int) ([]SearchResult, error) {
	seen := make(map[string]struct{}, limit)
	results := make([]SearchResult, 0, limit)

	run := func(where, pattern string, argCount int) error {
		args := make([]any, 0, argCount+1)
		for i := 0; i < argCount; i++ {
			args = append(args, pattern)
		}
		args = append(args, limit)
		rows, err := db.QueryContext(ctx, fmt.Sprintf(e.selectSQL, where), args...)
		if err != nil {
			return fmt.Errorf("slow search %s: %w", e.typ, err)
		}
		defer rows.Close()
		for rows.Next() {
			var item SearchResult
			if err := rows.Scan(&item.MBID, &item.Name); err != nil {
				return err
			}
			if _, dup := seen[item.MBID]; dup {
				continue
			}
			seen[item.MBID] = struct{}{}
			item.Type = e.typ
			results = append(results, item)
			if len(results) >= limit {
				return nil
			}
		}
		return rows.Err()
	}

	if err := run(e.exact, query, e.exactArgs); err != nil {
		return nil, err
	}
	if len(results) >= limit {
		return results, nil
	}
	if err := run(e.like, query+"%", e.likeArgs); err != nil {
		return nil, err
	}
	if len(results) >= limit {
		return results, nil
	}
	if err := run(e.like, "%"+query+"%", e.likeArgs); err != nil {
		return nil, err
	}
	return results, nil
}

var slowEntities = []slowEntity{
	{
		typ: "artist",
		selectSQL: `
SELECT a.mbid, a.name
FROM artists a
WHERE %s
ORDER BY a.name
LIMIT ?`,
		exact: `a.mbid = ?
   OR a.name LIKE ?
   OR a.sort_name LIKE ?
   OR EXISTS (SELECT 1 FROM artist_aliases aa WHERE aa.artist_mbid = a.mbid AND aa.name LIKE ?)`,
		exactArgs: 4,
		like: `a.name LIKE ?
   OR a.sort_name LIKE ?
   OR EXISTS (SELECT 1 FROM artist_aliases aa WHERE aa.artist_mbid = a.mbid AND aa.name LIKE ?)`,
		likeArgs: 3,
	},
	{
		typ: "label",
		selectSQL: `
SELECT l.mbid, l.name
FROM labels l
WHERE %s
ORDER BY l.name
LIMIT ?`,
		exact: `l.mbid = ?
   OR l.name LIKE ?
   OR l.sort_name LIKE ?
   OR EXISTS (SELECT 1 FROM label_aliases la WHERE la.label_mbid = l.mbid AND la.name LIKE ?)`,
		exactArgs: 4,
		like: `l.name LIKE ?
   OR l.sort_name LIKE ?
   OR EXISTS (SELECT 1 FROM label_aliases la WHERE la.label_mbid = l.mbid AND la.name LIKE ?)`,
		likeArgs: 3,
	},
	{
		typ: "work",
		selectSQL: `
SELECT w.mbid, w.title
FROM works w
WHERE %s
ORDER BY w.title
LIMIT ?`,
		exact: `w.mbid = ?
   OR w.title LIKE ?
   OR EXISTS (SELECT 1 FROM work_iswcs wi WHERE wi.work_mbid = w.mbid AND wi.iswc = ?)
   OR EXISTS (SELECT 1 FROM work_aliases wa WHERE wa.work_mbid = w.mbid AND wa.name LIKE ?)`,
		exactArgs: 4,
		like: `w.title LIKE ?
   OR EXISTS (SELECT 1 FROM work_aliases wa WHERE wa.work_mbid = w.mbid AND wa.name LIKE ?)`,
		likeArgs: 2,
	},
	{
		typ: "release_group",
		selectSQL: `
SELECT rg.mbid, rg.title
FROM release_groups rg
WHERE %s
ORDER BY rg.first_release_date DESC, rg.title
LIMIT ?`,
		exact: `rg.mbid = ?
   OR rg.title LIKE ?
   OR EXISTS (SELECT 1 FROM release_group_artists rga WHERE rga.release_group_mbid = rg.mbid AND rga.artist_name LIKE ?)`,
		exactArgs: 3,
		like: `rg.title LIKE ?
   OR EXISTS (SELECT 1 FROM release_group_artists rga WHERE rga.release_group_mbid = rg.mbid AND rga.artist_name LIKE ?)`,
		likeArgs: 2,
	},
	{
		typ: "release",
		selectSQL: `
SELECT r.mbid, r.title
FROM releases r
WHERE %s
ORDER BY r.date DESC, r.title
LIMIT ?`,
		exact: `r.mbid = ?
   OR r.title LIKE ?
   OR r.barcode = ?
   OR EXISTS (SELECT 1 FROM release_artists ra WHERE ra.release_mbid = r.mbid AND ra.artist_name LIKE ?)
   OR EXISTS (SELECT 1 FROM release_labels rl WHERE rl.release_mbid = r.mbid AND rl.label_name LIKE ?)`,
		exactArgs: 5,
		like: `r.title LIKE ?
   OR EXISTS (SELECT 1 FROM release_artists ra WHERE ra.release_mbid = r.mbid AND ra.artist_name LIKE ?)
   OR EXISTS (SELECT 1 FROM release_labels rl WHERE rl.release_mbid = r.mbid AND rl.label_name LIKE ?)`,
		likeArgs: 3,
	},
	{
		typ: "recording",
		selectSQL: `
SELECT r.mbid, r.title
FROM recordings r
WHERE %s
ORDER BY r.first_release_date DESC, r.title
LIMIT ?`,
		exact: `r.mbid = ?
   OR r.title LIKE ?
   OR EXISTS (SELECT 1 FROM recording_artists ra WHERE ra.recording_mbid = r.mbid AND ra.artist_name LIKE ?)
   OR EXISTS (SELECT 1 FROM recording_isrcs ri WHERE ri.recording_mbid = r.mbid AND ri.isrc = ?)
   OR EXISTS (SELECT 1 FROM tracks t WHERE t.recording_mbid = r.mbid AND t.title LIKE ?)`,
		exactArgs: 5,
		like: `r.title LIKE ?
   OR EXISTS (SELECT 1 FROM recording_artists ra WHERE ra.recording_mbid = r.mbid AND ra.artist_name LIKE ?)
   OR EXISTS (SELECT 1 FROM tracks t WHERE t.recording_mbid = r.mbid AND t.title LIKE ?)`,
		likeArgs: 3,
	},
	{
		typ: "track",
		selectSQL: `
SELECT t.mbid, t.title
FROM tracks t
JOIN releases r ON r.mbid = t.release_mbid
WHERE %s
ORDER BY r.date DESC, t.title
LIMIT ?`,
		exact: `t.mbid = ?
   OR t.title LIKE ?
   OR EXISTS (SELECT 1 FROM release_artists ra WHERE ra.release_mbid = t.release_mbid AND ra.artist_name LIKE ?)`,
		exactArgs: 3,
		like: `t.title LIKE ?
   OR EXISTS (SELECT 1 FROM release_artists ra WHERE ra.release_mbid = t.release_mbid AND ra.artist_name LIKE ?)`,
		likeArgs: 2,
	},
}
