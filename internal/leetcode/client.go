package leetcode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
	session string
	csrf    string
}

type Summary struct {
	FrontendID string
	Slug       string
	Title      string
	Difficulty string
	PaidOnly   bool
}

type Question struct {
	FrontendID    string
	QuestionID    string
	Slug          string
	Title         string
	Difficulty    string
	StatementHTML string
	ExampleTests  string
	Topics        []string
	PythonStub    string
}

type SubmitResult struct {
	SubmissionID int64
	Status       string
	Runtime      string
	Memory       string
}

func New(baseURL, session, csrf string) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://leetcode.com"
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 30 * time.Second},
		session: session,
		csrf:    csrf,
	}
}

func (c *Client) ListSummaries(ctx context.Context) ([]Summary, error) {
	u := c.baseURL + "/api/problems/all/"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	c.addAuth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list summaries failed: %s", string(b))
	}
	var raw struct {
		StatStatusPairs []struct {
			PaidOnly bool `json:"paid_only"`
			Stat     struct {
				QuestionTitleSlug  string `json:"question__title_slug"`
				QuestionTitle      string `json:"question__title"`
				QuestionFrontendID int    `json:"frontend_question_id"`
			} `json:"stat"`
			Difficulty struct {
				Level int `json:"level"`
			} `json:"difficulty"`
		} `json:"stat_status_pairs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]Summary, 0, len(raw.StatStatusPairs))
	for _, p := range raw.StatStatusPairs {
		out = append(out, Summary{
			FrontendID: fmt.Sprintf("%d", p.Stat.QuestionFrontendID),
			Slug:       p.Stat.QuestionTitleSlug,
			Title:      p.Stat.QuestionTitle,
			Difficulty: difficultyLabel(p.Difficulty.Level),
			PaidOnly:   p.PaidOnly,
		})
	}
	return out, nil
}

func (c *Client) PickRandom(ctx context.Context, difficulty string) (Summary, error) {
	all, err := c.ListSummaries(ctx)
	if err != nil {
		return Summary{}, err
	}
	pick := make([]Summary, 0)
	for _, s := range all {
		if s.PaidOnly {
			continue
		}
		if difficulty != "" && !strings.EqualFold(s.Difficulty, difficulty) {
			continue
		}
		pick = append(pick, s)
	}
	if len(pick) == 0 {
		return Summary{}, fmt.Errorf("no problems found with filters")
	}
	return pick[rand.Intn(len(pick))], nil
}

func (c *Client) Question(ctx context.Context, slug string) (Question, error) {
	query := `query questionData($titleSlug: String!) { question(titleSlug: $titleSlug) { questionId questionFrontendId title titleSlug difficulty content exampleTestcases topicTags { name } codeSnippets { langSlug code } } }`
	payload := map[string]any{
		"query":     query,
		"variables": map[string]string{"titleSlug": slug},
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/graphql", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return Question{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return Question{}, fmt.Errorf("question query failed: %s", string(body))
	}

	var raw struct {
		Data struct {
			Question struct {
				QuestionID         string `json:"questionId"`
				QuestionFrontendID string `json:"questionFrontendId"`
				Title              string `json:"title"`
				TitleSlug          string `json:"titleSlug"`
				Difficulty         string `json:"difficulty"`
				Content            string `json:"content"`
				ExampleTestcases   string `json:"exampleTestcases"`
				TopicTags          []struct {
					Name string `json:"name"`
				} `json:"topicTags"`
				CodeSnippets []struct {
					LangSlug string `json:"langSlug"`
					Code     string `json:"code"`
				} `json:"codeSnippets"`
			} `json:"question"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return Question{}, err
	}
	q := raw.Data.Question
	out := Question{
		FrontendID:    q.QuestionFrontendID,
		QuestionID:    q.QuestionID,
		Slug:          q.TitleSlug,
		Title:         q.Title,
		Difficulty:    q.Difficulty,
		StatementHTML: q.Content,
		ExampleTests:  q.ExampleTestcases,
	}
	for _, t := range q.TopicTags {
		out.Topics = append(out.Topics, t.Name)
	}
	for _, cs := range q.CodeSnippets {
		if cs.LangSlug == "python3" {
			out.PythonStub = cs.Code
			break
		}
	}
	return out, nil
}

