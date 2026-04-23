package embed

import (
	"context"
	"math"
	"testing"
)

func TestHashEmbedder_DimensionAndUnitNorm(t *testing.T) {
	e := NewHashEmbedder()
	v, err := e.Embed(context.Background(), "rice blast disease management")
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != Dimension {
		t.Fatalf("want %d dims, got %d", Dimension, len(v))
	}
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	if math.Abs(sum-1.0) > 1e-5 {
		t.Fatalf("expected unit norm, got %.4f", sum)
	}
}

func TestHashEmbedder_Deterministic(t *testing.T) {
	e := NewHashEmbedder()
	v1, _ := e.Embed(context.Background(), "tomato early blight")
	v2, _ := e.Embed(context.Background(), "tomato early blight")
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Fatalf("non-deterministic at index %d: %f vs %f", i, v1[i], v2[i])
		}
	}
}

func TestHashEmbedder_Similarity(t *testing.T) {
	// Sanity check: cosine similarity of related strings should be
	// higher than between unrelated strings. The hashing trick can
	// produce near-collisions, but the overall signal must be there.
	e := NewHashEmbedder()
	a, _ := e.Embed(context.Background(), "tomato early blight Mancozeb fungicide")
	b, _ := e.Embed(context.Background(), "tomato late blight Mancozeb spray")
	c, _ := e.Embed(context.Background(), "onion thrips reflective mulch ipm")

	simAB := dot(a, b)
	simAC := dot(a, c)
	if simAB <= simAC {
		t.Fatalf("expected related (sim=%.3f) > unrelated (sim=%.3f)", simAB, simAC)
	}
}

func TestHashEmbedder_EmptyText(t *testing.T) {
	v, err := NewHashEmbedder().Embed(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != Dimension {
		t.Fatalf("want %d dims for empty input, got %d", Dimension, len(v))
	}
	for _, x := range v {
		if x != 0 {
			t.Fatal("empty input should yield zero vector")
		}
	}
}

func TestTokenize_DropsStopWordsAndShortTokens(t *testing.T) {
	got := tokenize("How do I treat blast in my rice")
	want := map[string]bool{"treat": true, "blast": true, "rice": true}
	for _, tok := range got {
		if _, ok := want[tok]; !ok {
			t.Errorf("unexpected token %q in %v", tok, got)
		}
		delete(want, tok)
	}
	for tok := range want {
		t.Errorf("missing expected token %q in %v", tok, got)
	}
}

func dot(a, b []float32) float32 {
	var s float32
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}
