// Package govtrack is the library behind the govtrack command line:
// the HTTP client, request shaping, and typed data models for the GovTrack.us API v2
// (https://www.govtrack.us/api/v2).
//
// The API requires no key. The Client paces requests, sets a real User-Agent,
// and retries transient failures (429 and 5xx).
package govtrack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Host is the API hostname.
const Host = "www.govtrack.us"

// BaseURL is the root every API request is built from.
const BaseURL = "https://" + Host + "/api/v2"

// Config holds all tunable parameters for the Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns a Config with sensible defaults for GovTrack.us.
func DefaultConfig() Config {
	return Config{
		BaseURL:   BaseURL,
		UserAgent: "govtrack-cli/0.1.0 (github.com/tamnd/govtrack-cli)",
		Rate:      200 * time.Millisecond,
		Timeout:   30 * time.Second,
		Retries:   3,
	}
}

// Client talks to GovTrack.us API over HTTP.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client configured with cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// get fetches a path relative to cfg.BaseURL and returns the response body.
func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	rawURL := c.cfg.BaseURL + path
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("govtrack: all %d attempts failed: %w", c.cfg.Retries+1, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("govtrack: HTTP %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("govtrack: HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// --- wire types (API response nesting, unexported) ---

type wireBill struct {
	ID             int    `json:"id"`
	Congress       int    `json:"congress"`
	BillType       string `json:"bill_type"`
	Number         int    `json:"number"`
	Title          string `json:"title"`
	IntroducedDate string `json:"introduced_date"`
	CurrentStatus  string `json:"current_status"`
}

type wireVote struct {
	ID       int    `json:"id"`
	Congress int    `json:"congress"`
	Question string `json:"question"`
	Result   string `json:"result"`
	Created  string `json:"created"`
	Chamber  string `json:"chamber"`
}

type wirePerson struct {
	ID          int    `json:"id"`
	Firstname   string `json:"firstname"`
	Lastname    string `json:"lastname"`
	Party       string `json:"party"`
	CurrentRole *struct {
		RoleTypeLabel string `json:"role_type_label"`
		State         string `json:"state"`
	} `json:"current_role"`
}

type wireList[T any] struct {
	Meta struct {
		TotalCount int `json:"total_count"`
	} `json:"meta"`
	Objects []T `json:"objects"`
}

// --- API methods ---

// ListBills returns bills for a congress, limited to limit results.
func (c *Client) ListBills(ctx context.Context, congress, limit int) ([]Bill, error) {
	path := fmt.Sprintf("/bill?congress=%d&limit=%d&format=json", congress, limit)
	body, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	var resp wireList[wireBill]
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("govtrack: parse bills: %w", err)
	}
	return convertBills(resp.Objects), nil
}

// GetBill returns a single bill by numeric ID.
func (c *Client) GetBill(ctx context.Context, id int) (*Bill, error) {
	path := fmt.Sprintf("/bill/%d?format=json", id)
	body, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	var w wireBill
	if err := json.Unmarshal(body, &w); err != nil {
		return nil, fmt.Errorf("govtrack: parse bill: %w", err)
	}
	b := convertBill(w)
	return &b, nil
}

// ListVotes returns votes for a congress, limited to limit results.
func (c *Client) ListVotes(ctx context.Context, congress, limit int) ([]Vote, error) {
	path := fmt.Sprintf("/vote?congress=%d&limit=%d&format=json", congress, limit)
	body, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	var resp wireList[wireVote]
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("govtrack: parse votes: %w", err)
	}
	return convertVotes(resp.Objects), nil
}

// ListPeople returns congress members filtered by role type.
// role: "senator", "representative", or "" for all.
func (c *Client) ListPeople(ctx context.Context, role string, limit int) ([]Person, error) {
	q := fmt.Sprintf("/person?limit=%d&format=json", limit)
	if role != "" {
		q += "&roles__role_type=" + url.QueryEscape(role)
	}
	body, err := c.get(ctx, q)
	if err != nil {
		return nil, err
	}
	var resp wireList[wirePerson]
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("govtrack: parse people: %w", err)
	}
	return convertPeople(resp.Objects), nil
}

// SearchBills searches bills by title keyword.
func (c *Client) SearchBills(ctx context.Context, query string, limit int) ([]Bill, error) {
	q := fmt.Sprintf("/bill?title__contains=%s&limit=%d&format=json", url.QueryEscape(query), limit)
	body, err := c.get(ctx, q)
	if err != nil {
		return nil, err
	}
	var resp wireList[wireBill]
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("govtrack: parse bills: %w", err)
	}
	return convertBills(resp.Objects), nil
}

// --- converters ---

func convertBill(w wireBill) Bill {
	return Bill{
		ID:             w.ID,
		Congress:       w.Congress,
		Type:           w.BillType,
		Number:         w.Number,
		Title:          w.Title,
		IntroducedDate: w.IntroducedDate,
		Status:         w.CurrentStatus,
		URL:            fmt.Sprintf("https://%s/congress/bills/%d/%s%d", Host, w.Congress, w.BillType, w.Number),
	}
}

func convertBills(ws []wireBill) []Bill {
	out := make([]Bill, len(ws))
	for i, w := range ws {
		out[i] = convertBill(w)
	}
	return out
}

func convertVotes(ws []wireVote) []Vote {
	out := make([]Vote, len(ws))
	for i, w := range ws {
		out[i] = Vote{
			ID:       w.ID,
			Congress: w.Congress,
			Question: w.Question,
			Result:   w.Result,
			Created:  w.Created,
			Chamber:  w.Chamber,
			URL:      fmt.Sprintf("https://%s/congress/votes/%d/%s-%d", Host, w.Congress, w.Chamber, w.ID),
		}
	}
	return out
}

func convertPeople(ws []wirePerson) []Person {
	out := make([]Person, len(ws))
	for i, w := range ws {
		role := ""
		state := ""
		if w.CurrentRole != nil {
			role = w.CurrentRole.RoleTypeLabel
			state = w.CurrentRole.State
		}
		out[i] = Person{
			ID:        w.ID,
			FirstName: w.Firstname,
			LastName:  w.Lastname,
			Party:     w.Party,
			State:     state,
			Role:      role,
			URL:       fmt.Sprintf("https://%s/congress/members/%d", Host, w.ID),
		}
	}
	return out
}
