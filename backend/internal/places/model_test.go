package places

import "testing"

func TestSearchQueryNormalize(t *testing.T) {
	query := SearchQuery{Text: "  Central Park  ", Locality: " New York "}
	if err := query.Normalize(); err != nil {
		t.Fatal(err)
	}
	if query.Text != "Central Park" || query.Locality != "New York" || query.Limit != 20 {
		t.Fatalf("unexpected normalized query: %#v", query)
	}
}

func TestSearchQueryRejectsUnsafeBounds(t *testing.T) {
	cases := []SearchQuery{
		{Text: "x"},
		{Text: "valid", Limit: 51},
	}
	for _, query := range cases {
		if err := query.Normalize(); err == nil {
			t.Fatalf("expected invalid query: %#v", query)
		}
	}
}
