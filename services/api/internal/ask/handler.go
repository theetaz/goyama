// Package ask hosts the chat-agent retrieval endpoint POST /v1/ask.
//
// Design choice: the response is purely the *retrieved* chunks +
// authority chips + scores. No LLM-generated synthesis. CLAUDE.md is
// explicit ("Never let the LLM invent dosages, dates, or taxonomy")
// and the chunks themselves carry verbatim quotes that farmers can
// trust. A future PR may add an optional `synthesis: true` flag that
// asks an LLM to summarise the chunks with citations, but the
// retrieval surface stands on its own.
package ask

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/goyama/api/internal/embed"
	"github.com/goyama/api/internal/geo"
	"github.com/goyama/api/internal/knowledge"
	"github.com/goyama/api/internal/platform/httpx"
)

// ErrRequiresDatabase is returned when the API runs in JSONL mode —
// the retrieval pipe needs Postgres for the cosine join.
var ErrRequiresDatabase = errors.New("/v1/ask requires Postgres + embedded chunks (set DATABASE_URL and run cmd/embedchunks)")

// Handler answers POST /v1/ask. Holds the embedder so the same
// algorithm is used at backfill and at query time — otherwise the
// cosine math is meaningless.
type Handler struct {
	embedder embed.Embedder
	chunks   knowledge.SearchRepo
	geo      geo.Repository
}

// New returns an /v1/ask handler.
func New(embedder embed.Embedder, chunks knowledge.SearchRepo, geoRepo geo.Repository) *Handler {
	return &Handler{embedder: embedder, chunks: chunks, geo: geoRepo}
}

// Routes returns a chi sub-router mounted at /v1/ask.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.ask)
	return r
}

// AskRequest is the JSON body for POST /v1/ask.
type AskRequest struct {
	Question string   `json:"question"`
	CropSlug string   `json:"crop"`
	Lat      *float64 `json:"lat"`
	Lng      *float64 `json:"lng"`
	Country  string   `json:"country"`
	Limit    int      `json:"k"`
}

// AskResponse is what the chat agent renders.
type AskResponse struct {
	Question     string    `json:"question"`
	Hits         []HitView `json:"hits"`
	UsedCropSlug string    `json:"used_crop,omitempty"`
	UsedAEZCodes []string  `json:"used_aez_codes,omitempty"`
	UsedDistrict string    `json:"used_district,omitempty"`
	Embedder     string    `json:"embedder"`
	Count        int       `json:"count"`
	Disclaimer   string    `json:"disclaimer"`
}

// HitView is the per-hit projection sent to the client.
type HitView struct {
	Slug         string                `json:"slug"`
	Title        string                `json:"title,omitempty"`
	Body         string                `json:"body"`
	Quote        string                `json:"quote,omitempty"`
	Authority    string                `json:"authority"`
	Confidence   *float64              `json:"confidence,omitempty"`
	Score        float64               `json:"score"`
	Source       *knowledge.Source     `json:"source,omitempty"`
	EntityRefs   []knowledge.EntityRef `json:"entity_refs,omitempty"`
	TopicTags    []string              `json:"topic_tags,omitempty"`
	AppliesToAEZ []string              `json:"applies_to_aez_codes,omitempty"`
}

func (h *Handler) ask(w http.ResponseWriter, r *http.Request) {
	var body AskRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Problem(w, r, http.StatusBadRequest, "invalid-json", err.Error())
		return
	}
	body.Question = strings.TrimSpace(body.Question)
	if body.Question == "" {
		httpx.Problem(w, r, http.StatusBadRequest, "missing-question",
			"`question` is required")
		return
	}

	// Embed the query with the same embedder used at backfill — the
	// match operator only makes sense when both sides came out of the
	// same model.
	vec, err := h.embedder.Embed(r.Context(), body.Question)
	if err != nil {
		httpx.Problem(w, r, http.StatusInternalServerError, "embed-failed", err.Error())
		return
	}

	q := knowledge.SearchQuery{
		QueryVector: vec,
		CropSlug:    strings.TrimSpace(body.CropSlug),
		Limit:       body.Limit,
	}
	if c := strings.TrimSpace(body.Country); c != "" {
		q.Countries = []string{c}
	} else {
		q.Countries = []string{"LK"}
	}

	used := AskResponse{Question: body.Question, Embedder: h.embedder.Name()}
	used.UsedCropSlug = q.CropSlug

	// If the caller passed coordinates, resolve them to an AEZ envelope
	// and narrow the search to chunks that apply there. Falls back
	// gracefully when the geo lookup is unavailable so a missing geo
	// layer doesn't break the answer.
	if body.Lat != nil && body.Lng != nil {
		if env, err := h.geo.Lookup(r.Context(), geo.Point{Lat: *body.Lat, Lng: *body.Lng}); err == nil {
			if env.AEZ != nil {
				q.AEZCodes = []string{env.AEZ.Code}
				used.UsedAEZCodes = q.AEZCodes
			}
			if env.District != nil {
				used.UsedDistrict = env.District.NameEN
			}
		}
	}

	hits, err := h.chunks.Search(r.Context(), q)
	switch {
	case errors.Is(err, knowledge.ErrSearchRequiresDatabase):
		httpx.Problem(w, r, http.StatusServiceUnavailable, "ask-disabled",
			knowledge.ErrSearchRequiresDatabase.Error())
		return
	case err != nil:
		httpx.Problem(w, r, http.StatusInternalServerError, "search-failed", err.Error())
		return
	}

	// Stitch source metadata onto each hit so the client can render
	// publisher / licence / outbound URL without a second round-trip.
	views := make([]HitView, 0, len(hits))
	sourceCache := map[string]*knowledge.Source{}
	for _, h2 := range hits {
		var src *knowledge.Source
		if cached, ok := sourceCache[h2.Chunk.SourceSlug]; ok {
			src = cached
		} else if s, err := h.chunks.GetSource(r.Context(), h2.Chunk.SourceSlug); err == nil {
			src = &s
			sourceCache[h2.Chunk.SourceSlug] = src
		}
		views = append(views, HitView{
			Slug:         h2.Chunk.Slug,
			Title:        h2.Chunk.Title,
			Body:         h2.Chunk.Body,
			Quote:        h2.Chunk.Quote,
			Authority:    h2.Chunk.Authority,
			Confidence:   h2.Chunk.Confidence,
			Score:        h2.Score,
			Source:       src,
			EntityRefs:   h2.Chunk.EntityRefs,
			TopicTags:    h2.Chunk.TopicTags,
			AppliesToAEZ: h2.Chunk.AppliesToAEZCodes,
		})
	}

	used.Hits = views
	used.Count = len(views)
	used.Disclaimer = "Retrieved verbatim from the published knowledge corpus. Authority chips show source quality. " +
		"Lower-authority hits (regional / inferred-by-analogy) are advisory — verify with a DOA agronomist before applying chemical doses."

	httpx.JSON(w, http.StatusOK, used)
}
