package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
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
		return results, true, err
	}
	results, err = searchSlow(ctx, db, query, entityType, limit)
	return results, false, err
}

type SearchResult struct {
	Type   string  `json:"type"`
	MBID   string  `json:"mbid"`
	Name   string  `json:"name"`
	Detail string  `json:"detail,omitempty"`
	Meta   string  `json:"meta,omitempty"`
	Aux    string  `json:"aux,omitempty"`
	Score  float64 `json:"score,omitempty"`
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
SELECT entity_type, entity_mbid, heading, subtitle, meta, aux,
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
		if err := rows.Scan(&item.Type, &item.MBID, &item.Name, &item.Detail, &item.Meta, &item.Aux, &item.Score); err != nil {
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
	scan      func(*sql.Rows) (SearchResult, error)
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
			item, err := e.scan(rows)
			if err != nil {
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

func joinNonEmpty(sep string, parts ...string) string {
	kept := parts[:0:0]
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
}

func scanNamed(rows *sql.Rows) (SearchResult, error) {
	var item SearchResult
	var sortName, typ, country string
	if err := rows.Scan(&item.MBID, &item.Name, &sortName, &typ, &country); err != nil {
		return item, err
	}
	item.Detail = sortName
	item.Meta = joinNonEmpty(" ", typ, country)
	return item, nil
}

func scanTitled(rows *sql.Rows) (SearchResult, error) {
	var item SearchResult
	var meta1, meta2 string
	if err := rows.Scan(&item.MBID, &item.Name, &item.Detail, &meta1, &meta2); err != nil {
		return item, err
	}
	item.Meta = joinNonEmpty(" ", meta1, meta2)
	return item, nil
}

var slowEntities = []slowEntity{
	{
		typ: "artist",
		selectSQL: `
SELECT a.mbid, a.name, COALESCE(a.sort_name, ''), COALESCE(a.type, ''), COALESCE(a.country, '')
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
		scan:     scanNamed,
	},
	{
		typ: "label",
		selectSQL: `
SELECT l.mbid, l.name, COALESCE(l.sort_name, ''), COALESCE(l.type, ''), COALESCE(l.country, '')
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
		scan:     scanNamed,
	},
	{
		typ: "work",
		selectSQL: `
SELECT w.mbid, w.title, '', COALESCE(w.type, ''), ''
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
		scan:     scanTitled,
	},
	{
		typ: "release_group",
		selectSQL: `
SELECT rg.mbid, rg.title,
    ` + "COALESCE((SELECT group_concat(piece, '') FROM (SELECT rga.artist_name || rga.join_phrase AS piece FROM release_group_artists rga WHERE rga.release_group_mbid = rg.mbid ORDER BY rga.position)), '')" + `,
    COALESCE(rg.primary_type, ''), COALESCE(rg.first_release_date, '')
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
		scan:     scanTitled,
	},
	{
		typ: "release",
		selectSQL: `
SELECT r.mbid, r.title,
    ` + "COALESCE((SELECT group_concat(piece, '') FROM (SELECT ra.artist_name || ra.join_phrase AS piece FROM release_artists ra WHERE ra.release_mbid = r.mbid ORDER BY ra.position)), '')" + `,
    COALESCE(r.date, ''), COALESCE(r.country, '')
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
		scan:     scanTitled,
	},
	{
		typ: "recording",
		selectSQL: `
SELECT r.mbid, r.title,
    ` + "COALESCE((SELECT group_concat(piece, '') FROM (SELECT ra.artist_name || ra.join_phrase AS piece FROM recording_artists ra WHERE ra.recording_mbid = r.mbid ORDER BY ra.position)), '')" + `,
    COALESCE(r.first_release_date, ''), ''
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
		scan:     scanTitled,
	},
	{
		typ: "track",
		selectSQL: `
SELECT t.mbid, t.title,
    ` + "COALESCE((SELECT group_concat(piece, '') FROM (SELECT ra.artist_name || ra.join_phrase AS piece FROM release_artists ra WHERE ra.release_mbid = t.release_mbid ORDER BY ra.position)), '')" + `,
    t.number, COALESCE(r.title, '')
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
		scan:     scanTitled,
	},
}
