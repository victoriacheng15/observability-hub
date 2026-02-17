package postgres

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
)

// BrainStore handles persistence for thoughts in PostgreSQL.
type BrainStore struct {
	DB *sql.DB
}

// PARAStat represents stats for a specific category.
type PARAStat struct {
	Category    string
	TotalCount  int
	LatestEntry string
}

// NewBrainStore creates a new BrainStore.
func NewBrainStore(db *sql.DB) *BrainStore {
	return &BrainStore{DB: db}
}

// GetPARAStats retrieves the breakdown of thoughts by category from the stats view.
func (s *BrainStore) GetPARAStats(ctx context.Context) ([]PARAStat, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT category, total_entries, latest_entry FROM second_brain_stats")
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
	err := s.DB.QueryRowContext(ctx, "SELECT COALESCE(MAX(entry_date)::text, '1970-01-01') FROM second_brain").Scan(&latestDate)
	if err != nil {
		return "", err
	}
	return latestDate, nil
}

// InsertThought saves a single atomic thought into the database.
func (s *BrainStore) InsertThought(ctx context.Context, date, content, category string, tags []string, contextString, checksum string, tokenCount int) error {
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO second_brain (entry_date, content, category, origin_type, tags, context_string, checksum, token_count)
		VALUES ($1, $2, $3, 'journal', $4, $5, $6, $7)
		ON CONFLICT (checksum) DO NOTHING`,
		date, content, category, pq.Array(tags), contextString, checksum, tokenCount)
	return err
}
