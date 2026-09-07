package places

import (
	"context"
	"errors"
	"strings"
)

const MaxSearchResults = 50

var ErrInvalidSearch = errors.New("invalid place search")

type Place struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Category    *string `json:"category,omitempty"`
	Locality    *string `json:"locality,omitempty"`
	CountryCode *string `json:"countryCode,omitempty"`
	LatitudeE6  int     `json:"latitudeE6"`
	LongitudeE6 int     `json:"longitudeE6"`
	PrecisionM  int     `json:"precisionM"`
}

type SearchQuery struct {
	Text     string
	Locality string
	Limit    int
}

func (q *SearchQuery) Normalize() error {
	q.Text = strings.TrimSpace(q.Text)
	q.Locality = strings.TrimSpace(q.Locality)
	if len([]rune(q.Text)) < 2 || len([]rune(q.Text)) > 80 || len([]rune(q.Locality)) > 120 {
		return ErrInvalidSearch
	}
	if q.Limit == 0 {
		q.Limit = 20
	}
	if q.Limit < 1 || q.Limit > MaxSearchResults {
		return ErrInvalidSearch
	}
	return nil
}

type Store interface {
	Search(ctx context.Context, query SearchQuery) ([]Place, error)
}

type Service struct{ store Store }

func NewService(store Store) (*Service, error) {
	if store == nil {
		return nil, ErrInvalidSearch
	}
	return &Service{store: store}, nil
}

func (s *Service) Search(ctx context.Context, query SearchQuery) ([]Place, error) {
	if err := query.Normalize(); err != nil {
		return nil, err
	}
	return s.store.Search(ctx, query)
}
