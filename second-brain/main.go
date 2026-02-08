package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"

	"db"
	"secrets"

	"github.com/joho/godotenv"
	"github.com/lib/pq"
)

type AtomicThought struct {
	Date          string   `json:"date"`
	Content       string   `json:"content"`
	Tags          []string `json:"tags"`
	ContextString string   `json:"context_string"`
	Checksum      string   `json:"checksum"`
	TokenCount    int      `json:"token_count"`
}

var tagsMap = map[string][]string{
	"ai":            {"ai", "llm", "rag", "pgvector", "embedding", "openai", "gemini"},
	"observability": {"grafana", "loki", "alloy", "opentelemetry", "otel", "metrics", "tracing", "logs"},
	"kubernetes":    {"k3s", "pod", "pvc", "statefulset", "daemonset", "kubectl", "helm"},
	"database":      {"postgres", "postgresql", "jsonb", "timescaledb", "postgis", "sql"},
	"devops":        {"github actions", "gitops", "reconciliation", "ci-cd", "docker"},
	"career-growth": {"impostor syndrome", "growth", "senior", "leadership", "reflection"},
	"platform":      {"openbao", "tailscale", "security", "infrastructure", "zero-trust"},
}

func main() {
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")

	repo := os.Getenv("JOURNAL_REPO")
	if repo == "" {
		log.Fatal("❌ JOURNAL_REPO environment variable not set")
	}

	// 1. Initialize Secrets & DB
	store, err := secrets.NewBaoProvider()
	if err != nil {
		log.Fatalf("Failed to initialize secret store: %v", err)
	}
	defer store.Close()

	conn, err := db.ConnectPostgres("postgres", store)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close()

	// 2. Get Latest Entry Date from DB
	var latestDate string
	err = conn.QueryRow("SELECT COALESCE(MAX(entry_date)::text, '1970-01-01') FROM second_brain").Scan(&latestDate)
	if err != nil {
		log.Fatalf("Failed to query latest date: %v", err)
	}
	fmt.Printf("📊 Database check complete. Latest entry: %s\n", latestDate)

	// 3. Fetch recent issues
	allIssues, err := fetchIssues(repo)
	if err != nil {
		log.Fatalf("Failed to fetch issues: %v", err)
	}

	// 4. Filter for new issues (Title > latestDate)
	var newIssues []struct {
		Number int
		Title  string
	}
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
		body, err := fetchBody(repo, iss.Number)
		if err != nil {
			fmt.Printf("⚠️ Error fetching #%d: %v\n", iss.Number, err)
			continue
		}

		atoms := atomize(iss.Title, body)
		if err := saveToDB(conn, atoms); err != nil {
			fmt.Printf("❌ Database error for #%d: %v\n", iss.Number, err)
			continue
		}
		totalAtoms += len(atoms)
	}

	// Final Status
	var count int
	var latest string
	_ = conn.QueryRow("SELECT total_entries, latest_entry FROM second_brain_stats").Scan(&count, &latest)
	fmt.Printf("\n✅ Sync Complete! New atoms processed: %d\n", totalAtoms)
	fmt.Printf("🧠 Second Brain Status: %d total entries. Latest: %s\n", count, latest)
}

func fetchIssues(repo string) ([]struct {
	Number int
	Title  string
}, error) {
	fmt.Printf("🔍 Fetching recent journals from %s...\n", repo)
	cmd := exec.Command("gh", "issue", "list", "--repo", repo, "--label", "journal", "--state", "all", "--limit", "50", "--json", "number,title")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var issues []struct {
		Number int
		Title  string
	}
	if err := json.Unmarshal(out.Bytes(), &issues); err != nil {
		return nil, err
	}
	return issues, nil
}

func fetchBody(repo string, number int) (string, error) {
	cmd := exec.Command("gh", "issue", "view", fmt.Sprintf("%d", number), "--repo", repo, "--json", "body", "--jq", ".body")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}

func atomize(date, body string) []AtomicThought {
	scanner := bufio.NewScanner(strings.NewReader(body))
	var atoms []AtomicThought
	var currentBlocks []string
	capturing := false
	inComment := false

	flush := func() {
		if len(currentBlocks) > 0 {
			text := strings.TrimSpace(strings.Join(currentBlocks, "\n"))
			if text == "" {
				return
			}
			tags := getTags(text)
			context := fmt.Sprintf("Date: %s | Tags: %s | Content: %s", date, strings.Join(tags, ", "), text)
			atoms = append(atoms, AtomicThought{
				Date:          date,
				Content:       text,
				Tags:          tags,
				ContextString: context,
				Checksum:      getChecksum(text),
				TokenCount:    estimateTokens(context),
			})
			currentBlocks = nil
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if !inComment && strings.Contains(line, "<!--") {
			inComment = true
		}
		if inComment {
			if strings.Contains(line, "-->") {
				inComment = false
			}
			continue
		}

		if strings.Contains(line, "## Thoughts") {
			capturing = true
			continue
		}

		if capturing && (strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "---")) {
			capturing = false
			break
		}

		if capturing {
			if strings.HasPrefix(trimmed, "- [ ]") || strings.HasPrefix(trimmed, "- [x]") {
				continue
			}
			if trimmed != "" {
				currentBlocks = append(currentBlocks, line)
			}
		}
	}
	flush()
	return atoms
}

func saveToDB(conn *sql.DB, atoms []AtomicThought) error {
	for _, a := range atoms {
		_, err := conn.Exec(`
			INSERT INTO second_brain (entry_date, content, tags, context_string, checksum, token_count)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (checksum) DO NOTHING`,
			a.Date, a.Content, pq.Array(a.Tags), a.ContextString, a.Checksum, a.TokenCount)
		if err != nil {
			return err
		}
	}
	return nil
}

func getTags(text string) []string {
	textLower := strings.ToLower(text)
	foundMap := make(map[string]bool)
	for tag, keywords := range tagsMap {
		for _, kw := range keywords {
			if strings.Contains(textLower, kw) {
				foundMap[tag] = true
				break
			}
		}
	}
	var found []string
	for tag := range foundMap {
		found = append(found, tag)
	}
	sort.Strings(found)
	return found
}

func getChecksum(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

func estimateTokens(text string) int {
	words := strings.Fields(text)
	return int(float64(len(words))*1.33) + 1
}
