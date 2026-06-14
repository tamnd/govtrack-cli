package govtrack

import (
	"testing"

	"github.com/tamnd/any-cli/kit"
)

// These tests exercise the URI driver's pure string functions
// and the host wiring, which need no network.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "govtrack" {
		t.Errorf("Scheme = %q, want govtrack", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "govtrack" {
		t.Errorf("Identity.Binary = %q, want govtrack", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct{ in, typ, id string }{
		{"123456", "bill", "123456"},
		{"118", "bill", "118"},
	}
	for _, tc := range cases {
		typ, id, err := Domain{}.Classify(tc.in)
		if err != nil || typ != tc.typ || id != tc.id {
			t.Errorf("Classify(%q) = (%q, %q, %v), want (%q, %q, nil)",
				tc.in, typ, id, err, tc.typ, tc.id)
		}
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("bill", "118/s948")
	want := "https://" + Host + "/congress/bills/118/s948"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocatePerson(t *testing.T) {
	got, err := Domain{}.Locate("person", "300001")
	want := "https://" + Host + "/congress/members/300001"
	if err != nil || got != want {
		t.Errorf("Locate(person) = (%q, %v), want (%q, nil)", got, err, want)
	}
}

// TestHostWiring verifies the domain is registered with kit.
func TestHostWiring(t *testing.T) {
	_, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}
	// kit.Open succeeds because init() in domain.go called kit.Register.
}
