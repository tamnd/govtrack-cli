package govtrack

import (
	"context"
	"fmt"
	"strconv"

	"github.com/tamnd/any-cli/kit"
)

func init() { kit.Register(Domain{}) }

// Domain is the govtrack driver for the kit framework.
type Domain struct{}

// Info describes the scheme, hostnames, and binary identity.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "govtrack",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "govtrack",
			Short:  "A command line for GovTrack.us congressional data.",
			Long: `A command line for GovTrack.us — browse US congressional bills, votes, and members.

govtrack reads public data over HTTPS, shapes it into clean records,
and prints output that pipes into the rest of your tools. No API key required.`,
			Site: Host,
			Repo: "https://github.com/tamnd/govtrack-cli",
		},
	}
}

// Register installs the client factory and all operations onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{Name: "bills", Group: "read", List: true,
		Summary: "List bills (--congress, --limit)"}, billsOp)

	kit.Handle(app, kit.OpMeta{Name: "bill", Group: "read", Single: true,
		Summary: "Get a bill by numeric ID",
		Args:    []kit.Arg{{Name: "id", Help: "bill ID"}}}, billOp)

	kit.Handle(app, kit.OpMeta{Name: "votes", Group: "read", List: true,
		Summary: "List votes (--congress, --limit)"}, votesOp)

	kit.Handle(app, kit.OpMeta{Name: "people", Group: "read", List: true,
		Summary: "List congress members (--role senator|representative, --limit)"}, peopleOp)

	kit.Handle(app, kit.OpMeta{Name: "search", Group: "read", List: true,
		Summary: "Search bills by title keyword",
		Args:    []kit.Arg{{Name: "query", Help: "title keyword"}}}, searchOp)
}

// newClient builds the Client from the kit Config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- input structs ---

type billsInput struct {
	Congress int     `kit:"flag" help:"congress number (default 118)"`
	Limit    int     `kit:"flag,inherit" help:"max results"`
	Client   *Client `kit:"inject"`
}

type billInput struct {
	ID     string  `kit:"arg" help:"bill numeric ID"`
	Client *Client `kit:"inject"`
}

type votesInput struct {
	Congress int     `kit:"flag" help:"congress number (default 118)"`
	Limit    int     `kit:"flag,inherit" help:"max results"`
	Client   *Client `kit:"inject"`
}

type peopleInput struct {
	Role   string  `kit:"flag" help:"role: senator or representative"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type searchInput struct {
	Query  string  `kit:"arg" help:"title keyword to search"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func billsOp(ctx context.Context, in billsInput, emit func(*Bill) error) error {
	congress := in.Congress
	if congress == 0 {
		congress = 118
	}
	limit := in.Limit
	if limit == 0 {
		limit = 20
	}
	bills, err := in.Client.ListBills(ctx, congress, limit)
	if err != nil {
		return err
	}
	for i := range bills {
		if err := emit(&bills[i]); err != nil {
			return err
		}
	}
	return nil
}

func billOp(ctx context.Context, in billInput, emit func(*Bill) error) error {
	id, err := strconv.Atoi(in.ID)
	if err != nil {
		return fmt.Errorf("govtrack: invalid bill ID %q: %w", in.ID, err)
	}
	bill, err := in.Client.GetBill(ctx, id)
	if err != nil {
		return err
	}
	return emit(bill)
}

func votesOp(ctx context.Context, in votesInput, emit func(*Vote) error) error {
	congress := in.Congress
	if congress == 0 {
		congress = 118
	}
	limit := in.Limit
	if limit == 0 {
		limit = 20
	}
	votes, err := in.Client.ListVotes(ctx, congress, limit)
	if err != nil {
		return err
	}
	for i := range votes {
		if err := emit(&votes[i]); err != nil {
			return err
		}
	}
	return nil
}

func peopleOp(ctx context.Context, in peopleInput, emit func(*Person) error) error {
	limit := in.Limit
	if limit == 0 {
		limit = 20
	}
	people, err := in.Client.ListPeople(ctx, in.Role, limit)
	if err != nil {
		return err
	}
	for i := range people {
		if err := emit(&people[i]); err != nil {
			return err
		}
	}
	return nil
}

func searchOp(ctx context.Context, in searchInput, emit func(*Bill) error) error {
	limit := in.Limit
	if limit == 0 {
		limit = 20
	}
	bills, err := in.Client.SearchBills(ctx, in.Query, limit)
	if err != nil {
		return err
	}
	for i := range bills {
		if err := emit(&bills[i]); err != nil {
			return err
		}
	}
	return nil
}

// Classify turns any accepted input into (type, id).
func (Domain) Classify(input string) (string, string, error) {
	return "bill", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(t, id string) (string, error) {
	switch t {
	case "bill":
		return fmt.Sprintf("https://%s/congress/bills/%s", Host, id), nil
	case "vote":
		return fmt.Sprintf("https://%s/congress/votes/%s", Host, id), nil
	case "person":
		return fmt.Sprintf("https://%s/congress/members/%s", Host, id), nil
	default:
		return fmt.Sprintf("https://%s/congress/%s/%s", Host, t, id), nil
	}
}
