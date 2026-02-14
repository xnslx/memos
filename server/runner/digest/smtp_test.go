package digest

import (
	"os"
	"testing"

	"github.com/usememos/memos/plugin/email"
	"github.com/usememos/memos/plugin/supabase"
	"github.com/usememos/memos/store"
)

// TestSMTPResend tests sending an email via Resend SMTP.
// Run with: go test -v -run TestSMTPResend ./server/runner/digest/
func TestSMTPResend(t *testing.T) {
	// Resend SMTP config
	config := &email.Config{
		SMTPHost:     "smtp.resend.com",
		SMTPPort:     465,
		SMTPUsername: "resend",
		SMTPPassword: "re_SM575znx_D2t7fA5DNqZq1ny8a9ExPMe1",
		FromEmail:    "onboarding@resend.dev", // Resend's test domain
		FromName:     "Memos Digest Test",
		UseSSL:       true,
	}

	// Create a test digest (basic, no LLM)
	digest := &DigestContent{
		WeekStart:      parseTime("2026-02-03"),
		WeekEnd:        parseTime("2026-02-09"),
		TotalMemoCount: 3,
		Connections: []Connection{
			{
				NewMemo:    supabaseMemoEmbeddingForTest("New thought about productivity"),
				OldMemo:    supabaseMemoEmbeddingForTest("Old GTD notes from last month"),
				Similarity: 0.78,
				Insight:    "Building on your productivity thinking!",
			},
		},
		Themes: []ThemeCluster{
			{Theme: "Productivity", MemoCount: 3, IsNew: false},
		},
	}

	// Render the email
	htmlContent, err := RenderEmailHTML(digest, "http://localhost:5230")
	if err != nil {
		t.Fatalf("Failed to render email: %v", err)
	}

	recipientEmail := "xianlistudio@gmail.com"

	message := &email.Message{
		To:      []string{recipientEmail},
		Subject: "Test: Your Weekly Memos Digest (Basic)",
		Body:    htmlContent,
		IsHTML:  true,
	}

	t.Logf("Sending test email to %s via Resend...", recipientEmail)

	if err := email.Send(config, message); err != nil {
		t.Fatalf("Failed to send email: %v", err)
	}

	t.Log("Email sent successfully! Check Resend dashboard for delivery status.")
}

// TestSMTPResendWithLLM tests sending an email with LLM-generated analysis.
// Run with: OPENAI_API_KEY=xxx go test -v -run TestSMTPResendWithLLM ./server/runner/digest/
func TestSMTPResendWithLLM(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("Skipping LLM test. Set OPENAI_API_KEY to run.")
	}

	// Resend SMTP config
	smtpConfig := &email.Config{
		SMTPHost:     "smtp.resend.com",
		SMTPPort:     465,
		SMTPUsername: "resend",
		SMTPPassword: "re_SM575znx_D2t7fA5DNqZq1ny8a9ExPMe1",
		FromEmail:    "onboarding@resend.dev",
		FromName:     "Memos Digest Test",
		UseSSL:       true,
	}

	// Create sample memos for analysis
	sampleMemos := []*store.Memo{
		{
			UID:       "memo-1",
			Content:   "Been thinking about how to improve my morning routine. Wake up at 6am, do 20 minutes of meditation, then review my goals for the day. The key is consistency - doing it even when I don't feel like it.",
			CreatedTs: 1707200000,
		},
		{
			UID:       "memo-2",
			Content:   "Read an interesting article about second brain methodology. The idea of externalizing your thoughts into a trusted system resonates with me. Need to explore Zettelkasten more.",
			CreatedTs: 1707300000,
		},
		{
			UID:       "memo-3",
			Content:   "Project retrospective: What worked well was breaking down tasks into smaller chunks. What didn't work was trying to multitask. Next time, focus on one thing at a time.",
			CreatedTs: 1707400000,
		},
		{
			UID:       "memo-4",
			Content:   "Idea for the app: add a weekly review feature that summarizes what I've been thinking about. Could use AI to find patterns I might have missed.",
			CreatedTs: 1707500000,
		},
	}

	// Create sample connections
	connections := []Connection{
		{
			NewMemo: supabase.MemoEmbedding{
				MemoName: "memo-1",
				Content:  "Been thinking about how to improve my morning routine...",
			},
			OldMemo: supabase.MemoEmbedding{
				MemoName: "old-memo-habits",
				Content:  "Habits are built through repetition. The cue-routine-reward loop from Atomic Habits.",
			},
			Similarity: 0.72,
		},
		{
			NewMemo: supabase.MemoEmbedding{
				MemoName: "memo-2",
				Content:  "Read an interesting article about second brain methodology...",
			},
			OldMemo: supabase.MemoEmbedding{
				MemoName: "old-memo-pkm",
				Content:  "Personal knowledge management is about capturing, organizing, and retrieving information effectively.",
			},
			Similarity: 0.85,
		},
	}

	// Create analyzer and run analysis
	t.Log("Calling LLM for analysis...")
	analyzer, err := NewAnalyzer()
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}

	analysis, err := analyzer.AnalyzeMemos(sampleMemos, connections)
	if err != nil {
		t.Fatalf("Failed to analyze memos: %v", err)
	}

	t.Logf("LLM Analysis received:")
	t.Logf("  Summary length: %d chars", len(analysis.WeeklySummary))
	t.Logf("  Themes: %d", len(analysis.KeyThemes))
	t.Logf("  Connections: %d", len(analysis.Connections))
	t.Logf("  Advice items: %d", len(analysis.ActionableAdvice))

	// Create digest with LLM analysis
	digest := &DigestContent{
		WeekStart:      parseTime("2026-02-03"),
		WeekEnd:        parseTime("2026-02-09"),
		TotalMemoCount: len(sampleMemos),
		ThisWeekMemos:  sampleMemos,
		Connections:    connections,
		Analysis:       analysis,
	}

	// Render the email
	htmlContent, err := RenderEmailHTML(digest, "http://localhost:5230")
	if err != nil {
		t.Fatalf("Failed to render email: %v", err)
	}

	recipientEmail := "xianlistudio@gmail.com"

	message := &email.Message{
		To:      []string{recipientEmail},
		Subject: "Test: Your Weekly Memos Digest (with AI Analysis)",
		Body:    htmlContent,
		IsHTML:  true,
	}

	t.Logf("Sending LLM-enhanced email to %s via Resend...", recipientEmail)

	if err := email.Send(smtpConfig, message); err != nil {
		t.Fatalf("Failed to send email: %v", err)
	}

	t.Log("LLM-enhanced email sent successfully!")
}

func supabaseMemoEmbeddingForTest(content string) supabase.MemoEmbedding {
	return supabase.MemoEmbedding{
		MemoName: "memo-test",
		Content:  content,
	}
}
