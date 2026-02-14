package digest

import (
	"context"
	"log/slog"
	"os"
	"strconv"

	"github.com/pkg/errors"

	"github.com/usememos/memos/plugin/email"
	"github.com/usememos/memos/plugin/scheduler"
	"github.com/usememos/memos/store"
)

// DefaultSchedule is the default cron schedule for the digest (Sunday 8am UTC).
const DefaultSchedule = "0 8 * * 0"

// Runner manages the weekly digest email job.
type Runner struct {
	store     *store.Store
	scheduler *scheduler.Scheduler
	generator *Generator
	config    *Config
}

// Config holds the configuration for the digest runner.
type Config struct {
	// Enabled determines if the digest runner is active.
	Enabled bool
	// Schedule is the cron expression for when to send digests.
	Schedule string
	// AppURL is the base URL of the Memos application (for links in emails).
	AppURL string
	// EmailConfig is the SMTP configuration for sending emails.
	EmailConfig *email.Config
}

// NewRunner creates a new digest runner.
func NewRunner(store *store.Store) (*Runner, error) {
	config := loadConfigFromEnv()

	if !config.Enabled {
		return &Runner{
			store:  store,
			config: config,
		}, nil
	}

	generator, err := NewGenerator(store)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create digest generator")
	}

	sched := scheduler.New()

	return &Runner{
		store:     store,
		scheduler: sched,
		generator: generator,
		config:    config,
	}, nil
}

// loadConfigFromEnv loads digest configuration from environment variables.
func loadConfigFromEnv() *Config {
	enabled := os.Getenv("MEMOS_DIGEST_ENABLED") == "true"

	schedule := os.Getenv("MEMOS_DIGEST_SCHEDULE")
	if schedule == "" {
		schedule = DefaultSchedule
	}

	appURL := os.Getenv("MEMOS_APP_URL")
	if appURL == "" {
		appURL = "http://localhost:5230"
	}

	// Load SMTP config
	smtpPort := 587
	if portStr := os.Getenv("MEMOS_SMTP_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			smtpPort = p
		}
	}

	// Port 465 uses SSL, port 587 uses STARTTLS
	useSSL := smtpPort == 465
	useTLS := smtpPort == 587

	emailConfig := &email.Config{
		SMTPHost:     os.Getenv("MEMOS_SMTP_HOST"),
		SMTPPort:     smtpPort,
		SMTPUsername: os.Getenv("MEMOS_SMTP_USERNAME"),
		SMTPPassword: os.Getenv("MEMOS_SMTP_PASSWORD"),
		FromEmail:    os.Getenv("MEMOS_SMTP_FROM"),
		FromName:     "Memos Digest",
		UseTLS:       useTLS,
		UseSSL:       useSSL,
	}

	return &Config{
		Enabled:     enabled,
		Schedule:    schedule,
		AppURL:      appURL,
		EmailConfig: emailConfig,
	}
}

// Run starts the digest runner with the configured schedule.
func (r *Runner) Run(ctx context.Context) {
	if !r.config.Enabled {
		slog.Info("Digest runner is disabled")
		return
	}

	// Validate email config
	if err := r.config.EmailConfig.Validate(); err != nil {
		slog.Error("Invalid email configuration, digest runner disabled", "error", err)
		return
	}

	// Register the digest job
	job := &scheduler.Job{
		Name:        "weekly-digest",
		Schedule:    r.config.Schedule,
		Handler:     r.sendDigests,
		Description: "Send weekly digest emails to users",
	}

	if err := r.scheduler.Register(job); err != nil {
		slog.Error("Failed to register digest job", "error", err)
		return
	}

	// Start the scheduler
	if err := r.scheduler.Start(); err != nil {
		slog.Error("Failed to start digest scheduler", "error", err)
		return
	}

	slog.Info("Digest runner started", "schedule", r.config.Schedule)

	// Wait for context cancellation
	<-ctx.Done()

	// Stop the scheduler
	if err := r.scheduler.Stop(context.Background()); err != nil {
		slog.Error("Failed to stop digest scheduler", "error", err)
	}
}

// RunOnce executes a single digest run for all eligible users.
// Useful for testing or manual triggering.
func (r *Runner) RunOnce(ctx context.Context) error {
	if !r.config.Enabled {
		return errors.New("digest runner is disabled")
	}

	return r.sendDigests(ctx)
}

// sendDigests sends digest emails to all eligible users.
func (r *Runner) sendDigests(ctx context.Context) error {
	slog.Info("Starting weekly digest generation")

	// List all users with email addresses
	users, err := r.store.ListUsers(ctx, &store.FindUser{})
	if err != nil {
		slog.Error("Failed to list users for digest", "error", err)
		return errors.Wrap(err, "failed to list users")
	}

	var successCount, errorCount int

	for _, user := range users {
		// Skip users without email
		if user.Email == "" {
			continue
		}

		// Skip system bot
		if user.ID == store.SystemBotID {
			continue
		}

		// Generate digest for this user
		digest, err := r.generator.GenerateDigest(ctx, user)
		if err != nil {
			slog.Warn("Failed to generate digest for user",
				"user_id", user.ID,
				"error", err)
			errorCount++
			continue
		}

		// Skip if no activity this week
		if digest.TotalMemoCount == 0 {
			slog.Debug("Skipping digest for user with no activity", "user_id", user.ID)
			continue
		}

		// Render email content
		htmlContent, err := RenderEmailHTML(digest, r.config.AppURL)
		if err != nil {
			slog.Warn("Failed to render digest email",
				"user_id", user.ID,
				"error", err)
			errorCount++
			continue
		}

		// Send the email
		message := &email.Message{
			To:      []string{user.Email},
			Subject: "Your Weekly Memos Digest",
			Body:    htmlContent,
			IsHTML:  true,
		}

		// Send asynchronously to not block the digest generation
		email.SendAsync(r.config.EmailConfig, message)
		successCount++

		slog.Info("Sent digest email",
			"user_id", user.ID,
			"email", user.Email,
			"memo_count", digest.TotalMemoCount,
			"connections", len(digest.Connections))
	}

	slog.Info("Weekly digest generation completed",
		"success", successCount,
		"errors", errorCount)

	return nil
}
