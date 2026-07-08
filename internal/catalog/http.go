package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const (
	searchDefaultLimit = 10
	searchMaxLimit     = 50
)

func (s *Store) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /catalog/artist/{mbid}", entityHandler(s, "artist", (*Store).Artist))
	mux.HandleFunc("GET /catalog/release-group/{mbid}", entityHandler(s, "release group", (*Store).ReleaseGroup))
	mux.HandleFunc("GET /catalog/release/{mbid}", entityHandler(s, "release", (*Store).Release))
	mux.HandleFunc("GET /catalog/recording/{mbid}", entityHandler(s, "recording", (*Store).Recording))
	mux.HandleFunc("GET /catalog/label/{mbid}", entityHandler(s, "label", (*Store).Label))
	mux.HandleFunc("GET /catalog/work/{mbid}", entityHandler(s, "work", (*Store).Work))
	mux.HandleFunc("GET /catalog/search", s.handleSearch)
}

func entityHandler[T any](s *Store, kind string, lookup func(*Store, context.Context, string) (*T, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mbid := strings.ToLower(r.PathValue("mbid"))
		entity, err := lookup(s, r.Context(), mbid)
		if err != nil {
			log.Printf("catalog %s %s: %v", kind, mbid, err)
			writeError(w, http.StatusInternalServerError, "metadata lookup failed")
			return
		}
		if entity == nil {
			writeError(w, http.StatusNotFound, kind+" not found")
			return
		}
		writeJSON(w, http.StatusOK, entity)
	}
}

type searchResponse struct {
	Status  string         `json:"status"`
	Query   string         `json:"query"`
	Indexed bool           `json:"indexed"`
	Results []SearchResult `json:"results"`
}

func (s *Store) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeError(w, http.StatusBadRequest, "missing query parameter q")
		return
	}

	entityType := strings.ReplaceAll(r.URL.Query().Get("type"), "-", "_")
	if entityType != "" && !ValidSearchType(entityType) {
		writeError(w, http.StatusBadRequest, "unknown type: "+entityType)
		return
	}

	limit := searchDefaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = min(n, searchMaxLimit)
	}

	results, indexed, err := s.Search(r.Context(), q, entityType, limit)
	if err != nil {
		log.Printf("catalog search %q: %v", q, err)
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}
	writeJSON(w, http.StatusOK, searchResponse{Status: "ok", Query: q, Indexed: indexed, Results: results})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil && !errors.Is(err, http.ErrHandlerTimeout) {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"status": "ERROR", "error": msg})
}
