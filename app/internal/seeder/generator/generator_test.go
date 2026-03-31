package generator

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestMovieGeneratorProducesRealisticDocuments(t *testing.T) {
	t.Parallel()

	var counter atomic.Uint64
	gen := NewMovieGenerator(42, &counter)
	seenIDs := make(map[string]struct{})

	for range 50 {
		doc := gen.Generate()
		if _, exists := seenIDs[doc.ID]; exists {
			t.Fatalf("duplicate movie ID generated: %s", doc.ID)
		}
		seenIDs[doc.ID] = struct{}{}

		if !strings.HasPrefix(doc.ID, "movie-") {
			t.Fatalf("expected movie ID prefix, got %q", doc.ID)
		}
		if doc.Title == "" || strings.Contains(doc.Title, "title-") {
			t.Fatalf("unexpected movie title: %q", doc.Title)
		}
		if len(doc.Synopsis) < 40 {
			t.Fatalf("synopsis too short: %q", doc.Synopsis)
		}
		if doc.Genre == "" || doc.Director == "" || doc.Language == "" {
			t.Fatalf("missing required text fields: %+v", doc)
		}
		if len(doc.Cast) != 3 || doc.Cast[0] == doc.Cast[1] || doc.Cast[1] == doc.Cast[2] || doc.Cast[0] == doc.Cast[2] {
			t.Fatalf("cast should contain three distinct entries: %+v", doc.Cast)
		}
		if doc.ReleaseYear < 1980 || doc.ReleaseYear > time.Now().UTC().Year() {
			t.Fatalf("release year out of range: %d", doc.ReleaseYear)
		}
		if doc.RuntimeMinutes < 85 || doc.RuntimeMinutes > 180 {
			t.Fatalf("runtime out of range: %d", doc.RuntimeMinutes)
		}
		if doc.Rating < 5.4 || doc.Rating > 9.6 {
			t.Fatalf("rating out of range: %.1f", doc.Rating)
		}
	}
}

func TestBookGeneratorProducesRealisticDocuments(t *testing.T) {
	t.Parallel()

	var counter atomic.Uint64
	gen := NewBookGenerator(99, &counter)
	seenIDs := make(map[string]struct{})

	for range 50 {
		doc := gen.Generate()
		if _, exists := seenIDs[doc.ID]; exists {
			t.Fatalf("duplicate book ID generated: %s", doc.ID)
		}
		seenIDs[doc.ID] = struct{}{}

		if !strings.HasPrefix(doc.ID, "book-") {
			t.Fatalf("expected book ID prefix, got %q", doc.ID)
		}
		if doc.Title == "" || strings.Contains(doc.Title, "title-") {
			t.Fatalf("unexpected book title: %q", doc.Title)
		}
		if len(doc.Summary) < 40 {
			t.Fatalf("summary too short: %q", doc.Summary)
		}
		if doc.Author == "" || doc.Genre == "" || doc.Language == "" {
			t.Fatalf("missing required text fields: %+v", doc)
		}
		if doc.PublicationYear < 1900 || doc.PublicationYear > time.Now().UTC().Year() {
			t.Fatalf("publication year out of range: %d", doc.PublicationYear)
		}
		if doc.PageCount < 140 || doc.PageCount > 900 {
			t.Fatalf("page count out of range: %d", doc.PageCount)
		}
		if doc.Rating < 3.4 || doc.Rating > 5.0 {
			t.Fatalf("rating out of range: %.1f", doc.Rating)
		}
		if !validISBN13(doc.ISBN) {
			t.Fatalf("generated invalid ISBN-13: %s", doc.ISBN)
		}
	}
}

func validISBN13(isbn string) bool {
	if len(isbn) != 13 {
		return false
	}
	sum := 0
	for idx, r := range isbn {
		if r < '0' || r > '9' {
			return false
		}
		value := int(r - '0')
		if idx%2 == 0 {
			sum += value
		} else {
			sum += value * 3
		}
	}
	return sum%10 == 0
}
