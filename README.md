# chromakopia

audio fingerprinting and identification service using fpcalc and AcoustID.<br/>
utilized within [oxygen](https://zephyra.lol/oxygen), our music streaming platform

## Prerequisites

- Bun runtime
- fpcalc (Chromaprint tools)

Install fpcalc:

```bash
# macOS
brew install chromaprint

# Ubuntu/Debian
sudo apt-get install libchromaprint-tools

# Fedora
sudo dnf install chromaprint
```

## Setup

Create a `.env` file with your AcoustID API key:

```bash
ACOUSTID_API_KEY=your_api_key_here
```

Get an API key at https://acoustid.org/new-application

## Install

```bash
bun install
```

## Run

```bash
bun run src/index.ts
```

Server runs on http://localhost:3000

## API Endpoints

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
      "id": "acoustid-track-id",
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

### GET /health

Health check endpoint.

**Response:**

```json
{
  "status": "ok",
  "queueLength": 0
}
```

## Rate Limiting

Requests are queued and processed at 3 requests per second (AcoustID API rate limit).
