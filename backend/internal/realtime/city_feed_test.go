package realtime

import (
	"context"
	"errors"
	"testing"
)

type fakeCityFeedStore struct {
	batch     CityBatch
	err       error
	cursor    int64
	cursorErr error
}

func (f *fakeCityFeedStore) PullCity(context.Context, string, int64, int) (CityBatch, error) {
	return f.batch, f.err
}

func (f *fakeCityFeedStore) CurrentCursor(context.Context) (int64, error) {
	return f.cursor, f.cursorErr
}

func TestCityFeedServiceCurrentCursor(t *testing.T) {
	store := &fakeCityFeedStore{cursor: 99}
	service, err := NewCityFeedService(store)
	if err != nil {
		t.Fatal(err)
	}
	got, err := service.CurrentCursor(context.Background(), " viewer ")
	if err != nil {
		t.Fatal(err)
	}
	if got != 99 {
		t.Fatalf("expected cursor 99, got %d", got)
	}
}

func TestCityFeedServiceCurrentCursorRejectsEmptyViewer(t *testing.T) {
	service, _ := NewCityFeedService(&fakeCityFeedStore{})
	if _, err := service.CurrentCursor(context.Background(), ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCityFeedServiceCurrentCursorPropagatesStoreFailure(t *testing.T) {
	want := errors.New("db down")
	service, _ := NewCityFeedService(&fakeCityFeedStore{cursorErr: want})
	if _, err := service.CurrentCursor(context.Background(), "viewer"); !errors.Is(err, want) {
		t.Fatalf("err=%v", err)
	}
}
