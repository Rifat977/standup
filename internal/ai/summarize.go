package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	openai "github.com/sashabaranov/go-openai"

	"github.com/rifat977/standup/internal/config"
	gitscan "github.com/rifat977/standup/internal/git"
	ghclient "github.com/rifat977/standup/internal/github"
	"github.com/rifat977/standup/internal/logx"
)

// Data is the input bundle handed to the AI.
type Data struct {
	Commits []gitscan.Commit
	PRs     []ghclient.PR
	Today   string
	Blocker string
}

const systemPrompt = `You are a helpful assistant that writes concise daily standup notes
for software engineers. Be brief, professional, and clear.
Format the output in exactly three sections: Yesterday, Today, Blockers.
Keep each section to 2-4 sentences maximum.
Do not include commit hashes or PR numbers unless directly relevant.
Write in first person, past tense for Yesterday, present/future for Today.`

// BuildUserPrompt produces the structured user message.
func BuildUserPrompt(d Data) string {
	var b strings.Builder
	fmt.Fprintln(&b, "Here is my activity from the recent window:")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "COMMITS:")
	if len(d.Commits) == 0 {
		fmt.Fprintln(&b, "- (none)")
	} else {
		for _, c := range d.Commits {
			fmt.Fprintf(&b, "- [%s] %s (%s)\n", c.Repo, c.Subject, c.Hash)
		}
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "PULL REQUESTS:")
	if len(d.PRs) == 0 {
		fmt.Fprintln(&b, "- (none)")
	} else {
		for _, p := range d.PRs {
			fmt.Fprintf(&b, "- #%d %s — %s — %s — CI %s — %s\n",
				p.Number, p.Title, strings.ToUpper(p.State),
				strings.ToUpper(p.Review), p.CI, p.AgeString())
		}
	}
	fmt.Fprintln(&b)
	if strings.TrimSpace(d.Today) != "" {
		fmt.Fprintf(&b, "TODAY (my notes): %s\n", d.Today)
	}
	if strings.TrimSpace(d.Blocker) != "" {
		fmt.Fprintf(&b, "BLOCKERS: %s\n", d.Blocker)
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Write a natural standup summary.")
	return b.String()
}

func buildMessages(d Data) []openai.ChatCompletionMessage {
	return []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
		{Role: openai.ChatMessageRoleUser, Content: BuildUserPrompt(d)},
	}
}

// Summarize calls OpenAI synchronously and returns the full response.
func Summarize(ctx context.Context, cfg *config.Config, d Data) (string, error) {
	if cfg.OpenAI.APIKey == "" {
		logx.Error("ai: openai api key not set")
		return "", errors.New("openai api key not set (config or OPENAI_API_KEY)")
	}
	logx.Info("ai: summarize model=%s commits=%d prs=%d", cfg.OpenAI.Model, len(d.Commits), len(d.PRs))
	client := openai.NewClient(cfg.OpenAI.APIKey)
	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     cfg.OpenAI.Model,
		MaxTokens: cfg.OpenAI.MaxTokens,
		Messages:  buildMessages(d),
	})
	if err != nil {
		logx.Error("ai: chat completion failed: %v", err)
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("empty response")
	}
	return resp.Choices[0].Message.Content, nil
}

// TokenMsg is emitted for each streamed token.
type TokenMsg struct{ Token string }

// DoneMsg signals the stream finished.
type DoneMsg struct{ Err error }

// Stream calls the OpenAI streaming endpoint and pushes tokens onto out.
// The channel is closed when the stream finishes or errors. The final value
// sent before close is a DoneMsg.
func Stream(ctx context.Context, cfg *config.Config, d Data, out chan<- any) {
	defer close(out)
	if cfg.OpenAI.APIKey == "" {
		logx.Error("ai: stream — openai api key not set")
		out <- DoneMsg{Err: errors.New("openai api key not set")}
		return
	}
	logx.Info("ai: stream begin model=%s commits=%d prs=%d", cfg.OpenAI.Model, len(d.Commits), len(d.PRs))
	client := openai.NewClient(cfg.OpenAI.APIKey)
	stream, err := client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:     cfg.OpenAI.Model,
		MaxTokens: cfg.OpenAI.MaxTokens,
		Messages:  buildMessages(d),
		Stream:    true,
	})
	if err != nil {
		logx.Error("ai: stream open failed: %v", err)
		out <- DoneMsg{Err: err}
		return
	}
	defer stream.Close()
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			logx.Info("ai: stream complete")
			out <- DoneMsg{}
			return
		}
		if err != nil {
			logx.Error("ai: stream recv failed: %v", err)
			out <- DoneMsg{Err: err}
			return
		}
		if len(resp.Choices) == 0 {
			continue
		}
		token := resp.Choices[0].Delta.Content
		if token != "" {
			out <- TokenMsg{Token: token}
		}
	}
}
