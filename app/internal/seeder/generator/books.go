package generator

import (
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"
)

var bookGenres = []string{
	"Fantasy", "Science Fiction", "Mystery", "Historical Fiction", "Literary Fiction", "Romance", "Thriller", "Adventure", "Classic",
}

type BookGenerator struct {
	rng     *rand.Rand
	counter *atomic.Uint64
}

func NewBookGenerator(seed int64, counter *atomic.Uint64) *BookGenerator {
	return &BookGenerator{
		rng:     rand.New(rand.NewSource(seed)),
		counter: counter,
	}
}

func (g *BookGenerator) Generate() BookDoc {
	genre := pick(g.rng, bookGenres)
	return BookDoc{
		ID:              nextID("book", g.counter),
		Title:           g.title(),
		Summary:         g.summary(genre),
		Author:          pick(g.rng, authorNames),
		Genre:           genre,
		ISBN:            g.isbn(),
		PublicationYear: weightedYear(g.rng, 1900, time.Now().UTC().Year()),
		Language:        pick(g.rng, languages),
		PageCount:       g.pageCount(genre),
		Rating:          ratingBetween(g.rng, 3.4, 5.0),
	}
}

func (g *BookGenerator) title() string {
	switch g.rng.Intn(4) {
	case 0:
		return fmt.Sprintf("The %s %s", pick(g.rng, bookTitleQualifiers), pick(g.rng, bookTitleNouns))
	case 1:
		return fmt.Sprintf("%s at %s Hall", pick(g.rng, []string{"Winter", "Midnight", "Summer", "Autumn"}), pick(g.rng, bookTitleQualifiers))
	case 2:
		return fmt.Sprintf("Letters from the %s %s", pick(g.rng, bookTitleQualifiers), pick(g.rng, []string{"Province", "Harbor", "Frontier", "Archive"}))
	default:
		return fmt.Sprintf("The %s's %s", pick(g.rng, []string{"Cartographer", "Bookseller", "Astronomer", "Caretaker", "Violinist"}), pick(g.rng, bookTitleNouns))
	}
}

func (g *BookGenerator) summary(genre string) string {
	conflicts := bookConflicts[genre]
	return fmt.Sprintf("%s %s when %s, and must act %s.",
		sentenceCase(pick(g.rng, bookProtagonists)),
		pick(g.rng, bookSettings),
		pick(g.rng, conflicts),
		pick(g.rng, bookStakes),
	)
}

func (g *BookGenerator) isbn() string {
	prefix := fmt.Sprintf("978%09d", g.rng.Intn(1_000_000_000))
	return checksumISBN13(prefix)
}

func (g *BookGenerator) pageCount(genre string) int {
	switch genre {
	case "Classic", "Literary Fiction", "Romance":
		return randomInt(g.rng, 180, 420)
	case "Fantasy", "Historical Fiction":
		return randomInt(g.rng, 320, 900)
	case "Science Fiction", "Adventure", "Thriller":
		return randomInt(g.rng, 220, 560)
	default:
		return randomInt(g.rng, 140, 480)
	}
}
