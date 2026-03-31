package generator

import (
	"fmt"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"
)

var movieGenres = []string{
	"Science Fiction", "Fantasy", "Thriller", "Drama", "Mystery", "Crime", "Adventure", "Animation", "Romance",
}

type MovieGenerator struct {
	rng     *rand.Rand
	counter *atomic.Uint64
}

func NewMovieGenerator(seed int64, counter *atomic.Uint64) *MovieGenerator {
	return &MovieGenerator{
		rng:     rand.New(rand.NewSource(seed)),
		counter: counter,
	}
}

func (g *MovieGenerator) Generate() MovieDoc {
	genre := pick(g.rng, movieGenres)
	language := pick(g.rng, languages)
	return MovieDoc{
		ID:             nextID("movie", g.counter),
		Title:          g.title(),
		Synopsis:       g.synopsis(genre),
		Genre:          genre,
		ReleaseYear:    weightedYear(g.rng, 1980, time.Now().UTC().Year()),
		Director:       pick(g.rng, directorNames),
		Cast:           pickDistinct(g.rng, actorNames, 3),
		Language:       language,
		RuntimeMinutes: g.runtime(genre),
		Rating:         ratingBetween(g.rng, 5.4, 9.6),
	}
}

func (g *MovieGenerator) title() string {
	switch g.rng.Intn(3) {
	case 0:
		return fmt.Sprintf("%s %s", pick(g.rng, movieTitleAdjectives), pick(g.rng, movieTitleNouns))
	case 1:
		return fmt.Sprintf("The %s %s", pick(g.rng, movieTitleQualifiers), pick(g.rng, movieTitleNouns))
	default:
		return fmt.Sprintf("%s of the %s %s", pick(g.rng, movieTitleNouns), pick(g.rng, movieTitleQualifiers), pick(g.rng, movieTitleNouns))
	}
}

func (g *MovieGenerator) synopsis(genre string) string {
	conflicts := movieConflicts[genre]
	return fmt.Sprintf("%s %s %s and %s %s.",
		sentenceCase(pick(g.rng, movieProtagonists)),
		pick(g.rng, movieSettings),
		pick(g.rng, conflicts),
		pick(g.rng, []string{"must outmaneuver old allies", "is forced to face an impossible bargain", "has to trust a stranger with everything", "must choose what to sacrifice"}),
		pick(g.rng, movieStakes),
	)
}

func (g *MovieGenerator) runtime(genre string) int {
	switch genre {
	case "Animation":
		return randomInt(g.rng, 85, 112)
	case "Drama", "Romance":
		return randomInt(g.rng, 95, 135)
	case "Adventure", "Fantasy", "Science Fiction":
		return randomInt(g.rng, 110, 180)
	default:
		return randomInt(g.rng, 98, 148)
	}
}

func sentenceCase(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}
