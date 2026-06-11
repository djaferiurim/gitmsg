// Package ai provides an optional, OpenAI-compatible message generator.
//
// It is only used when an API key is available. gitmsg works fully offline
// without it; AI mode simply produces richer summaries.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultURL   = "https://api.openai.com/v1/chat/completions"
	defaultModel = "gpt-4o-mini"
	maxDiffChars = 12000
)

// Available reports whether an API key is configured.
func Available() bool {
	return apiKey() != ""
}

func apiKey() string {
	if k := os.Getenv("GITMSG_API_KEY"); k != "" {
		return k
	}
	return os.Getenv("OPENAI_API_KEY")
}

func endpoint() string {
	if u := os.Getenv("GITMSG_API_URL"); u != "" {
		return u
	}
	return defaultURL
}

func model() string {
	if m := os.Getenv("GITMSG_MODEL"); m != "" {
		return m
	}
	return defaultModel
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// CommitMessage asks the model to write a Conventional Commit message for the
// given staged diff. The returned string is the full commit message.
func CommitMessage(ctx context.Context, diff string) (string, error) {
	system := "You are an expert engineer who writes concise Conventional Commit messages. " +
		"Reply with only the commit message: a single header line of the form " +
		"`type(scope): summary` (lowercase, imperative, <= 72 chars), optionally " +
		"followed by a blank line and a short bullet-point body. Valid types: " +
		"feat, fix, docs, style, refactor, perf, test, build, ci, chore."
	user := "Write a commit message for this staged diff:\n\n" + truncate(diff)
	return complete(ctx, system, user)
}

// PullRequest asks the model to write a PR title and markdown body.
func PullRequest(ctx context.Context, commits, diff string) (string, error) {
	system := "You write clear pull request descriptions in GitHub-flavored markdown. " +
		"Return a single line `Title: <pr title>` followed by a blank line and a " +
		"markdown body with ## Summary and ## Changes sections."
	user := fmt.Sprintf("Commits:\n%s\n\nDiff (may be truncated):\n%s", commits, truncate(diff))
	return complete(ctx, system, user)
}

func complete(ctx context.Context, system, user string) (string, error) {
	key := apiKey()
	if key == "" {
		return "", fmt.Errorf("no API key configured")
	}

	payload, err := json.Marshal(chatRequest{
		Model:       model(),
		Temperature: 0.2,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	})
	if err != nil {
		return "", err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint(), bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}

	var parsed chatResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", fmt.Errorf("unexpected response (%d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("api error: %s", parsed.Error.Message)
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("api returned status %d", resp.StatusCode)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("api returned no choices")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

func truncate(s string) string {
	if len(s) <= maxDiffChars {
		return s
	}
	return s[:maxDiffChars] + "\n... (diff truncated)"
}
