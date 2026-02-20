package main

import (
	"context"
	"fmt"

	"db/postgres"
)

const (
	tableSecondBrain = "second_brain"
	viewBrainStats   = "second_brain_stats"
)

// BrainStore handles persistence for thoughts in PostgreSQL.
type BrainStore struct {
	Wrapper *postgres.PostgresWrapper
}

// PARAStat represents stats for a specific category.
type PARAStat struct {
	Category    string
	TotalCount  int
	LatestEntry string
}

// NewBrainStore creates a new BrainStore.
func NewBrainStore(w *postgres.PostgresWrapper) *BrainStore {
	return &BrainStore{Wrapper: w}
}

// GetPARAStats retrieves the breakdown of thoughts by category from the stats view.
func (s *BrainStore) GetPARAStats(ctx context.Context) ([]PARAStat, error) {
	query := fmt.Sprintf("SELECT category, total_entries, latest_entry FROM %s", viewBrainStats)
	rows, err := s.Wrapper.Query(ctx, "db.postgres.get_para_stats", query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []PARAStat
	for rows.Next() {
		var st PARAStat
		if err := rows.Scan(&st.Category, &st.TotalCount, &st.LatestEntry); err != nil {
			return nil, err
		}
		stats = append(stats, st)
	}
	return stats, nil
}

// GetLatestEntryDate retrieves the date of the most recent entry in the second_brain table.
func (s *BrainStore) GetLatestEntryDate(ctx context.Context) (string, error) {
	var latestDate string
	query := fmt.Sprintf("SELECT COALESCE(MAX(entry_date)::text, '1970-01-01') FROM %s", tableSecondBrain)
	err := s.Wrapper.QueryRow(ctx, "db.postgres.get_latest_entry_date", query).Scan(&latestDate)
	if err != nil {
		return "", err
	}
	return latestDate, nil
}

// InsertThought saves a single atomic thought into the database.
func (s *BrainStore) InsertThought(ctx context.Context, date, content, category string, tags []string, contextString, checksum string, tokenCount int) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (entry_date, content, category, origin_type, tags, context_string, checksum, token_count)
		VALUES ($1, $2, $3, 'journal', $4, $5, $6, $7)
		ON CONFLICT (checksum) DO NOTHING`, tableSecondBrain)

	_, err := s.Wrapper.Exec(ctx, "db.postgres.insert_thought", query,
		date, content, category, s.Wrapper.Array(tags), contextString, checksum, tokenCount)
	return err
}
