package govtrack_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tamnd/govtrack-cli/govtrack"
)

// newTestClient creates a Client pointing at a test server. Rate is 0 for speed.
func newTestClient(t *testing.T, handler http.HandlerFunc) (*govtrack.Client, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	cfg := govtrack.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	return govtrack.NewClient(cfg), srv.Close
}

func serveJSON(t *testing.T, v any) http.HandlerFunc {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}
}

// --- fixtures ---

var billsFixture = map[string]any{
	"meta":    map[string]any{"total_count": 19315, "offset": 0, "limit": 2},
	"objects": []any{
		map[string]any{
			"id":              123456,
			"congress":        118,
			"bill_type":       "s",
			"number":          948,
			"title":           "Healthy Moms and Babies Act",
			"introduced_date": "2023-03-30",
			"current_status":  "introduced",
		},
		map[string]any{
			"id":              123457,
			"congress":        118,
			"bill_type":       "hr",
			"number":          1234,
			"title":           "Infrastructure Investment Act",
			"introduced_date": "2023-04-01",
			"current_status":  "passed_simpleres",
		},
	},
}

var singleBillFixture = map[string]any{
	"id":              123456,
	"congress":        118,
	"bill_type":       "s",
	"number":          948,
	"title":           "Healthy Moms and Babies Act",
	"introduced_date": "2023-03-30",
	"current_status":  "introduced",
}

var votesFixture = map[string]any{
	"meta":    map[string]any{"total_count": 1932},
	"objects": []any{
		map[string]any{
			"id":       1,
			"congress": 118,
			"question": "Call By States",
			"result":   "Passed",
			"created":  "2023-01-03T00:00:00",
			"chamber":  "senate",
		},
		map[string]any{
			"id":       2,
			"congress": 118,
			"question": "On the Motion to Adjourn",
			"result":   "Failed",
			"created":  "2023-01-04T00:00:00",
			"chamber":  "house",
		},
	},
}

var peopleFixture = map[string]any{
	"meta":    map[string]any{"total_count": 100},
	"objects": []any{
		map[string]any{
			"id":        300001,
			"firstname": "Joe",
			"lastname":  "Biden",
			"party":     "Democrat",
			"current_role": map[string]any{
				"role_type_label": "Senator",
				"state":           "DE",
			},
		},
		map[string]any{
			"id":        300002,
			"firstname": "Mitch",
			"lastname":  "McConnell",
			"party":     "Republican",
			"current_role": map[string]any{
				"role_type_label": "Senator",
				"state":           "KY",
			},
		},
	},
}

// TestListBills verifies count, first bill Title, Congress, and Type.
func TestListBills(t *testing.T) {
	c, close := newTestClient(t, serveJSON(t, billsFixture))
	defer close()

	bills, err := c.ListBills(context.Background(), 118, 2)
	if err != nil {
		t.Fatalf("ListBills: %v", err)
	}
	if len(bills) != 2 {
		t.Fatalf("got %d bills, want 2", len(bills))
	}
	b := bills[0]
	if b.Title != "Healthy Moms and Babies Act" {
		t.Errorf("Title = %q, want Healthy Moms and Babies Act", b.Title)
	}
	if b.Congress != 118 {
		t.Errorf("Congress = %d, want 118", b.Congress)
	}
	if b.Type != "s" {
		t.Errorf("Type = %q, want s", b.Type)
	}
	if b.Number != 948 {
		t.Errorf("Number = %d, want 948", b.Number)
	}
	if !strings.Contains(b.URL, "118") {
		t.Errorf("URL = %q, want it to contain 118", b.URL)
	}
}

// TestGetBill verifies a single bill by ID.
func TestGetBill(t *testing.T) {
	c, close := newTestClient(t, serveJSON(t, singleBillFixture))
	defer close()

	bill, err := c.GetBill(context.Background(), 123456)
	if err != nil {
		t.Fatalf("GetBill: %v", err)
	}
	if bill.ID != 123456 {
		t.Errorf("ID = %d, want 123456", bill.ID)
	}
	if bill.Type != "s" {
		t.Errorf("Type = %q, want s", bill.Type)
	}
	if bill.Title != "Healthy Moms and Babies Act" {
		t.Errorf("Title = %q, want Healthy Moms and Babies Act", bill.Title)
	}
	if bill.Status != "introduced" {
		t.Errorf("Status = %q, want introduced", bill.Status)
	}
}

