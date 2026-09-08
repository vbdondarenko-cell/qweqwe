package citycontext

import (
	"testing"
	"time"
)

func TestObservationRejectsStaleMockedAndLowAccuracy(t *testing.T) {
	policy := DefaultPolicy()
	now := time.Date(2026, 9, 8, 6, 0, 0, 0, time.UTC)
	base := Observation{
		LatitudeE6: 50_450_100, LongitudeE6: 30_523_400, AccuracyM: 25,
		PermissionClass: PermissionPrecise, CapturedAt: now.Add(-time.Minute),
	}
	if !base.Valid(now, policy) {
		t.Fatal("fresh precise observation should be valid")
	}
	stale := base
	stale.CapturedAt = now.Add(-3 * time.Minute)
	if stale.Valid(now, policy) {
		t.Fatal("stale precise observation must be rejected")
	}
	mocked := base
	mocked.Mocked = true
	if mocked.Valid(now, policy) {
		t.Fatal("mocked observation must be rejected")
	}
	lowAccuracy := base
	lowAccuracy.AccuracyM = policy.PreciseMaxAccuracyM + 1
	if lowAccuracy.Valid(now, policy) {
		t.Fatal("low accuracy precise observation must be rejected")
	}
}

func TestApproximateObservationUsesSeparateFreshnessAndAccuracy(t *testing.T) {
	policy := DefaultPolicy()
	now := time.Date(2026, 9, 8, 6, 0, 0, 0, time.UTC)
	observation := Observation{
		LatitudeE6: 50_450_100, LongitudeE6: 30_523_400, AccuracyM: 3_000,
		PermissionClass: PermissionApproximate, CapturedAt: now.Add(-5 * time.Minute),
	}
	if !observation.Valid(now, policy) {
		t.Fatal("approximate observation inside approximate policy should be valid")
	}
}

func TestNextLockRequiresStableBoundarySwitch(t *testing.T) {
	policy := DefaultPolicy()
	now := time.Date(2026, 9, 8, 6, 0, 0, 0, time.UTC)
	kyiv := Locality{ID: "00000000-0000-0000-0000-000000000001", Name: "Kyiv", CountryCode: "UA", Timezone: "Europe/Kyiv"}
	other := Locality{ID: "00000000-0000-0000-0000-000000000002", Name: "Other", CountryCode: "UA", Timezone: "Europe/Kyiv"}
	firstObservation := Observation{
		LatitudeE6: 50_450_100, LongitudeE6: 30_523_400, AccuracyM: 20,
		PermissionClass: PermissionPrecise, CapturedAt: now,
	}
	lock, err := NextLock(nil, kyiv, firstObservation, now, policy)
	if err != nil {
		t.Fatal(err)
	}
	lock.UserID = "00000000-0000-0000-0000-000000000099"

	boundaryObservation := firstObservation
	boundaryObservation.CapturedAt = now.Add(time.Minute)
	pending, err := NextLock(&lock, other, boundaryObservation, now.Add(time.Minute), policy)
	if err != nil {
		t.Fatal(err)
	}
	if pending.Locality.ID != kyiv.ID || pending.CandidateLocalityID == nil || *pending.CandidateLocalityID != other.ID || pending.CandidateCount != 1 {
		t.Fatalf("first boundary observation must keep current locality and record candidate: %+v", pending)
	}

	confirmedObservation := boundaryObservation
	confirmedObservation.CapturedAt = now.Add(2 * time.Minute)
	switched, err := NextLock(&pending, other, confirmedObservation, now.Add(2*time.Minute), policy)
	if err != nil {
		t.Fatal(err)
	}
	if switched.Locality.ID != other.ID || switched.CandidateLocalityID != nil || switched.CandidateCount != 0 {
		t.Fatalf("second stable observation should switch locality: %+v", switched)
	}
	if switched.UserID != lock.UserID {
		t.Fatal("switch must preserve user identity")
	}
}

func TestReturningToLockedLocalityClearsBoundaryCandidate(t *testing.T) {
	policy := DefaultPolicy()
	now := time.Date(2026, 9, 8, 6, 0, 0, 0, time.UTC)
	kyiv := Locality{ID: "00000000-0000-0000-0000-000000000001", Name: "Kyiv", CountryCode: "UA", Timezone: "Europe/Kyiv"}
	candidate := "00000000-0000-0000-0000-000000000002"
	candidateAt := now.Add(-time.Minute)
	current := Lock{
		UserID: "00000000-0000-0000-0000-000000000099", Locality: kyiv,
		PermissionClass: PermissionPrecise, AccuracyM: 30, ObservedAt: now.Add(-2 * time.Minute),
		ExpiresAt: now.Add(time.Hour), CandidateLocalityID: &candidate, CandidateCount: 1, CandidateObservedAt: &candidateAt,
	}
	observation := Observation{
		LatitudeE6: 50_450_100, LongitudeE6: 30_523_400, AccuracyM: 25,
		PermissionClass: PermissionPrecise, CapturedAt: now,
	}
	next, err := NextLock(&current, kyiv, observation, now, policy)
	if err != nil {
		t.Fatal(err)
	}
	if next.CandidateLocalityID != nil || next.CandidateCount != 0 || next.CandidateObservedAt != nil {
		t.Fatal("returning to locked locality must clear candidate")
	}
}
