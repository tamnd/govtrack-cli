package govtrack

// Bill holds flattened bill metadata from GovTrack.us.
type Bill struct {
	ID             int    `json:"id" kit:"id"`
	Congress       int    `json:"congress"`
	Type           string `json:"type"`
	Number         int    `json:"number"`
	Title          string `json:"title"`
	IntroducedDate string `json:"introducedDate,omitempty"`
	Status         string `json:"status,omitempty"`
	URL            string `json:"url"`
}

// Vote holds flattened vote metadata from GovTrack.us.
type Vote struct {
	ID       int    `json:"id" kit:"id"`
	Congress int    `json:"congress"`
	Question string `json:"question"`
	Result   string `json:"result"`
	Created  string `json:"created,omitempty"`
	Chamber  string `json:"chamber,omitempty"`
	URL      string `json:"url"`
}

// Person holds a congress member's metadata from GovTrack.us.
type Person struct {
	ID        int    `json:"id" kit:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Party     string `json:"party,omitempty"`
	State     string `json:"state,omitempty"`
	Role      string `json:"role,omitempty"`
	URL       string `json:"url"`
}