func (c *Client) ValidateAuth(ctx context.Context) (string, error) {
	query := `query globalData { userStatus { username } }`
	payload := map[string]any{"query": query}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/graphql", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("auth validation failed: %s", string(body))
	}
	var raw struct {
		Data struct {
			UserStatus struct {
				Username string `json:"username"`
			} `json:"userStatus"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return "", err
	}
	if raw.Data.UserStatus.Username == "" {
		return "", fmt.Errorf("cookie auth failed")
	}
	return raw.Data.UserStatus.Username, nil
}

func (c *Client) Submit(ctx context.Context, slug, questionID, code string) (SubmitResult, error) {
	if c.session == "" || c.csrf == "" {
		return SubmitResult{}, fmt.Errorf("missing auth cookies")
	}
	submitURL := c.baseURL + "/problems/" + slug + "/submit/"
	body := map[string]any{
		"lang":        "python3",
		"question_id": questionID,
		"typed_code":  code,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, submitURL, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-csrftoken", c.csrf)
	req.Header.Set("origin", c.baseURL)
	req.Header.Set("referer", c.baseURL+"/problems/"+slug+"/")
	c.addAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return SubmitResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return SubmitResult{}, fmt.Errorf("submit failed: %s", string(raw))
	}
	var sr struct {
		SubmissionID int64 `json:"submission_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return SubmitResult{}, err
	}
	if sr.SubmissionID == 0 {
		return SubmitResult{}, fmt.Errorf("no submission id returned")
	}

	checkURL := c.baseURL + "/submissions/detail/" + fmt.Sprintf("%d", sr.SubmissionID) + "/check/"
	deadline := time.Now().Add(40 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		checkReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, checkURL, nil)
		checkReq.Header.Set("x-csrftoken", c.csrf)
		checkReq.Header.Set("referer", c.baseURL+"/problems/"+slug+"/")
		c.addAuth(checkReq)
		checkResp, err := c.http.Do(checkReq)
		if err != nil {
			continue
		}
		var chk struct {
			State     string `json:"state"`
			StatusMsg string `json:"status_msg"`
			Runtime   string `json:"status_runtime"`
			Memory    string `json:"memory"`
		}
		_ = json.NewDecoder(checkResp.Body).Decode(&chk)
		checkResp.Body.Close()
		if strings.EqualFold(chk.State, "SUCCESS") || chk.StatusMsg != "" && chk.StatusMsg != "Pending" {
			return SubmitResult{SubmissionID: sr.SubmissionID, Status: chk.StatusMsg, Runtime: chk.Runtime, Memory: chk.Memory}, nil
		}
	}
	return SubmitResult{SubmissionID: sr.SubmissionID, Status: "Pending"}, nil
}

func (c *Client) addAuth(req *http.Request) {
	u, _ := url.Parse(c.baseURL)
	req.AddCookie(&http.Cookie{Name: "LEETCODE_SESSION", Value: c.session, Path: "/", Domain: u.Hostname()})
	req.AddCookie(&http.Cookie{Name: "csrftoken", Value: c.csrf, Path: "/", Domain: u.Hostname()})
	if c.csrf != "" {
		req.Header.Set("x-csrftoken", c.csrf)
	}
	req.Header.Set("User-Agent", "LeetCLI/0.1")
	req.URL.Path = path.Clean(req.URL.Path)
}

func difficultyLabel(level int) string {
	switch level {
	case 1:
		return "Easy"
	case 2:
		return "Medium"
	case 3:
		return "Hard"
	default:
		return "Unknown"
	}
}
