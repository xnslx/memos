package digest

import (
	"os"
	"testing"
	"time"

	"github.com/usememos/memos/plugin/supabase"
	"github.com/usememos/memos/store"
)

// TestDigestManual is a manual test for the digest runner.
// Run with: go test -v -run TestDigestManual ./server/runner/digest/
//
// Required env vars:
//   MEMOS_DIGEST_ENABLED=true
//   SUPABASE_URL=https://your-project.supabase.co
//   SUPABASE_SERVICE_KEY=your-service-role-key
//   MEMOS_SMTP_HOST=smtp.gmail.com (or your SMTP host)
//   MEMOS_SMTP_PORT=587
//   MEMOS_SMTP_USERNAME=your-email
//   MEMOS_SMTP_PASSWORD=your-password
//   MEMOS_SMTP_FROM=noreply@example.com
//   MEMOS_APP_URL=http://localhost:5230
//
// For testing without sending real emails, use Mailtrap:
//   MEMOS_SMTP_HOST=sandbox.smtp.mailtrap.io
//   MEMOS_SMTP_PORT=587
//   MEMOS_SMTP_USERNAME=<mailtrap-username>
//   MEMOS_SMTP_PASSWORD=<mailtrap-password>
func TestDigestManual(t *testing.T) {
	if os.Getenv("MEMOS_DIGEST_ENABLED") != "true" {
		t.Skip("Skipping manual digest test. Set MEMOS_DIGEST_ENABLED=true to run.")
	}

	// This test requires a running database with the store
	// For a full integration test, you'd need to set up the store
	t.Log("To run a full integration test, use the main application with RunOnce()")
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{-1, 0, 0},
			expected: -1.0,
		},
		{
			name:     "similar vectors",
			a:        []float64{1, 1, 0},
			b:        []float64{1, 0, 0},
			expected: 0.7071067811865475, // 1/sqrt(2)
		},
		{
			name:     "empty vectors",
			a:        []float64{},
			b:        []float64{},
			expected: 0.0,
		},
		{
			name:     "mismatched lengths",
			a:        []float64{1, 2},
			b:        []float64{1, 2, 3},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CosineSimilarity(tt.a, tt.b)
			if !floatEquals(result, tt.expected, 0.0001) {
				t.Errorf("CosineSimilarity(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestGenerateInsight(t *testing.T) {
	now := int64(1707500000) // Feb 9, 2024

	tests := []struct {
		name      string
		newTs     int64
		oldTs     int64
		expected  string
	}{
		{
			name:     "less than a week",
			newTs:    now,
			oldTs:    now - 3*86400, // 3 days ago
			expected: "A thought evolving in your mind",
		},
		{
			name:     "about two weeks",
			newTs:    now,
			oldTs:    now - 14*86400, // 14 days ago
			expected: "Connects to 14 days ago",
		},
		{
			name:     "about two months",
			newTs:    now,
			oldTs:    now - 60*86400, // 60 days ago
			expected: "Echoes something from 2 months ago",
		},
		{
			name:     "over a year",
			newTs:    now,
			oldTs:    now - 400*86400, // 400 days ago
			expected: "Links to an idea from over a year ago!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newMemo := toStoreMemo(tt.newTs)
			oldMemo := toStoreMemo(tt.oldTs)
			result := GenerateInsight(newMemo, oldMemo, 0.5)
			if result != tt.expected {
				t.Errorf("GenerateInsight() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFindSemanticConnections(t *testing.T) {
	thisWeekEmb := []supabase.MemoEmbedding{
		{MemoName: "memo1", Embedding: []float64{1, 0, 0}},
		{MemoName: "memo2", Embedding: []float64{0, 1, 0}},
	}

	prevEmb := []supabase.MemoEmbedding{
		{MemoName: "old1", Embedding: []float64{0.9, 0.1, 0}},  // Similar to memo1
		{MemoName: "old2", Embedding: []float64{0, 0, 1}},       // Not similar to either
		{MemoName: "old3", Embedding: []float64{0.1, 0.9, 0.1}}, // Similar to memo2
	}

	connections := FindSemanticConnections(thisWeekEmb, prevEmb)

	// Should find connections above threshold (0.4)
	if len(connections) == 0 {
		t.Error("Expected to find some connections")
	}

	// Check that connections are sorted by similarity
	for i := 1; i < len(connections); i++ {
		if connections[i].Similarity > connections[i-1].Similarity {
			t.Error("Connections should be sorted by similarity descending")
		}
	}
}

func TestRenderEmailHTML(t *testing.T) {
	digest := &DigestContent{
		WeekStart:      parseTime("2024-02-03"),
		WeekEnd:        parseTime("2024-02-09"),
		TotalMemoCount: 5,
		Connections:    nil,
		Themes:         nil,
	}

	html, err := RenderEmailHTML(digest, "http://localhost:5230")
	if err != nil {
		t.Fatalf("RenderEmailHTML() error = %v", err)
	}

	// Check for key elements
	if !contains(html, "Weekly Memos Digest") {
		t.Error("HTML should contain title")
	}
	if !contains(html, "5") {
		t.Error("HTML should contain memo count")
	}
	if !contains(html, "http://localhost:5230") {
		t.Error("HTML should contain app URL")
	}
}

// Helper functions for testing

func toStoreMemo(createdTs int64) *store.Memo {
	return &store.Memo{CreatedTs: createdTs}
}


func floatEquals(a, b, epsilon float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
}

func parseTime(s string) (t time.Time) {
	t, _ = time.Parse("2006-01-02", s)
	return t
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
