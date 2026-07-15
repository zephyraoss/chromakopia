# chromakopia

audio fingerprinting, identification, and metadata catalog service backed by
local CKAF and mbforge datasets.<br/>
utilized within [oxygen](https://zephyra.lol/oxygen), our music streaming platform

chromakopia v2 is a Go service with two serving surfaces in one binary:

- **identify** — matches fpcalc fingerprints against a
  [chromaforge](../chromaforge)-built CKAF dataset with
  [libchroma](../libchroma) — no AcoustID API key, no rate limits, no network
  dependency for matching.
- **metadata** — a read-only HTTP-JSON catalog API (`/catalog/*`) over an
  [mbforge](../mbforge) metadata database, so consumers (the oxygen backend,
  identify nodes) never touch SQL directly.

> [!NOTE]
> The previous Bun/TypeScript implementation, which proxied api.acoustid.org,
> lives unmaintained in [legacy/](legacy/).

## Deployment model

Two Fyrastack nodes, one binary, different flags:

- **Node A — identify mode**: CKAF dataset on local disk, joins display
  metadata over HTTP from Node B via `--metadata-url http://node-b:3000`.
- **Node B — metadata mode**: `metadata.db` on local disk, serves
  `/catalog/*` to Node A and to the oxygen backend.

**sqld is no longer needed anywhere.** The metadata database is opened
directly as a local SQLite file (read-only), and every remote consumer goes
through the `/catalog` HTTP API instead of a SQL wire protocol.

The metadata DB is rebuilt monthly and deployed by replacing the file
wholesale. A running metadata node picks the new file up automatically (the
file is polled every 30s for changes) or immediately on `SIGHUP`.

For a single-box dev setup, `--mode both` serves everything from one
process, joining identify metadata from the local DB.

## Prerequisites

- Go 1.25+ (build only; the SQLite driver is pure Go — no CGO)
- fpcalc (Chromaprint tools) — identify mode
- a CKAF dataset built by `chromaforge build-ckaf`: `<prefix>.ckd`,
  `<prefix>.cki`, and `<prefix>.ckm` — identify mode
- an mbforge metadata database file — metadata mode (optional for identify)

Install fpcalc:

```bash
# macOS
brew install chromaprint

# Ubuntu/Debian
sudo apt-get install libchromaprint-tools

# Fedora
sudo dnf install chromaprint
```

## Build

```bash
go build ./cmd/chromakopia
```

## Run

```bash
# Node A: identify, joining metadata from Node B
./chromakopia --mode identify \
  --dataset /data/ckaf/acoustid \
  --metadata-url http://node-b:3000

# Node B: metadata catalog
./chromakopia --mode metadata \
  --metadata-db /data/mbforge/metadata.db

# single box: both surfaces, local joins
./chromakopia --mode both \
  --dataset /data/ckaf/acoustid \
  --metadata-db /data/mbforge/metadata.db
```

| Flag | Env | Default | Description |
|------|-----|---------|-------------|
| `--mode` | `CHROMAKOPIA_MODE` | `identify` | serving mode: `identify`, `metadata`, or `both` |
| `--dataset` | `CHROMAKOPIA_DATASET` | (required for identify) | CKAF dataset prefix (path without `.ckd`/`.cki`/`.ckm` extension) |
| `--metadata-db` | `CHROMAKOPIA_METADATA_DB` | (required for metadata) | mbforge metadata DB file (local path, opened read-only) |
| `--metadata-url` | `CHROMAKOPIA_METADATA_URL` | (none) | base URL of a chromakopia metadata-mode node to join identify metadata from |
| `--listen` | `CHROMAKOPIA_LISTEN` | `:3000` | listen address |
| `--fpcalc` | `CHROMAKOPIA_FPCALC` | `fpcalc` | fpcalc executable path |

In identify mode the metadata source is optional and failures degrade
gracefully: if the metadata DB is missing or the remote node is unreachable,
identification still works and matches carry MBIDs with placeholder
title/artist. The dataset files are required. In metadata mode the metadata
DB is the point, so an unusable file fails startup.

## Identify API

### POST /identify/file

Identify an audio file by uploading it.

**Request:** multipart/form-data

| Parameter | Type | Description |
|-----------|------|-------------|
| file | File | Audio file to identify |

**Response:**

```json
{
  "status": "ok",
  "matches": [
    {
      "id": "musicbrainz-recording-mbid",
      "score": 1,
      "recording": {
        "title": "Song Title",
        "artist": "Artist Name",
        "album": "Album Name",
        "duration": 216
      }
    }
  ]
}
```

`id` is the MusicBrainz recording ID the matched fingerprints collapse to
(v1 returned the AcoustID track ID here). `score` is exact-alignment
coverage in 0–1: the fraction of the query's alignable sub-fingerprints that
voted for the match, given the dataset's posting-index sampling stride.
`recording` is joined through the catalog layer (local DB or remote node):
`album` is the release-group title of the earliest release the recording
appears on.

**Example:**

```bash
curl -X POST -F "file=@song.mp3" http://localhost:3000/identify/file
```

### POST /identify/url

Identify an audio file from a URL.

**Request:** application/json

| Parameter | Type | Description |
|-----------|------|-------------|
| url | string | URL to audio file |

**Response:** Same as /identify/file

**Example:**

```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/song.mp3"}' \
  http://localhost:3000/identify/url
```

### POST /identify/fingerprint

Identify a fingerprint you already computed — mainly for testing. Takes
`fpcalc -raw` output.

**Request:** application/json

| Parameter | Type | Description |
|-----------|------|-------------|
| fingerprint | string | comma-separated raw sub-fingerprints (the `FINGERPRINT=` line of `fpcalc -raw`) |
| duration | number | audio duration in seconds (optional) |

**Response:** Same as /identify/file

**Example:**

```bash
fp=$(fpcalc -raw song.mp3 | sed -n 's/^FINGERPRINT=//p')
curl -X POST -H "Content-Type: application/json" \
  -d "{\"fingerprint\": \"$fp\"}" \
  http://localhost:3000/identify/fingerprint
```

## Catalog API

Read-only lookups against the mbforge metadata database. MBIDs are
case-insensitive. Unknown MBIDs return `404` with
`{"status": "ERROR", "error": "<entity> not found"}`. The JSON shapes are
chromakopia's own — deliberately more modest than the MusicBrainz ws/2 API.

### GET /catalog/artist/:mbid

Artist with aliases, tags, genres, artist relationships (band members,
collaborations), and release groups.

```json
{
  "mbid": "a0000000-…", "name": "Alpha", "sortName": "Alpha",
  "type": "Group", "country": "GB", "beginDate": "1990", "ended": false,
  "area": {"mbid": "f0000000-…", "name": "United Kingdom"},
  "aliases": [{"name": "The Alphas", "sortName": "Alphas, The", "type": "Artist name", "primary": true}],
  "tags": [{"name": "rock", "count": 5}],
  "genres": [{"mbid": "f0000000-…", "name": "rock", "count": 5}],
  "relationships": [
    {"artistMbid": "a0000000-…", "artistName": "Beta", "type": "member of band",
     "direction": "backward", "beginDate": "1990"}
  ],
  "releaseGroups": [
    {"mbid": "d0000000-…", "title": "Test Album", "primaryType": "Album",
     "firstReleaseDate": "2001-01-01", "artistCredit": "Alpha"}
  ]
}
```

### GET /catalog/release-group/:mbid

Release group with its releases (earliest first).

```json
{
  "mbid": "d0000000-…", "title": "Test Album", "primaryType": "Album",
  "secondaryTypes": ["Compilation"], "firstReleaseDate": "2001-01-01",
  "artists": [{"mbid": "a0000000-…", "name": "Alpha"}], "artistCredit": "Alpha",
  "releases": [
    {"mbid": "d0000000-…", "title": "Test Album (2001 press)", "status": "Official",
     "date": "2001-01-01", "country": "GB", "trackCount": 12}
  ]
}
```

### GET /catalog/release/:mbid

Release with labels, media, and tracks (each track carries its recording
MBID).

```json
{
  "mbid": "d0000000-…", "title": "Test Album (2001 press)", "status": "Official",
  "date": "2001-01-01", "country": "GB", "barcode": "1234567890123",
  "artists": [{"mbid": "a0000000-…", "name": "Alpha"}], "artistCredit": "Alpha",
  "releaseGroup": {"mbid": "d0000000-…", "title": "Test Album", "primaryType": "Album"},
  "labels": [{"mbid": "b0000000-…", "name": "Fern Records", "catalogNumber": "FERN-001"}],
  "media": [{"position": 1, "format": "CD", "trackCount": 12}],
  "tracks": [
    {"mbid": "e0000000-…", "mediaPosition": 1, "position": 1, "number": "1",
     "title": "Test Song", "lengthMs": 100000, "recordingMbid": "11111111-…"}
  ]
}
```

### GET /catalog/recording/:mbid

Recording with artist credits, ISRCs, the works it embodies, and the
releases it appears on. `releases` is ordered earliest-dated first (undated
last), so the first entry is the stable "album" pick — the identify join
uses exactly this.

```json
{
  "mbid": "11111111-…", "title": "Test Song", "lengthMs": 100000,
  "firstReleaseDate": "2001-01-01",
  "artists": [
    {"mbid": "a0000000-…", "name": "Alpha", "joinPhrase": " feat. "},
    {"mbid": "a0000000-…", "name": "Beta"}
  ],
  "artistCredit": "Alpha feat. Beta",
  "isrcs": ["GBAAA0100001"],
  "works": [{"mbid": "c0000000-…", "title": "Test Song", "type": "Song", "relationship": "performance"}],
  "releases": [
    {"mbid": "d0000000-…", "title": "Test Album (2001 press)", "status": "Official",
     "date": "2001-01-01", "country": "GB",
     "releaseGroup": {"mbid": "d0000000-…", "title": "Test Album", "primaryType": "Album"}}
  ]
}
```

### GET /catalog/label/:mbid

Label with aliases and the releases published on it (with catalog numbers).

```json
{
  "mbid": "b0000000-…", "name": "Fern Records", "sortName": "Fern Records",
  "type": "Original Production", "labelCode": 123, "country": "GB", "ended": false,
  "aliases": [{"name": "Fern", "sortName": "Fern"}],
  "releases": [
    {"mbid": "d0000000-…", "title": "Test Album (2001 press)", "date": "2001-01-01",
     "country": "GB", "catalogNumber": "FERN-001", "artistCredit": "Alpha"}
  ]
}
```

### GET /catalog/work/:mbid

Work with ISWCs and the recordings of it.

```json
{
  "mbid": "c0000000-…", "title": "Test Song", "type": "Song", "languages": "eng",
  "iswcs": ["T-123.456.789-0"],
  "recordings": [
    {"mbid": "11111111-…", "title": "Test Song", "artistCredit": "Alpha feat. Beta",
     "relationship": "performance"}
  ]
}
```

### GET /catalog/search?q=&type=&limit=

Full-text search across artists, labels, works, release groups, releases,
recordings, and tracks — the same matching and ranking as `mbforge search`.
When the database carries the FTS index built by `mbforge search-index` the
query uses it (`"indexed": true`); otherwise the service falls back to slower
tiered LIKE queries (exact, prefix, substring — including alias, ISRC, and
barcode matches).

| Parameter | Description |
|-----------|-------------|
| q | query text (required) |
| type | optional filter: `artist`, `label`, `work`, `release_group` (or `release-group`), `release`, `recording`, `track` |
| limit | max results per entity type, default 10, max 50 |

Every result carries `type`, `mbid`, and `name`. After matching, each hit is
enriched from the base tables with display fields that depend on the type;
empty fields are omitted:

| Type | Extra fields |
|------|--------------|
| `recording`, `release`, `release_group`, `track` | `artist` (joined artist credit), `year` (release year) |
| `artist` | `disambiguation`, `artistType` |
| `label` | `disambiguation`, `country` |
| `work` | `workType` |

```json
{
  "status": "ok", "query": "test song", "indexed": true,
  "results": [
    {"type": "recording", "mbid": "11111111-…", "name": "Test Song",
     "artist": "Alpha feat. Beta", "year": 2001, "score": -1.82},
    {"type": "work", "mbid": "c0000000-…", "name": "Test Song",
     "workType": "Song", "score": -1.64}
  ]
}
```

`score` is bm25 (lower is better) and only present on the indexed path.

### GET /health

Health check, reporting whatever modes are enabled:

```json
{
  "status": "ok",
  "modes": ["identify", "metadata"],
  "records": 92600000,
  "metadataDb": "ok",
  "catalog": "ok"
}
```

`records` and `metadataDb` appear in identify mode (`metadataDb` is `"ok"`
when a local DB or remote node is configured, `"disabled"` otherwise);
`catalog` appears in metadata mode.

## Errors

Failures return `{"status": "ERROR", "error": "..."}` with an appropriate
4xx/5xx status code.

## Test

```bash
go test ./...
```

The integration tests build a miniature CKAF dataset with the libchroma
builders and a fixture metadata database using mbforge's schema
(internal/catalog/catalogtest), so they need neither fpcalc nor real data.
The identify-via-remote-metadata path is tested against a second in-process
chromakopia instance running in metadata mode.
