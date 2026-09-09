package catalog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const MaxImportBytes = 16 << 20

var ErrInvalidCatalog = errors.New("invalid canonical catalog")

type LocalityRecord struct {
	Source              string `json:"source"`
	SourceLocalityID    string `json:"sourceLocalityId"`
	Name                string `json:"name"`
	CountryCode         string `json:"countryCode"`
	Timezone            string `json:"timezone"`
	CentroidLatitudeE6  int    `json:"centroidLatitudeE6"`
	CentroidLongitudeE6 int    `json:"centroidLongitudeE6"`
	BoundaryWKT         string `json:"boundaryWkt"`
	Active              *bool  `json:"active"`
}

type PlaceRecord struct {
	Source           string `json:"source"`
	SourcePlaceID    string `json:"sourcePlaceId"`
	Name             string `json:"name"`
	Category         string `json:"category"`
	LocalitySource   string `json:"localitySource"`
	LocalitySourceID string `json:"localitySourceId"`
	CountryCode      string `json:"countryCode"`
	LatitudeE6       int    `json:"latitudeE6"`
	LongitudeE6      int    `json:"longitudeE6"`
	PrecisionM       int    `json:"precisionM"`
	Active           *bool  `json:"active"`
}

type Document struct {
	Localities []LocalityRecord `json:"localities"`
	Places     []PlaceRecord    `json:"places"`
}

func Decode(r io.Reader) (Document, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxImportBytes+1))
	if err != nil || len(raw) > MaxImportBytes {
		return Document{}, ErrInvalidCatalog
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var doc Document
	if err := dec.Decode(&doc); err != nil {
		return Document{}, fmt.Errorf("%w: %v", ErrInvalidCatalog, err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return Document{}, ErrInvalidCatalog
	}
	if err := doc.NormalizeAndValidate(); err != nil {
		return Document{}, err
	}
	return doc, nil
}

func (d *Document) NormalizeAndValidate() error {
	if d == nil || (len(d.Localities) == 0 && len(d.Places) == 0) || len(d.Localities) > 10_000 || len(d.Places) > 100_000 {
		return ErrInvalidCatalog
	}
	localityKeys := make(map[string]struct{}, len(d.Localities))
	for i := range d.Localities {
		if err := d.Localities[i].normalizeAndValidate(); err != nil {
			return err
		}
		key := d.Localities[i].Source + "\x00" + d.Localities[i].SourceLocalityID
		if _, duplicate := localityKeys[key]; duplicate {
			return ErrInvalidCatalog
		}
		localityKeys[key] = struct{}{}
	}
	placeKeys := make(map[string]struct{}, len(d.Places))
	for i := range d.Places {
		if err := d.Places[i].normalizeAndValidate(); err != nil {
			return err
		}
		key := d.Places[i].Source + "\x00" + d.Places[i].SourcePlaceID
		if _, duplicate := placeKeys[key]; duplicate {
			return ErrInvalidCatalog
		}
		placeKeys[key] = struct{}{}
	}
	return nil
}

func (r *LocalityRecord) normalizeAndValidate() error {
	r.Source = strings.ToLower(strings.TrimSpace(r.Source))
	r.SourceLocalityID = strings.TrimSpace(r.SourceLocalityID)
	r.Name = strings.TrimSpace(r.Name)
	r.CountryCode = strings.ToUpper(strings.TrimSpace(r.CountryCode))
	r.Timezone = strings.TrimSpace(r.Timezone)
	r.BoundaryWKT = strings.TrimSpace(r.BoundaryWKT)
	upperWKT := strings.ToUpper(r.BoundaryWKT)
	if !validSource(r.Source) || !validLen(r.SourceLocalityID, 1, 160) || !validLen(r.Name, 1, 160) ||
		!validCountryCode(r.CountryCode) || !validLen(r.Timezone, 1, 80) || !validTimezone(r.Timezone) || r.Active == nil ||
		r.CentroidLatitudeE6 < -90_000_000 || r.CentroidLatitudeE6 > 90_000_000 ||
		r.CentroidLongitudeE6 < -180_000_000 || r.CentroidLongitudeE6 > 180_000_000 ||
		len(r.BoundaryWKT) < 10 || len(r.BoundaryWKT) > 4_000_000 ||
		(!strings.HasPrefix(upperWKT, "POLYGON") && !strings.HasPrefix(upperWKT, "MULTIPOLYGON")) {
		return ErrInvalidCatalog
	}
	return nil
}

func (r *PlaceRecord) normalizeAndValidate() error {
	r.Source = strings.ToLower(strings.TrimSpace(r.Source))
	r.SourcePlaceID = strings.TrimSpace(r.SourcePlaceID)
	r.Name = strings.TrimSpace(r.Name)
	r.Category = strings.TrimSpace(r.Category)
	r.LocalitySource = strings.ToLower(strings.TrimSpace(r.LocalitySource))
	r.LocalitySourceID = strings.TrimSpace(r.LocalitySourceID)
	r.CountryCode = strings.ToUpper(strings.TrimSpace(r.CountryCode))
	linked := r.LocalitySource != "" || r.LocalitySourceID != ""
	if !validSource(r.Source) || !validLen(r.SourcePlaceID, 1, 160) || !validLen(r.Name, 1, 160) ||
		len(r.Category) > 64 || !validCountryCode(r.CountryCode) || r.Active == nil || r.PrecisionM < 1 || r.PrecisionM > 10_000 ||
		r.LatitudeE6 < -90_000_000 || r.LatitudeE6 > 90_000_000 || r.LongitudeE6 < -180_000_000 || r.LongitudeE6 > 180_000_000 ||
		(linked && (!validSource(r.LocalitySource) || !validLen(r.LocalitySourceID, 1, 160))) {
		return ErrInvalidCatalog
	}
	return nil
}

func validLen(value string, min, max int) bool {
	n := len([]rune(value))
	return n >= min && n <= max
}

func validSource(value string) bool {
	if !validLen(value, 1, 32) {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func validCountryCode(value string) bool {
	return len(value) == 2 && value[0] >= 'A' && value[0] <= 'Z' && value[1] >= 'A' && value[1] <= 'Z'
}

func validTimezone(value string) bool {
	_, err := time.LoadLocation(value)
	return err == nil
}
