package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	chroma "github.com/zephyraoss/libchroma"

	"github.com/zephyraoss/chromakopia/internal/catalog"
	"github.com/zephyraoss/chromakopia/internal/dataset"
	"github.com/zephyraoss/chromakopia/internal/fpcalc"
)

const (
	queryMinHits = 3
	queryTopK    = 20

	maxUploadBytes = 256 << 20

	maxPostingOrdinals = 256
)

type Server struct {
	Dataset  *dataset.Dataset
	Metadata catalog.RecordingSource
	Fpcalc   *fpcalc.Runner
	TempDir  string
	Download *http.Client
}

type Recording struct {
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album,omitempty"`
	Duration int    `json:"duration"`
}

type Match struct {
	ID        string    `json:"id"`
	Score     float64   `json:"score"`
	Recording Recording `json:"recording"`
}

type identifyResponse struct {
	Status  string  `json:"status"`
	Matches []Match `json:"matches"`
}

type errorResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

func (s *Server) Handler() http.Handler {
	return Routes(s, nil)
}

func Routes(identify *Server, cat *catalog.Store) http.Handler {
	mux := http.NewServeMux()
	modes := make([]string, 0, 2)
	if identify != nil {
		identify.register(mux)
		modes = append(modes, "identify")
	}
	if cat != nil {
		cat.Register(mux)
		modes = append(modes, "metadata")
	}
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		body := map[string]any{"status": "ok", "modes": modes}
		if identify != nil {
			body["records"] = identify.Dataset.RecordCount()
			metaStatus := "disabled"
			if identify.Metadata != nil {
				metaStatus = "ok"
			}
			body["metadataDb"] = metaStatus
		}
		if cat != nil {
			body["catalog"] = "ok"
		}
		writeJSON(w, http.StatusOK, body)
	})
	return mux
}

func (s *Server) register(mux *http.ServeMux) {
	mux.HandleFunc("POST /identify/file", s.handleIdentifyFile)
	mux.HandleFunc("POST /identify/url", s.handleIdentifyURL)
	mux.HandleFunc("POST /identify/fingerprint", s.handleIdentifyFingerprint)
}

func (s *Server) handleIdentifyFile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "No file provided")
		return
	}
	defer file.Close()
	if header.Filename == "" {
		writeError(w, http.StatusBadRequest, "No file provided")
		return
	}

	tempPath, err := s.saveTemp(file, header.Filename)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer os.Remove(tempPath)

	s.identifyFile(w, r.Context(), tempPath)
}

func (s *Server) handleIdentifyURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil || req.URL == "" {
		writeError(w, http.StatusBadRequest, "No URL provided")
		return
	}

	tempPath, err := s.downloadTemp(r.Context(), req.URL)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer os.Remove(tempPath)

	s.identifyFile(w, r.Context(), tempPath)
}

func (s *Server) handleIdentifyFingerprint(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Fingerprint string `json:"fingerprint"`
		Duration    int    `json:"duration"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 16<<20)).Decode(&req); err != nil || req.Fingerprint == "" {
		writeError(w, http.StatusBadRequest, "No fingerprint provided")
		return
	}

	values, err := fpcalc.ParseRawFingerprint(req.Fingerprint)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.identify(w, r.Context(), values)
}

func (s *Server) identifyFile(w http.ResponseWriter, ctx context.Context, path string) {
	result, err := s.Fpcalc.Run(ctx, path)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	s.identify(w, ctx, result.Values)
}

func (s *Server) identify(w http.ResponseWriter, ctx context.Context, values []uint32) {
	hits, err := s.Dataset.QueryFull(values, &chroma.PostingQueryOptions{
		MinHits: queryMinHits,
		TopK:    queryTopK,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	matches, err := s.collapseHits(ctx, hits, len(values))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, identifyResponse{Status: "ok", Matches: matches})
}

func (s *Server) collapseHits(ctx context.Context, hits []chroma.PostingHit, queryLen int) ([]Match, error) {
	matches := make([]Match, 0, len(hits))
	seen := make(map[uuid.UUID]bool, len(hits))

	for _, hit := range hits {
		resolved, err := s.Dataset.Resolve(hit.FingerprintID)
		if err != nil {
			return nil, err
		}
		if resolved.MBID == uuid.Nil || seen[resolved.MBID] {
			continue
		}
		seen[resolved.MBID] = true

		match := Match{
			ID:    resolved.MBID.String(),
			Score: coverageScore(hit.Hits, queryLen, s.Dataset.Stride()),
			Recording: Recording{
				Title:    "Unknown Title",
				Artist:   "Unknown Artist",
				Duration: int((resolved.DurationMs + 500) / 1000),
			},
		}
		if resolved.Embedded != nil {
			applyEmbedded(&match.Recording, resolved.Embedded)
		}
		s.applyDisplayMetadata(ctx, &match.Recording, resolved.MBID)
		matches = append(matches, match)
	}
	return matches, nil
}

func (s *Server) applyDisplayMetadata(ctx context.Context, rec *Recording, mbid uuid.UUID) {
	if s.Metadata == nil {
		return
	}
	meta, err := s.Metadata.RecordingSummary(ctx, mbid.String())
	if err != nil {
		log.Printf("metadata lookup %s failed: %v", mbid, err)
		return
	}
	if meta == nil {
		return
	}
	if meta.Title != "" {
		rec.Title = meta.Title
	}
	if meta.Artist != "" {
		rec.Artist = meta.Artist
	}
	if meta.Album != "" {
		rec.Album = meta.Album
	}
	if meta.DurationMs > 0 {
		rec.Duration = int((meta.DurationMs + 500) / 1000)
	}
}

func applyEmbedded(rec *Recording, meta *chroma.TrackMetadata) {
	if meta.Title != "" {
		rec.Title = meta.Title
	}
	if meta.Artist != "" {
		rec.Artist = meta.Artist
	}
	if meta.Release != "" {
		rec.Album = meta.Release
	}
}

func coverageScore(hits, queryLen, stride int) float64 {
	if queryLen == 0 || stride <= 0 {
		return 0
	}
	maxHits := (queryLen + stride - 1) / stride
	if maxHits > maxPostingOrdinals {
		maxHits = maxPostingOrdinals
	}
	score := float64(hits) / float64(maxHits)
	if score > 1 {
		score = 1
	}
	return float64(int(score*100+0.5)) / 100
}

func (s *Server) saveTemp(r io.Reader, filename string) (string, error) {
	f, err := os.CreateTemp(s.TempDir, "chromakopia-*"+filepath.Ext(filename))
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("write temp file: %w", err)
	}
	return f.Name(), nil
}

func (s *Server) downloadTemp(ctx context.Context, url string) (string, error) {
	client := s.Download
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return s.saveTemp(io.LimitReader(resp.Body, maxUploadBytes), filepath.Base(req.URL.Path))
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil && !errors.Is(err, http.ErrHandlerTimeout) {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Status: "ERROR", Error: msg})
}
