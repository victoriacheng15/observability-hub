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

	"db/postgres"
	"secrets"

	"github.com/joho/godotenv"
	"github.com/lib/pq"
)

type AtomicThought struct {
	Date          string   `json:"date"`
	Content       string   `json:"content"`
	Category      string   `json:"category"`
	Tags          []string `json:"tags"`
	ContextString string   `json:"context_string"`
	Checksum      string   `json:"checksum"`
	TokenCount    int      `json:"token_count"`
}

var tagsMap = map[string][]string{
	"ai":            {"ai", "llm", "rag", "pgvector", "embedding", "openai", "gemini", "agent", "deepseek"},
	"observability": {"grafana", "loki", "alloy", "opentelemetry", "otel", "metrics", "tracing", "logs", "prometheus"},
	"kubernetes":    {"k3s", "pod", "pvc", "statefulset", "daemonset", "kubectl", "helm", "k8s"},
	"database":      {"postgres", "postgresql", "jsonb", "timescaledb", "postgis", "sql", "mongodb", "database", "db"},
	"devops":        {"github actions", "gitops", "reconciliation", "ci-cd", "docker", "terraform", "nix", "shell.nix"},
	"career-growth": {"impostor syndrome", "growth", "senior", "leadership", "reflection", "mentorship", "career", "brag", "win", "impact"},
	"platform":      {"openbao", "tailscale", "security", "infrastructure", "zero-trust", "secrets", "bao"},
	"language":      {"go", "golang", "python", "rust", "typescript", "javascript", "bash", "shell"},
	"linux":         {"linux", "systemd", "kernel", "gpu", "psu", "hardware", "cpu", "memory", "nixos"},
	"productivity":  {"para", "second brain", "zettelkasten", "notion", "obsidian", "journal", "workflow"},
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

	conn, err := postgres.ConnectPostgres("postgres", store)
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
	rows, err := conn.Query("SELECT category, total_entries, latest_entry FROM second_brain_stats")
	if err != nil {
		fmt.Printf("⚠️ Could not fetch final stats: %v\n", err)
	} else {
		defer rows.Close()
		fmt.Printf("\n✅ Sync Complete! New atoms processed: %d\n", totalAtoms)
		fmt.Println("🧠 Second Brain Status (PARA):")
		for rows.Next() {
			var cat, latest string
			var count int
			_ = rows.Scan(&cat, &count, &latest)
			fmt.Printf("   - [%-8s] %3d entries (Latest: %s)\n", cat, count, latest)
		}
	}
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
	currentCategory := "resource"

	flush := func() {
		if len(currentBlocks) > 0 {
			text := strings.TrimSpace(strings.Join(currentBlocks, "\n"))
			if text == "" {
				return
			}
			tags := getTags(text)
			context := fmt.Sprintf("Date: %s | Category: %s | Tags: %s | Content: %s", date, currentCategory, strings.Join(tags, ", "), text)
			atoms = append(atoms, AtomicThought{
				Date:          date,
				Content:       text,
				Category:      currentCategory,
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

		// 1. Skip Comments
		if !inComment && strings.Contains(line, "<!--") {
			inComment = true
		}
		if inComment {
			if strings.Contains(line, "-->") {
				inComment = false
			}
			continue
		}

		// 2. Detect Singular PARA Headers
		if strings.HasPrefix(trimmed, "## ") {
			if capturing {
				flush()
			}

			switch trimmed {
			case "## Project":
				currentCategory = "project"
				capturing = true
				continue
			case "## Area":
				currentCategory = "area"
				capturing = true
				continue
			case "## Resource":
				currentCategory = "resource"
				capturing = true
				continue
			case "## Archive":
				currentCategory = "archive"
				capturing = true
				continue
			case "## Thought":
				currentCategory = "resource"
				capturing = true
				continue
			default:
				// Today's Task and any other header should stop capturing
				capturing = false
			}
		}

		// 3. Stop capturing on separator
		if capturing && strings.HasPrefix(trimmed, "---") {
			flush()
			capturing = false
			continue
		}

		// 4. Capture Content (Skip actual task lines just in case they are nested)
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
			INSERT INTO second_brain (entry_date, content, category, origin_type, tags, context_string, checksum, token_count)
			VALUES ($1, $2, $3, 'journal', $4, $5, $6, $7)
			ON CONFLICT (checksum) DO NOTHING`,
			a.Date, a.Content, a.Category, pq.Array(a.Tags), a.ContextString, a.Checksum, a.TokenCount)
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
