package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"log/slog"
)

// User represents a user domain model
type User struct {
	ID       int     `json:"id"`
	Username string  `json:"username_s"`
	Email    string  `json:"email_s"`
	Age      int     `json:"age_i"`
	Active   bool    `json:"active_b"`
	Balance  float64 `json:"balance_f"`
}

// Movie represents a movie domain model
type Movie struct {
	ID          int     `json:"id"`
	Title       string  `json:"title_s"`
	Director    string  `json:"director_s"`
	ReleaseYear int     `json:"release_year_i"`
	Genre       string  `json:"genre_s"`
	Rating      float32 `json:"rating_f"`
	IsAvailable bool    `json:"is_available_b"`
	Duration    float64 `json:"duration_f"`
}

// randomUser generates a random User instance
func randomUser() User {
	usernames := []string{"alice", "bob", "charlie", "dave", "eve", "frank", "grace", "heidi"}
	domains := []string{"example.com", "mail.com", "test.org", "demo.net"}
	username := usernames[rand.Intn(len(usernames))]
	domain := domains[rand.Intn(len(domains))]
	age := rand.Intn(60) + 18        // 18-77
	balance := rand.Float64() * 1000 // $0.00 - $999.99
	active := rand.Intn(2) == 1
	id := rand.Intn(1000000)
	return User{
		ID:       id,
		Username: username,
		Email:    fmt.Sprintf("%s%d@%s", username, id, domain),
		Age:      age,
		Active:   active,
		Balance:  balance,
	}
}

// randomMovie generates a random Movie instance
func randomMovie() Movie {
	titles := []string{"The Example", "Another Film", "Go Adventure", "Mystery Night", "Comedy Hour", "Sci-Fi Saga", "Drama Days", "Action Blast"}
	directors := []string{"Jane Doe", "John Smith", "Alex Lee", "Sam Kim", "Morgan Yu", "Chris Ray"}
	genres := []string{"Drama", "Comedy", "Action", "Sci-Fi", "Horror", "Romance"}
	title := titles[rand.Intn(len(titles))]
	director := directors[rand.Intn(len(directors))]
	genre := genres[rand.Intn(len(genres))]
	year := rand.Intn(41) + 1985       // 1985-2025
	rating := rand.Float32()*4.0 + 5.0 // 5.0-9.0
	available := rand.Intn(2) == 1
	duration := rand.Float64()*60 + 60 // 60-120 min
	id := rand.Intn(1000000)
	return Movie{
		ID:          id,
		Title:       title,
		Director:    director,
		ReleaseYear: year,
		Genre:       genre,
		Rating:      rating,
		IsAvailable: available,
		Duration:    duration,
	}
}

// postToSolr sends a slice of documents to the Solr JSON update API for the given collection.
func postToSolr(ctx context.Context, solrURL, collection string, docs interface{}) error {
	// Solr JSON update API endpoint: /solr/<collection>/update?commit=true
	url := fmt.Sprintf("%s/solr/%s/update?commit=true", solrURL, collection)

	b, err := json.Marshal(docs)
	if err != nil {
		return fmt.Errorf("marshal docs: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("solr returned status %s", resp.Status)
	}

	return nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: false}))
	ctx := context.Background()
	rand.Seed(time.Now().UnixNano())

	solrURL := os.Getenv("SOLR_MASTER_URL")
	if solrURL == "" {
		logger.Error("SOLR_MASTER_URL is not set")
		os.Exit(1)
	}

	logger.InfoContext(ctx, "starting seeding", "solr_master", solrURL)

	const (
		totalUsers  = 1_000_000
		totalMovies = 1_000_000
		batchSize   = 1000
	)

	var wg sync.WaitGroup
	wg.Add(2)

	// Seed users in a goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < totalUsers; i += batchSize {
			batch := make([]User, 0, batchSize)
			for j := 0; j < batchSize && i+j < totalUsers; j++ {
				batch = append(batch, randomUser())
			}
			if err := postToSolr(ctx, solrURL, "users", batch); err != nil {
				logger.ErrorContext(ctx, "failed to post user batch", "error", err, "batch_start", i)
			} else if (i/batchSize)%10 == 0 {
				logger.InfoContext(ctx, "seeded user batch", "batch_start", i, "batch_size", len(batch))
			}
		}
		logger.InfoContext(ctx, "finished seeding users")
	}()

	// Seed movies in a goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < totalMovies; i += batchSize {
			batch := make([]Movie, 0, batchSize)
			for j := 0; j < batchSize && i+j < totalMovies; j++ {
				batch = append(batch, randomMovie())
			}
			if err := postToSolr(ctx, solrURL, "movies", batch); err != nil {
				logger.ErrorContext(ctx, "failed to post movie batch", "error", err, "batch_start", i)
			} else if (i/batchSize)%10 == 0 {
				logger.InfoContext(ctx, "seeded movie batch", "batch_start", i, "batch_size", len(batch))
			}
		}
		logger.InfoContext(ctx, "finished seeding movies")
	}()

	wg.Wait()
	logger.InfoContext(ctx, "seeding complete, exiting")
}
