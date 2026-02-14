package digest

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/pkg/errors"

	"github.com/usememos/memos/plugin/supabase"
	"github.com/usememos/memos/store"
)

// DigestContent represents the content of a weekly digest.
type DigestContent struct {
	User           *store.User
	WeekStart      time.Time
	WeekEnd        time.Time
	ThisWeekMemos  []*store.Memo
	Connections    []Connection
	Themes         []ThemeCluster
	TotalMemoCount int
	// LLM-generated analysis
	Analysis       *AnalysisResult
}

// Generator creates weekly digest content for users.
type Generator struct {
	store          *store.Store
	supabaseClient *supabase.Client
	analyzer       *Analyzer
}

// NewGenerator creates a new digest generator.
func NewGenerator(store *store.Store) (*Generator, error) {
	client, err := supabase.NewClient()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create supabase client")
	}

	// Try to create analyzer (optional - will work without it)
	analyzer, err := NewAnalyzer()
	if err != nil {
		slog.Warn("Failed to create LLM analyzer, digests will be basic", "error", err)
	}

	return &Generator{
		store:          store,
		supabaseClient: client,
		analyzer:       analyzer,
	}, nil
}

// NewGeneratorWithClient creates a new digest generator with a provided Supabase client.
func NewGeneratorWithClient(store *store.Store, client *supabase.Client) *Generator {
	analyzer, _ := NewAnalyzer() // Optional
	return &Generator{
		store:          store,
		supabaseClient: client,
		analyzer:       analyzer,
	}
}

// GenerateDigest creates a digest for the specified user for the past week.
func (g *Generator) GenerateDigest(ctx context.Context, user *store.User) (*DigestContent, error) {
	if user == nil {
		return nil, errors.New("user is required")
	}

	// Calculate week boundaries (Sunday to Sunday)
	now := time.Now().UTC()
	weekEnd := now
	weekStart := now.AddDate(0, 0, -7)

	// Fetch this week's memos for the user
	thisWeekMemos, err := g.fetchThisWeekMemos(ctx, user.ID, weekStart)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch this week's memos")
	}

	// If no memos this week, return a minimal digest
	if len(thisWeekMemos) == 0 {
		return &DigestContent{
			User:           user,
			WeekStart:      weekStart,
			WeekEnd:        weekEnd,
			ThisWeekMemos:  thisWeekMemos,
			Connections:    nil,
			Themes:         nil,
			TotalMemoCount: 0,
		}, nil
	}

	// Get memo names for this week
	memoNames := make([]string, len(thisWeekMemos))
	for i, memo := range thisWeekMemos {
		memoNames[i] = memo.UID
	}

	// Fetch all embeddings from Supabase
	allEmbeddings, err := g.supabaseClient.GetAllEmbeddings()
	if err != nil {
		slog.Warn("Failed to fetch embeddings from Supabase, skipping semantic connections",
			"error", err, "user_id", user.ID)
		// Continue without connections
		return &DigestContent{
			User:           user,
			WeekStart:      weekStart,
			WeekEnd:        weekEnd,
			ThisWeekMemos:  thisWeekMemos,
			Connections:    nil,
			Themes:         IdentifyThemes(thisWeekMemos, nil),
			TotalMemoCount: len(thisWeekMemos),
		}, nil
	}

	// Separate this week's embeddings from previous
	thisWeekEmbeddings, previousEmbeddings := g.separateEmbeddings(allEmbeddings, memoNames, weekStart)

	// Find semantic connections
	connections := FindSemanticConnections(thisWeekEmbeddings, previousEmbeddings)

	// Generate insights for each connection
	memoByUID := make(map[string]*store.Memo)
	for _, memo := range thisWeekMemos {
		memoByUID[memo.UID] = memo
	}

	// Fetch old memos for insights
	for i := range connections {
		newMemo := memoByUID[connections[i].NewMemo.MemoName]
		oldMemo, _ := g.store.GetMemo(ctx, &store.FindMemo{UID: &connections[i].OldMemo.MemoName})
		connections[i].Insight = GenerateInsight(newMemo, oldMemo, connections[i].Similarity)
	}

	// Identify themes
	themes := IdentifyThemes(thisWeekMemos, connections)

	// Generate LLM analysis if analyzer is available
	var analysis *AnalysisResult
	if g.analyzer != nil {
		var err error
		analysis, err = g.analyzer.AnalyzeMemos(thisWeekMemos, connections)
		if err != nil {
			slog.Warn("Failed to generate LLM analysis, using basic digest",
				"error", err, "user_id", user.ID)
		}
	}

	return &DigestContent{
		User:           user,
		WeekStart:      weekStart,
		WeekEnd:        weekEnd,
		ThisWeekMemos:  thisWeekMemos,
		Connections:    connections,
		Themes:         themes,
		TotalMemoCount: len(thisWeekMemos),
		Analysis:       analysis,
	}, nil
}

// fetchThisWeekMemos fetches memos created after the week start time.
func (g *Generator) fetchThisWeekMemos(ctx context.Context, userID int32, weekStart time.Time) ([]*store.Memo, error) {
	// Get all memos for the user
	allMemos, err := g.store.ListMemos(ctx, &store.FindMemo{
		CreatorID: &userID,
	})
	if err != nil {
		return nil, err
	}

	// Filter to this week's memos
	weekStartTs := weekStart.Unix()
	var thisWeekMemos []*store.Memo
	for _, memo := range allMemos {
		if memo.CreatedTs >= weekStartTs {
			thisWeekMemos = append(thisWeekMemos, memo)
		}
	}

	return thisWeekMemos, nil
}

// separateEmbeddings separates embeddings into this week's and previous.
func (g *Generator) separateEmbeddings(allEmbeddings []supabase.MemoEmbedding, thisWeekMemoNames []string, weekStart time.Time) (thisWeek, previous []supabase.MemoEmbedding) {
	// Create a set of this week's memo names
	thisWeekSet := make(map[string]bool)
	for _, name := range thisWeekMemoNames {
		thisWeekSet[name] = true
	}

	for _, emb := range allEmbeddings {
		if thisWeekSet[emb.MemoName] {
			thisWeek = append(thisWeek, emb)
		} else {
			previous = append(previous, emb)
		}
	}

	return thisWeek, previous
}

// TruncateContent truncates content to a maximum length for display.
func TruncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}

// FormatDateRange formats a date range for display.
func FormatDateRange(start, end time.Time) string {
	if start.Year() == end.Year() && start.Month() == end.Month() {
		return fmt.Sprintf("%s %d - %d, %d",
			start.Month().String()[:3],
			start.Day(),
			end.Day(),
			start.Year())
	}
	return fmt.Sprintf("%s %d - %s %d, %d",
		start.Month().String()[:3],
		start.Day(),
		end.Month().String()[:3],
		end.Day(),
		end.Year())
}
