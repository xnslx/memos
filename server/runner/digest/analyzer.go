package digest

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/usememos/memos/plugin/openai"
	"github.com/usememos/memos/store"
)

// Analyzer uses LLM to generate valuable insights from memos.
type Analyzer struct {
	client *openai.Client
}

// AnalysisResult contains the LLM-generated analysis in structured format.
type AnalysisResult struct {
	XMLName          xml.Name          `xml:"analysis"`
	WeeklySummary    string            `xml:"weekly_summary"`
	KeyThemes        []Theme           `xml:"themes>theme"`
	Connections      []ConnectionInsight `xml:"connections>connection"`
	ActionableAdvice []Advice          `xml:"advice>item"`
	Reflection       string            `xml:"reflection"`
	LookingAhead     string            `xml:"looking_ahead"`
}

// Theme represents an identified theme in the week's notes.
type Theme struct {
	Name        string `xml:"name"`
	Description string `xml:"description"`
	MemoCount   int    `xml:"memo_count"`
}

// ConnectionInsight represents an LLM-analyzed connection between memos.
type ConnectionInsight struct {
	NewMemoExcerpt string `xml:"new_memo"`
	OldMemoExcerpt string `xml:"old_memo"`
	Analysis       string `xml:"analysis"`
	Significance   string `xml:"significance"`
}

// Advice represents actionable advice based on the notes.
type Advice struct {
	Category    string `xml:"category"`
	Suggestion  string `xml:"suggestion"`
	Rationale   string `xml:"rationale"`
}

// NewAnalyzer creates a new analyzer with OpenAI client.
func NewAnalyzer() (*Analyzer, error) {
	client, err := openai.NewClient()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create OpenAI client")
	}

	return &Analyzer{client: client}, nil
}

// AnalyzeMemos generates a comprehensive analysis of the user's memos.
func (a *Analyzer) AnalyzeMemos(thisWeekMemos []*store.Memo, connections []Connection) (*AnalysisResult, error) {
	prompt := a.buildPrompt(thisWeekMemos, connections)

	messages := []openai.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Use 8000 tokens to allow room for reasoning + output (reasoning models need more tokens)
	response, err := a.client.Chat(messages, 8000)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get LLM analysis")
	}

	// Parse XML response
	result, err := a.parseResponse(response)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse LLM response")
	}

	return result, nil
}

const systemPrompt = `You are a thoughtful personal knowledge assistant helping users gain insights from their notes and memos. Your role is to:

1. Identify patterns and themes across their thinking
2. Find meaningful connections between new and old ideas
3. Provide actionable advice based on what they're exploring
4. Offer gentle reflection prompts to deepen their understanding
5. Help them see the bigger picture of their intellectual journey

Be warm, insightful, and genuinely helpful. Write in a conversational but professional tone.
Focus on providing real value - not generic advice, but specific insights tied to THEIR notes.

You MUST respond in valid XML format as specified in the user prompt.`