// TestListVotes verifies count, Question, Result, and Chamber.
func TestListVotes(t *testing.T) {
	c, close := newTestClient(t, serveJSON(t, votesFixture))
	defer close()

	votes, err := c.ListVotes(context.Background(), 118, 2)
	if err != nil {
		t.Fatalf("ListVotes: %v", err)
	}
	if len(votes) != 2 {
		t.Fatalf("got %d votes, want 2", len(votes))
	}
	v := votes[0]
	if v.Question != "Call By States" {
		t.Errorf("Question = %q, want Call By States", v.Question)
	}
	if v.Result != "Passed" {
		t.Errorf("Result = %q, want Passed", v.Result)
	}
	if v.Chamber != "senate" {
		t.Errorf("Chamber = %q, want senate", v.Chamber)
	}
	if v.Created != "2023-01-03T00:00:00" {
		t.Errorf("Created = %q, want 2023-01-03T00:00:00", v.Created)
	}
}

// TestListPeople verifies FirstName, LastName, Party, State, Role.
func TestListPeople(t *testing.T) {
	c, close := newTestClient(t, serveJSON(t, peopleFixture))
	defer close()

	people, err := c.ListPeople(context.Background(), "senator", 2)
	if err != nil {
		t.Fatalf("ListPeople: %v", err)
	}
	if len(people) != 2 {
		t.Fatalf("got %d people, want 2", len(people))
	}
	p := people[0]
	if p.FirstName != "Joe" {
		t.Errorf("FirstName = %q, want Joe", p.FirstName)
	}
	if p.LastName != "Biden" {
		t.Errorf("LastName = %q, want Biden", p.LastName)
	}
	if p.Party != "Democrat" {
		t.Errorf("Party = %q, want Democrat", p.Party)
	}
	if p.State != "DE" {
		t.Errorf("State = %q, want DE", p.State)
	}
	if p.Role != "Senator" {
		t.Errorf("Role = %q, want Senator", p.Role)
	}
}

// TestSearchBills verifies title-keyword search returns matching bills.
func TestSearchBills(t *testing.T) {
	c, close := newTestClient(t, serveJSON(t, billsFixture))
	defer close()

	bills, err := c.SearchBills(context.Background(), "infrastructure", 2)
	if err != nil {
		t.Fatalf("SearchBills: %v", err)
	}
	if len(bills) == 0 {
		t.Fatal("expected at least one bill from search")
	}
	// Verify at least one bill has a title (the fixture returns all bills).
	if bills[0].Title == "" {
		t.Error("first bill has empty title")
	}
}

// TestRetryOn503 checks that the client retries on 503 and succeeds on the third try.
func TestRetryOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		b, _ := json.Marshal(billsFixture)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	cfg := govtrack.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := govtrack.NewClient(cfg)

	start := time.Now()
	bills, err := c.ListBills(context.Background(), 118, 2)
	if err != nil {
		t.Fatalf("ListBills after retries: %v", err)
	}
	if len(bills) == 0 {
		t.Error("expected at least one bill after retries")
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off (expected >= 500ms)")
	}
}

// TestCongressFilter verifies that the congress parameter appears in the request URL.
func TestCongressFilter(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		b, _ := json.Marshal(billsFixture)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	cfg := govtrack.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	c := govtrack.NewClient(cfg)

	_, err := c.ListBills(context.Background(), 118, 5)
	if err != nil {
		t.Fatalf("ListBills: %v", err)
	}
	if !strings.Contains(gotQuery, "congress=118") {
		t.Errorf("query = %q, want it to contain congress=118", gotQuery)
	}
}

// TestRoleFilter verifies that the role parameter appears in the people request URL.
func TestRoleFilter(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		b, _ := json.Marshal(peopleFixture)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	cfg := govtrack.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	c := govtrack.NewClient(cfg)

	_, err := c.ListPeople(context.Background(), "senator", 5)
	if err != nil {
		t.Fatalf("ListPeople: %v", err)
	}
	if !strings.Contains(gotQuery, "senator") {
		t.Errorf("query = %q, want it to contain senator", gotQuery)
	}
}

// TestUserAgent verifies that every request carries the govtrack-cli User-Agent.
func TestUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		b, _ := json.Marshal(billsFixture)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	cfg := govtrack.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	c := govtrack.NewClient(cfg)

	_, err := c.ListBills(context.Background(), 118, 2)
	if err != nil {
		t.Fatalf("ListBills: %v", err)
	}
	if !strings.Contains(gotUA, "govtrack-cli") {
		t.Errorf("User-Agent = %q, want it to contain govtrack-cli", gotUA)
	}
}
