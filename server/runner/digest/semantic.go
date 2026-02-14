package digest

import (
	"fmt"
	"math"
	"sort"

	"github.com/usememos/memos/plugin/supabase"
	"github.com/usememos/memos/store"
)

// Connection represents a semantic connection between two memos.
type Connection struct {
	NewMemo    supabase.MemoEmbedding
	OldMemo    supabase.MemoEmbedding
	Similarity float64
	Insight    string
}

// ConnectionThreshold is the minimum cosine similarity for a connection.
const ConnectionThreshold = 0.4

// MaxConnections is the maximum number of connections to return.
const MaxConnections = 5

// CosineSimilarity computes the cosine similarity between two embedding vectors.
// Returns a value between -1 and 1, where 1 means identical direction.
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// FindSemanticConnections finds semantic connections between this week's memos
// and all previous memos based on embedding similarity.
func FindSemanticConnections(thisWeek, allPrevious []supabase.MemoEmbedding) []Connection {
	var connections []Connection

	// Create a set of this week's memo names to avoid self-connections
	thisWeekNames := make(map[string]bool)
	for _, emb := range thisWeek {
		thisWeekNames[emb.MemoName] = true
	}

	for _, newEmb := range thisWeek {
		for _, oldEmb := range allPrevious {
			// Skip if same memo or if oldEmb is from this week
			if newEmb.MemoName == oldEmb.MemoName || thisWeekNames[oldEmb.MemoName] {
				continue
			}

			sim := CosineSimilarity(newEmb.Embedding, oldEmb.Embedding)
			if sim >= ConnectionThreshold {
				connections = append(connections, Connection{
					NewMemo:    newEmb,
					OldMemo:    oldEmb,
					Similarity: sim,
				})
			}
		}
	}

	// Sort by similarity descending
	sort.Slice(connections, func(i, j int) bool {
		return connections[i].Similarity > connections[j].Similarity
	})

	// Take top connections
	if len(connections) > MaxConnections {
		connections = connections[:MaxConnections]
	}

	return connections
}

// GenerateInsight creates a human-readable insight about the connection
// based on the time difference between the memos.
func GenerateInsight(newMemo, oldMemo *store.Memo, similarity float64) string {
	if newMemo == nil || oldMemo == nil {
		return "A surprising connection!"
	}

	// Calculate days difference
	daysDiff := (newMemo.CreatedTs - oldMemo.CreatedTs) / 86400

	switch {
	case daysDiff < 7:
		return "A thought evolving in your mind"
	case daysDiff < 30:
		return fmt.Sprintf("Connects to %d days ago", daysDiff)
	case daysDiff < 365:
		months := daysDiff / 30
		if months == 1 {
			return "Echoes something from a month ago"
		}
		return fmt.Sprintf("Echoes something from %d months ago", months)
	default:
		years := daysDiff / 365
		if years == 1 {
			return "Links to an idea from over a year ago!"
		}
		return fmt.Sprintf("Links to an idea from %d years ago!", years)
	}
}

// ThemeCluster represents a group of related memos.
type ThemeCluster struct {
	Theme     string
	MemoCount int
	IsNew     bool // true if this is a newly emerging theme
}

// IdentifyThemes analyzes memos to identify emerging themes.
// This is a simple implementation based on content keywords.
func IdentifyThemes(thisWeekMemos []*store.Memo, connections []Connection) []ThemeCluster {
	// For now, we return a simple summary based on memo count
	// A more sophisticated implementation would use NLP or the LLM
	var themes []ThemeCluster

	if len(thisWeekMemos) >= 3 {
		themes = append(themes, ThemeCluster{
			Theme:     "Active week",
			MemoCount: len(thisWeekMemos),
			IsNew:     false,
		})
	}

	if len(connections) >= 2 {
		themes = append(themes, ThemeCluster{
			Theme:     "Building on past ideas",
			MemoCount: len(connections),
			IsNew:     false,
		})
	}

	return themes
}
