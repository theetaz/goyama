// Package embed produces 1024-dimension vector embeddings for the
// knowledge chunks + queries that drive /v1/ask retrieval.
//
// Two implementations:
//
//  1. HashEmbedder — deterministic feature-hashing trick over
//     unicode-tokenised text. No external dependencies, no API keys,
//     no model bytes. Cosine similarity on hashed vectors approximates
//     a Jaccard-style token overlap, which is good enough for dev,
//     CI, and offline smoke-tests.
//
//  2. VoyageEmbedder — wraps Voyage AI's REST embedding API
//     (voyage-3-lite, 1024-dim by default). Picked because Voyage is
//     the standard embedding pairing with Anthropic Claude and we use
//     Claude elsewhere in the pipeline.
//
// The active embedder is chosen via env: VOYAGE_API_KEY set →
// VoyageEmbedder, otherwise HashEmbedder. The same instance must be
// used for both chunk backfill and query-time embedding so the cosine
// math stays meaningful.
package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"
)

// Dimension is the fixed embedding width — must match the
// `vector(1024)` column in migration 0010.
const Dimension = 1024

// Embedder turns a stretch of natural-language text into a unit-norm
// 1024-dim vector. Implementations are concurrency-safe.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	// Name identifies which embedder produced a vector. Stored alongside
	// each chunk's embedding so a future migration to a different
	// embedder can re-backfill only the rows that need it.
	Name() string
}

// FromEnv selects an embedder based on environment configuration:
// VOYAGE_API_KEY → Voyage (production); otherwise HashEmbedder (dev).
// The reason for the env-driven default is that the same binary runs
// in CI (no key) and in production (key set) without code changes.
func FromEnv() Embedder {
	if key := strings.TrimSpace(os.Getenv("VOYAGE_API_KEY")); key != "" {
		return NewVoyageEmbedder(key)
	}
	return NewHashEmbedder()
}

// ─── hash embedder ────────────────────────────────────────────────────────

// HashEmbedder turns text into a 1024-dim TF vector via the hashing
// trick: each token's FNV-1a hash mod Dimension picks a coordinate to
// increment. The resulting vector is L2-normalised so cosine
// similarity reduces to a dot product. Stop words are dropped so they
// don't dominate the signal.
type HashEmbedder struct{}

// NewHashEmbedder returns a stateless dev embedder.
func NewHashEmbedder() *HashEmbedder { return &HashEmbedder{} }

// Name returns the stable identifier for this embedder.
func (*HashEmbedder) Name() string { return "hash-fnv-1024" }

// Embed produces a unit-norm 1024-dim vector for text.
func (*HashEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	tokens := tokenize(text)
	if len(tokens) == 0 {
		// Empty text deserves an empty vector — caller can decide
		// whether to skip or carry on. We return zeros so the column
		// type stays consistent.
		return make([]float32, Dimension), nil
	}
	vec := make([]float32, Dimension)
	for _, tok := range tokens {
		h := fnv.New32a()
		_, _ = h.Write([]byte(tok))
		idx := h.Sum32() % Dimension
		// Sign bit flips on odd hash bytes so two unrelated tokens
		// don't accidentally reinforce each other when hashed to the
		// same bin (the standard "signed hashing" trick).
		if h.Sum32()&1 == 1 {
			vec[idx] += 1
		} else {
			vec[idx] -= 1
		}
	}
	return l2Normalize(vec), nil
}

// stopWords is a deliberately tiny list — just the words that flatten
// agronomic queries ("how do I treat blast in my rice"). Bigger
// stop-word lists pull semantic content out for short queries, which
// hurts more than it helps here.
var stopWords = map[string]struct{}{
	"a": {}, "an": {}, "the": {}, "is": {}, "are": {}, "was": {}, "were": {},
	"of": {}, "in": {}, "on": {}, "at": {}, "to": {}, "for": {}, "and": {},
	"or": {}, "but": {}, "with": {}, "by": {}, "from": {}, "as": {}, "be": {},
	"this": {}, "that": {}, "these": {}, "those": {}, "it": {}, "its": {},
	"i": {}, "you": {}, "we": {}, "my": {}, "our": {},
	"how": {}, "do": {}, "does": {}, "did": {}, "can": {}, "should": {},
}

func tokenize(text string) []string {
	out := make([]string, 0, 32)
	var sb strings.Builder
	flush := func() {
		if sb.Len() == 0 {
			return
		}
		t := strings.ToLower(sb.String())
		sb.Reset()
		if _, drop := stopWords[t]; drop {
			return
		}
		if len(t) < 2 {
			return
		}
		out = append(out, t)
	}
	for _, r := range text {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			sb.WriteRune(r)
		default:
			flush()
		}
	}
	flush()
	return out
}

func l2Normalize(v []float32) []float32 {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	if sum == 0 {
		return v
	}
	inv := float32(1.0 / math.Sqrt(sum))
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = x * inv
	}
	return out
}

// ─── voyage embedder ──────────────────────────────────────────────────────

// VoyageEmbedder calls Voyage AI's `/v1/embeddings` endpoint. We pin
// the model to voyage-3-lite (1024 dims, $0.02/1M tokens at the time
// of writing). Switching models is a one-line change here; the bulk
// re-backfill is handled by passing a different `--embedder` value to
// cmd/embedchunks.
type VoyageEmbedder struct {
	apiKey string
	model  string
	client *http.Client
}

// NewVoyageEmbedder returns a Voyage AI embedder. Caller is
// responsible for protecting the API key (don't pass user-supplied
// data through it without a quota).
func NewVoyageEmbedder(apiKey string) *VoyageEmbedder {
	return &VoyageEmbedder{
		apiKey: apiKey,
		model:  "voyage-3-lite",
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns "voyage-3-lite" so backfilled vectors can be associated
// with the model that produced them.
func (e *VoyageEmbedder) Name() string { return e.model }

// Embed sends `text` to Voyage AI and returns the embedding vector.
func (e *VoyageEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	body, err := json.Marshal(map[string]any{
		"input":      []string{text},
		"model":      e.model,
		"input_type": "document",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.voyageai.com/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("voyage request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("voyage returned %d: %s", resp.StatusCode, string(respBody))
	}
	var out struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(out.Data) == 0 {
		return nil, fmt.Errorf("voyage returned no embedding data")
	}
	if len(out.Data[0].Embedding) != Dimension {
		return nil, fmt.Errorf("voyage returned %d dims, expected %d",
			len(out.Data[0].Embedding), Dimension)
	}
	return out.Data[0].Embedding, nil
}
