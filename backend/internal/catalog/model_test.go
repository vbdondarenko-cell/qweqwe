package catalog

import (
	"errors"
	"strings"
	"testing"
)

func boolPtr(v bool) *bool { return &v }

func TestDecodeNormalizesTrustedCatalog(t *testing.T) {
	raw := `{"localities":[{"source":" OSM ","sourceLocalityId":"relation/1","name":" Cherkasy ","countryCode":"ua","timezone":"Europe/Kyiv","centroidLatitudeE6":49444000,"centroidLongitudeE6":32059000,"boundaryWkt":"MULTIPOLYGON(((32 49,33 49,33 50,32 49)))","active":true}],"places":[{"source":"google","sourcePlaceId":"abc","name":"Park","category":"park","localitySource":"OSM","localitySourceId":"relation/1","countryCode":"ua","latitudeE6":49440000,"longitudeE6":32060000,"precisionM":50,"active":true}]}`
	doc, err := Decode(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Localities[0].Source != "osm" || doc.Localities[0].CountryCode != "UA" || doc.Places[0].LocalitySource != "osm" {
		t.Fatalf("normalization failed: %#v", doc)
	}
}

func TestCatalogRejectsUnknownDuplicateAndPartialLocalityLink(t *testing.T) {
	if _, err := Decode(strings.NewReader(`{"localities":[],"places":[],"unknown":1}`)); !errors.Is(err, ErrInvalidCatalog) {
		t.Fatalf("unknown field: %v", err)
	}
	active := true
	doc := Document{Localities: []LocalityRecord{
		{Source: "osm", SourceLocalityID: "1", Name: "A", CountryCode: "UA", Timezone: "Europe/Kyiv", BoundaryWKT: "POLYGON((0 0,1 0,1 1,0 0))", Active: &active},
		{Source: "osm", SourceLocalityID: "1", Name: "B", CountryCode: "UA", Timezone: "Europe/Kyiv", BoundaryWKT: "POLYGON((0 0,1 0,1 1,0 0))", Active: &active},
	}}
	if err := doc.NormalizeAndValidate(); !errors.Is(err, ErrInvalidCatalog) {
		t.Fatalf("duplicate locality: %v", err)
	}
	place := PlaceRecord{Source: "google", SourcePlaceID: "p", Name: "P", CountryCode: "UA", LocalitySource: "osm", PrecisionM: 10, Active: boolPtr(true)}
	doc = Document{Places: []PlaceRecord{place}}
	if err := doc.NormalizeAndValidate(); !errors.Is(err, ErrInvalidCatalog) {
		t.Fatalf("partial locality link: %v", err)
	}
}
