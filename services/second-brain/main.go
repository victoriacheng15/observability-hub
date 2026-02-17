package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"

	"brain"
	"db/postgres"
	"secrets"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	_ = godotenv.Load("../../.env")

	repo := os.Getenv("JOURNAL_REPO")
	if repo == "" {
		log.Fatal("❌ JOURNAL_REPO environment variable not set")
	}

	ctx := context.Background()

	// 1. Initialize Secrets & DB
	store, err := secrets.NewBaoProvider()
	if err != nil {
		log.Fatalf("Failed to initialize secret store: %v", err)
	}
	defer store.Close()

	conn, err := postgres.ConnectPostgres("postgres", store)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close()

	brainStore := postgres.NewBrainStore(conn)

	// 2. Get Latest Entry Date from DB
	latestDate, err := brainStore.GetLatestEntryDate(ctx)
	if err != nil {
		log.Fatalf("Failed to query latest date: %v", err)
	}
	fmt.Printf("📊 Database check complete. Latest entry: %s\n", latestDate)

	// 3. Fetch recent issues
	allIssues, err := brain.FetchRecentJournals(repo)
	if err != nil {
		log.Fatalf("Failed to fetch issues: %v", err)
	}

	// 4. Filter for new issues (Title > latestDate)
	var newIssues []brain.GitHubIssue
	for _, iss := range allIssues {
		// Journal titles follow YYYY-MM-DD format
		if iss.Title > latestDate {
			newIssues = append(newIssues, iss)
		}
	}

	if len(newIssues) == 0 {
		fmt.Println("✨ Second Brain is already up to date. No new thoughts found.")
		return
	}

	sort.Slice(newIssues, func(i, j int) bool { return newIssues[i].Title < newIssues[j].Title })

	// 5. Process and Ingest
	totalAtoms := 0
	for _, iss := range newIssues {
		fmt.Printf("📦 Ingesting delta: #%d (%s)...\n", iss.Number, iss.Title)
		body, err := brain.FetchIssueBody(repo, iss.Number)
		if err != nil {
			fmt.Printf("⚠️ Error fetching #%d: %v\n", iss.Number, err)
			continue
		}

		atoms := brain.Atomize(iss.Title, body)
		for _, a := range atoms {
			if err := brainStore.InsertThought(ctx, a.Date, a.Content, a.Category, a.Tags, a.ContextString, a.Checksum, a.TokenCount); err != nil {
				fmt.Printf("❌ Database error for atom in #%d: %v\n", iss.Number, err)
				continue
			}
		}
		totalAtoms += len(atoms)
	}

	// Final Status
	stats, err := brainStore.GetPARAStats(ctx)
	if err != nil {
		fmt.Printf("⚠️ Could not fetch final stats: %v\n", err)
	} else {
		fmt.Printf("\n✅ Sync Complete! New atoms processed: %d\n", totalAtoms)
		fmt.Println("🧠 Second Brain Status (PARA):")
		for _, st := range stats {
			fmt.Printf("   - [%-8s] %3d entries (Latest: %s)\n", st.Category, st.TotalCount, st.LatestEntry)
		}
	}
}