func (a *Analyzer) buildPrompt(memos []*store.Memo, connections []Connection) string {
	var sb strings.Builder

	sb.WriteString("Please analyze the following notes from this week and provide a comprehensive weekly digest.\n\n")

	// Add this week's memos
	sb.WriteString("## THIS WEEK'S NOTES\n\n")
	for i, memo := range memos {
		content := TruncateContent(memo.Content, 500)
		sb.WriteString(fmt.Sprintf("### Note %d\n%s\n\n", i+1, content))
	}

	// Add semantic connections found
	if len(connections) > 0 {
		sb.WriteString("## SEMANTIC CONNECTIONS FOUND\n")
		sb.WriteString("These new notes are semantically similar to older notes:\n\n")
		for i, conn := range connections {
			sb.WriteString(fmt.Sprintf("### Connection %d (%.0f%% similar)\n", i+1, conn.Similarity*100))
			sb.WriteString(fmt.Sprintf("NEW: %s\n", TruncateContent(conn.NewMemo.Content, 300)))
			sb.WriteString(fmt.Sprintf("OLD: %s\n\n", TruncateContent(conn.OldMemo.Content, 300)))
		}
	}

	sb.WriteString(`## YOUR TASK

Analyze these notes and provide a valuable weekly digest. Respond in the following XML format:

<analysis>
  <weekly_summary>
    A 2-3 paragraph summary of what the user explored this week. Be specific about the topics and ideas. Highlight what seems most important or interesting. (150-200 words)
  </weekly_summary>

  <themes>
    <theme>
      <name>Theme name</name>
      <description>What this theme is about and why it matters (50-75 words)</description>
      <memo_count>Number of memos related to this theme</memo_count>
    </theme>
    <!-- Include 2-4 themes -->
  </themes>

  <connections>
    <connection>
      <new_memo>Brief excerpt or description of the new memo</new_memo>
      <old_memo>Brief excerpt or description of the connected old memo</old_memo>
      <analysis>Deep analysis of how these ideas connect and what it reveals about the user's thinking (75-100 words)</analysis>
      <significance>Why this connection matters for their personal/professional growth</significance>
    </connection>
    <!-- Include analysis for each semantic connection provided -->
  </connections>

  <advice>
    <item>
      <category>Category (e.g., Learning, Productivity, Health, Career, Creativity)</category>
      <suggestion>Specific, actionable suggestion based on their notes (1-2 sentences)</suggestion>
      <rationale>Why this advice is relevant based on what you observed in their notes (2-3 sentences)</rationale>
    </item>
    <!-- Include 3-5 actionable advice items -->
  </advice>

  <reflection>
    A thoughtful reflection prompt or question to help them think deeper about a pattern you noticed. This should be specific to their notes, not generic. (50-75 words)
  </reflection>

  <looking_ahead>
    Based on the trajectory of their thinking, suggest what they might want to explore next week. Be specific and tie it to themes you identified. (75-100 words)
  </looking_ahead>
</analysis>

Important:
- Be specific to THEIR notes, not generic
- Provide genuinely useful insights they couldn't easily see themselves
- Keep the total response under 1000 words
- Ensure valid XML format
`)

	return sb.String()
}

func (a *Analyzer) parseResponse(response string) (*AnalysisResult, error) {
	// Extract XML from response (in case there's text before/after)
	startIdx := strings.Index(response, "<analysis>")
	endIdx := strings.LastIndex(response, "</analysis>")

	if startIdx == -1 || endIdx == -1 {
		// Log first 500 chars of response for debugging
		preview := response
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		return nil, errors.Errorf("could not find <analysis> tags in response. Preview: %s", preview)
	}

	xmlContent := response[startIdx : endIdx+len("</analysis>")]

	// Sanitize XML: escape unescaped ampersands that aren't already entities
	xmlContent = sanitizeXML(xmlContent)

	var result AnalysisResult
	if err := xml.Unmarshal([]byte(xmlContent), &result); err != nil {
		return nil, errors.Wrapf(err, "failed to parse XML: %s", xmlContent[:min(200, len(xmlContent))])
	}

	return &result, nil
}

// sanitizeXML escapes special characters that might break XML parsing.
func sanitizeXML(s string) string {
	// Replace unescaped ampersands (not part of entities like &amp; &lt; &gt; &quot; &apos;)
	// This is a simple approach - find & not followed by amp; lt; gt; quot; apos; #
	result := strings.Builder{}
	runes := []rune(s)

	for i := 0; i < len(runes); i++ {
		if runes[i] == '&' {
			// Check if this is already an entity
			remaining := string(runes[i:])
			if strings.HasPrefix(remaining, "&amp;") ||
				strings.HasPrefix(remaining, "&lt;") ||
				strings.HasPrefix(remaining, "&gt;") ||
				strings.HasPrefix(remaining, "&quot;") ||
				strings.HasPrefix(remaining, "&apos;") ||
				(len(remaining) > 2 && remaining[1] == '#') {
				result.WriteRune(runes[i])
			} else {
				result.WriteString("&amp;")
			}
		} else {
			result.WriteRune(runes[i])
		}
	}

	return result.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
