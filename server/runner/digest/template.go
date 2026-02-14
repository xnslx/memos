package digest

import (
	"bytes"
	"fmt"
	"html/template"
)

// EmailTemplate generates HTML email content for the weekly digest.
const emailTemplateHTML = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Your Weekly Memos Digest</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            line-height: 1.7;
            color: #333;
            max-width: 650px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            background-color: #ffffff;
            border-radius: 12px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 35px 25px;
            text-align: center;
        }
        .header h1 {
            margin: 0 0 8px 0;
            font-size: 26px;
            font-weight: 600;
        }
        .header .date-range {
            opacity: 0.9;
            font-size: 15px;
        }
        .section {
            padding: 25px;
            border-bottom: 1px solid #eee;
        }
        .section:last-child {
            border-bottom: none;
        }
        .section-title {
            font-size: 13px;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 1.5px;
            color: #667eea;
            margin: 0 0 18px 0;
            padding-bottom: 8px;
            border-bottom: 2px solid #667eea;
            display: inline-block;
        }
        .stat-row {
            display: flex;
            align-items: baseline;
            margin-bottom: 15px;
        }
        .stat-highlight {
            font-size: 42px;
            font-weight: 700;
            color: #667eea;
            margin-right: 10px;
        }
        .stat-label {
            color: #666;
            font-size: 16px;
        }
        .summary-text {
            font-size: 15px;
            color: #444;
            line-height: 1.8;
            margin-bottom: 15px;
        }
        .summary-text p {
            margin: 0 0 12px 0;
        }
        .theme-card {
            background: linear-gradient(135deg, #f8f9ff 0%, #f0f4ff 100%);
            border-radius: 10px;
            padding: 18px;
            margin-bottom: 15px;
            border-left: 4px solid #667eea;
        }
        .theme-card:last-child {
            margin-bottom: 0;
        }
        .theme-name {
            font-size: 16px;
            font-weight: 600;
            color: #333;
            margin-bottom: 8px;
        }
        .theme-description {
            font-size: 14px;
            color: #555;
            line-height: 1.6;
        }
        .theme-count {
            display: inline-block;
            background-color: #667eea;
            color: white;
            padding: 3px 10px;
            border-radius: 12px;
            font-size: 12px;
            font-weight: 600;
            margin-top: 10px;
        }
        .connection-card {
            background-color: #f8f9fa;
            border-radius: 10px;
            padding: 20px;
            margin-bottom: 18px;
        }
        .connection-card:last-child {
            margin-bottom: 0;
        }
        .connection-memos {
            display: flex;
            gap: 15px;
            margin-bottom: 15px;
        }
        .connection-memo {
            flex: 1;
            background: white;
            border-radius: 8px;
            padding: 12px;
            font-size: 13px;
            color: #555;
            border: 1px solid #e0e0e0;
        }
        .connection-memo-label {
            font-size: 10px;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 6px;
            color: #667eea;
        }
        .connection-memo-label.old {
            color: #888;
        }
        .connection-analysis {
            font-size: 14px;
            color: #444;
            line-height: 1.7;
            margin-bottom: 12px;
        }
        .connection-significance {
            font-size: 13px;
            color: #764ba2;
            font-style: italic;
            padding-left: 15px;
            border-left: 3px solid #764ba2;
        }
        .advice-card {
            background: linear-gradient(135deg, #fff9f0 0%, #fff5e6 100%);
            border-radius: 10px;
            padding: 18px;
            margin-bottom: 15px;
            border-left: 4px solid #f5a623;
        }
        .advice-card:last-child {
            margin-bottom: 0;
        }
        .advice-category {
            display: inline-block;
            background-color: #f5a623;
            color: white;
            padding: 3px 10px;
            border-radius: 12px;
            font-size: 11px;
            font-weight: 600;
            text-transform: uppercase;
            margin-bottom: 10px;
        }
        .advice-suggestion {
            font-size: 15px;
            font-weight: 600;
            color: #333;
            margin-bottom: 10px;
        }
        .advice-rationale {
            font-size: 14px;
            color: #555;
            line-height: 1.6;
        }
        .reflection-box {
            background: linear-gradient(135deg, #e8f5e9 0%, #c8e6c9 100%);
            border-radius: 10px;
            padding: 22px;
            text-align: center;
        }
        .reflection-icon {
            font-size: 28px;
            margin-bottom: 12px;
        }
        .reflection-text {
            font-size: 16px;
            color: #2e7d32;
            font-style: italic;
            line-height: 1.7;
        }
        .looking-ahead-box {
            background: linear-gradient(135deg, #e3f2fd 0%, #bbdefb 100%);
            border-radius: 10px;
            padding: 20px;
        }
        .looking-ahead-title {
            font-size: 14px;
            font-weight: 600;
            color: #1565c0;
            margin-bottom: 12px;
        }
        .looking-ahead-text {
            font-size: 14px;
            color: #1976d2;
            line-height: 1.7;
        }
        .cta-button {
            display: inline-block;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            text-decoration: none;
            padding: 14px 35px;
            border-radius: 25px;
            font-weight: 600;
            font-size: 15px;
            margin: 15px 0;
        }
        .footer {
            text-align: center;
            padding: 25px;
            color: #888;
            font-size: 12px;
        }
        .no-activity {
            text-align: center;
            color: #666;
            padding: 30px 0;
        }
        .divider {
            height: 1px;
            background: linear-gradient(90deg, transparent, #ddd, transparent);
            margin: 20px 0;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Your Weekly Memos Digest</h1>
            <div class="date-range">{{.DateRange}}</div>
        </div>

        {{if gt .MemoCount 0}}
        <div class="section">
            <div class="stat-row">
                <span class="stat-highlight">{{.MemoCount}}</span>
                <span class="stat-label">memo{{if ne .MemoCount 1}}s{{end}} captured this week</span>
            </div>
        </div>

        {{if .HasAnalysis}}
        <!-- Weekly Summary from LLM -->
        <div class="section">
            <div class="section-title">Weekly Summary</div>
            <div class="summary-text">{{.WeeklySummary}}</div>
        </div>

        <!-- Key Themes -->
        {{if .HasThemes}}
        <div class="section">
            <div class="section-title">Key Themes This Week</div>
            {{range .AnalysisThemes}}
            <div class="theme-card">
                <div class="theme-name">{{.Name}}</div>
                <div class="theme-description">{{.Description}}</div>
                {{if gt .MemoCount 0}}
                <span class="theme-count">{{.MemoCount}} memo{{if ne .MemoCount 1}}s{{end}}</span>
                {{end}}
            </div>
            {{end}}
        </div>
        {{end}}

        <!-- Semantic Connections with Analysis -->
        {{if .HasConnections}}
        <div class="section">
            <div class="section-title">Connections Discovered</div>
            {{range .AnalysisConnections}}
            <div class="connection-card">
                <div class="connection-memos">
                    <div class="connection-memo">
                        <div class="connection-memo-label">This Week</div>
                        {{.NewMemoExcerpt}}
                    </div>
                    <div class="connection-memo">
                        <div class="connection-memo-label old">Connected To</div>
                        {{.OldMemoExcerpt}}
                    </div>
                </div>
                <div class="connection-analysis">{{.Analysis}}</div>
                {{if .Significance}}
                <div class="connection-significance">{{.Significance}}</div>
                {{end}}
            </div>
            {{end}}
        </div>
        {{end}}

        <!-- Actionable Advice -->
        {{if .HasAdvice}}
        <div class="section">
            <div class="section-title">Actionable Insights</div>
            {{range .Advice}}
            <div class="advice-card">
                <span class="advice-category">{{.Category}}</span>
                <div class="advice-suggestion">{{.Suggestion}}</div>
                <div class="advice-rationale">{{.Rationale}}</div>
            </div>
            {{end}}
        </div>
        {{end}}

        <!-- Reflection Prompt -->
        {{if .Reflection}}
        <div class="section">
            <div class="section-title">Reflection</div>
            <div class="reflection-box">
                <div class="reflection-icon">💭</div>
                <div class="reflection-text">{{.Reflection}}</div>
            </div>
        </div>
        {{end}}

        <!-- Looking Ahead -->
        {{if .LookingAhead}}
        <div class="section">
            <div class="section-title">Looking Ahead</div>
            <div class="looking-ahead-box">
                <div class="looking-ahead-text">{{.LookingAhead}}</div>
            </div>
        </div>
        {{end}}

        {{else}}
        <!-- Fallback to basic template when no LLM analysis -->
        {{if .HasBasicConnections}}
        <div class="section">
            <div class="section-title">Connections Discovered</div>
            {{range .BasicConnections}}
            <div class="connection-card">
                <div class="connection-memos">
                    <div class="connection-memo">
                        <div class="connection-memo-label">New</div>
                        {{.NewContent}}
                    </div>
                    <div class="connection-memo">
                        <div class="connection-memo-label old">Related</div>
                        {{.OldContent}}
                    </div>
                </div>
                <div class="connection-analysis">{{.SimilarityPercent}}% similar - {{.Insight}}</div>
            </div>
            {{end}}
        </div>
        {{end}}
        {{end}}

        {{else}}
        <div class="section">
            <div class="no-activity">
                <p style="font-size: 18px; margin-bottom: 10px;">📝</p>
                <p>No memos this week. Time to capture some thoughts!</p>
            </div>
        </div>
        {{end}}

        <div class="section" style="text-align: center;">
            <a href="{{.AppURL}}" class="cta-button">Open Memos</a>
        </div>

        <div class="footer">
            <p>This digest was thoughtfully generated by Memos with AI assistance.</p>
            <p>You're receiving this because you have digest emails enabled.</p>
        </div>
    </div>
</body>
</html>`

// TemplateData contains the data needed to render the email template.
type TemplateData struct {
	DateRange      string
	MemoCount      int
	AppURL         string

	// LLM Analysis (preferred)
	HasAnalysis         bool
	WeeklySummary       string
	HasThemes           bool
	AnalysisThemes      []Theme
	HasConnections      bool
	AnalysisConnections []ConnectionInsight
	HasAdvice           bool
	Advice              []Advice
	Reflection          string
	LookingAhead        string

	// Basic fallback (no LLM)
	HasBasicConnections bool
	BasicConnections    []ConnectionDisplay
}

// ConnectionDisplay represents a connection formatted for display (basic mode).
type ConnectionDisplay struct {
	NewContent        string
	OldContent        string
	SimilarityPercent int
	Insight           string
}

// RenderEmailHTML renders the digest content as an HTML email.
func RenderEmailHTML(digest *DigestContent, appURL string) (string, error) {
	if digest == nil {
		return "", fmt.Errorf("digest content is required")
	}

	data := TemplateData{
		DateRange: FormatDateRange(digest.WeekStart, digest.WeekEnd),
		MemoCount: digest.TotalMemoCount,
		AppURL:    appURL,
	}

	// Use LLM analysis if available
	if digest.Analysis != nil {
		data.HasAnalysis = true
		data.WeeklySummary = digest.Analysis.WeeklySummary
		data.HasThemes = len(digest.Analysis.KeyThemes) > 0
		data.AnalysisThemes = digest.Analysis.KeyThemes
		data.HasConnections = len(digest.Analysis.Connections) > 0
		data.AnalysisConnections = digest.Analysis.Connections
		data.HasAdvice = len(digest.Analysis.ActionableAdvice) > 0
		data.Advice = digest.Analysis.ActionableAdvice
		data.Reflection = digest.Analysis.Reflection
		data.LookingAhead = digest.Analysis.LookingAhead
	} else {
		// Fallback to basic connections
		var connections []ConnectionDisplay
		for _, conn := range digest.Connections {
			connections = append(connections, ConnectionDisplay{
				NewContent:        TruncateContent(conn.NewMemo.Content, 100),
				OldContent:        TruncateContent(conn.OldMemo.Content, 100),
				SimilarityPercent: int(conn.Similarity * 100),
				Insight:           conn.Insight,
			})
		}
		data.HasBasicConnections = len(connections) > 0
		data.BasicConnections = connections
	}

	tmpl, err := template.New("digest").Parse(emailTemplateHTML)
	if err != nil {
		return "", fmt.Errorf("failed to parse email template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute email template: %w", err)
	}

	return buf.String(), nil
}

// RenderPlainText renders a plain text version of the digest.
func RenderPlainText(digest *DigestContent, appURL string) string {
	if digest == nil {
		return "No digest content available."
	}

	var buf bytes.Buffer

	buf.WriteString("YOUR WEEKLY MEMOS DIGEST\n")
	buf.WriteString(FormatDateRange(digest.WeekStart, digest.WeekEnd) + "\n")
	buf.WriteString("========================\n\n")

	buf.WriteString(fmt.Sprintf("THIS WEEK: %d memo(s) captured\n\n", digest.TotalMemoCount))

	// Use LLM analysis if available
	if digest.Analysis != nil {
		buf.WriteString("WEEKLY SUMMARY\n")
		buf.WriteString("--------------\n")
		buf.WriteString(digest.Analysis.WeeklySummary + "\n\n")

		if len(digest.Analysis.KeyThemes) > 0 {
			buf.WriteString("KEY THEMES\n")
			buf.WriteString("----------\n")
			for _, theme := range digest.Analysis.KeyThemes {
				buf.WriteString(fmt.Sprintf("• %s: %s\n", theme.Name, theme.Description))
			}
			buf.WriteString("\n")
		}

		if len(digest.Analysis.Connections) > 0 {
			buf.WriteString("CONNECTIONS DISCOVERED\n")
			buf.WriteString("----------------------\n")
			for i, conn := range digest.Analysis.Connections {
				buf.WriteString(fmt.Sprintf("\n%d. New: %s\n", i+1, conn.NewMemoExcerpt))
				buf.WriteString(fmt.Sprintf("   Connected to: %s\n", conn.OldMemoExcerpt))
				buf.WriteString(fmt.Sprintf("   Analysis: %s\n", conn.Analysis))
				if conn.Significance != "" {
					buf.WriteString(fmt.Sprintf("   Why it matters: %s\n", conn.Significance))
				}
			}
			buf.WriteString("\n")
		}

		if len(digest.Analysis.ActionableAdvice) > 0 {
			buf.WriteString("ACTIONABLE INSIGHTS\n")
			buf.WriteString("-------------------\n")
			for _, advice := range digest.Analysis.ActionableAdvice {
				buf.WriteString(fmt.Sprintf("\n[%s]\n", advice.Category))
				buf.WriteString(fmt.Sprintf("→ %s\n", advice.Suggestion))
				buf.WriteString(fmt.Sprintf("  %s\n", advice.Rationale))
			}
			buf.WriteString("\n")
		}

		if digest.Analysis.Reflection != "" {
			buf.WriteString("REFLECTION\n")
			buf.WriteString("----------\n")
			buf.WriteString(digest.Analysis.Reflection + "\n\n")
		}

		if digest.Analysis.LookingAhead != "" {
			buf.WriteString("LOOKING AHEAD\n")
			buf.WriteString("-------------\n")
			buf.WriteString(digest.Analysis.LookingAhead + "\n\n")
		}
	} else {
		// Basic fallback
		if len(digest.Connections) > 0 {
			buf.WriteString("CONNECTIONS DISCOVERED\n")
			buf.WriteString("----------------------\n")
			for i, conn := range digest.Connections {
				buf.WriteString(fmt.Sprintf("\n%d. New: %s\n", i+1, TruncateContent(conn.NewMemo.Content, 80)))
				buf.WriteString(fmt.Sprintf("   Related: %s\n", TruncateContent(conn.OldMemo.Content, 80)))
				buf.WriteString(fmt.Sprintf("   %d%% similar - %s\n", int(conn.Similarity*100), conn.Insight))
			}
			buf.WriteString("\n")
		}
	}

	buf.WriteString(fmt.Sprintf("Open Memos: %s\n", appURL))

	return buf.String()
}
