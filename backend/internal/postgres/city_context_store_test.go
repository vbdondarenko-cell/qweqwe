package postgres

import "testing"

func TestRoundToGridNeverStoresRawValue(t *testing.T) {
	cases := []struct {
		value, grid, want int
	}{
		{0, 1000, 0},
		{499, 1000, 0},
		{500, 1000, 1000},
		{501, 1000, 1000},
		{1499, 1000, 1000},
		{1500, 1000, 2000},
		{-499, 1000, 0},
		{-500, 1000, -1000},
		{-501, 1000, -1000},
		{-1500, 1000, -2000},
	}
	for _, tc := range cases {
		if got := roundToGrid(tc.value, tc.grid); got != tc.want {
			t.Fatalf("roundToGrid(%d,%d)=%d, want %d", tc.value, tc.grid, got, tc.want)
		}
	}
}

func TestRoundToPointGridBoundsErrorToHalfAGridCell(t *testing.T) {
	lat, lng := roundToPointGrid(49412345, 32087654)
	if abs(lat-49412345) > pointGridE6/2 {
		t.Fatalf("rounded latitude %d too far from raw 49412345", lat)
	}
	if abs(lng-32087654) > pointGridE6/2 {
		t.Fatalf("rounded longitude %d too far from raw 32087654", lng)
	}
	if lat%pointGridE6 != 0 || lng%pointGridE6 != 0 {
		t.Fatalf("expected both coordinates aligned to the grid, got lat=%d lng=%d", lat, lng)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
