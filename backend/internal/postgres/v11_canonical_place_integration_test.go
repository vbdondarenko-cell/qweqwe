package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citymap"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/places"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

func TestV11CanonicalPlaceSlotAndMapIntegration(t *testing.T) {
	dsn := os.Getenv("LINKUP_TEST_DATABASE_URL")
	if dsn == "" || os.Getenv("LINKUP_TEST_DATABASE_DESTRUCTIVE") != "1" {
		t.Skip("disposable PostgreSQL requires LINKUP_TEST_DATABASE_URL and LINKUP_TEST_DATABASE_DESTRUCTIVE=1")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	if err := migrate.Apply(ctx, pool, migrationDir(t)); err != nil {
		t.Fatalf("apply canonical migrations: %v", err)
	}
	assertMigrationCount(t, ctx, pool, 15)

	accountService, err := account.NewService(NewAccountStore(pool), password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	viewer := registerIntegrationUser(t, ctx, accountService, "map", suffix)
	placeID, err := identifier.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	localityID, err := identifier.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	otherPlaceID, err := identifier.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	otherLocalityID, err := identifier.NewUUID()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM slots WHERE host_id=$1::uuid`, viewer.User.ID); err != nil {
			t.Errorf("cleanup slots: %v", err)
		}
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM app_users WHERE id=$1::uuid`, viewer.User.ID); err != nil {
			t.Errorf("cleanup user: %v", err)
		}
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM canonical_places WHERE id IN ($1::uuid,$2::uuid)`, placeID, otherPlaceID); err != nil {
			t.Errorf("cleanup places: %v", err)
		}
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM localities WHERE id IN ($1::uuid,$2::uuid)`, localityID, otherLocalityID); err != nil {
			t.Errorf("cleanup localities: %v", err)
		}
	})

	_, err = pool.Exec(ctx, `
		INSERT INTO localities (
			id,source,source_locality_id,name,country_code,timezone_name,
			centroid_latitude_e6,centroid_longitude_e6,boundary_wkt
		) VALUES ($1::uuid,'integration',$2,'Kyiv','UA','Europe/Kyiv',
			50500000,30500000,'MULTIPOLYGON(((30 50,31 50,31 51,30 51,30 50)))')`,
		localityID, "locality-"+suffix,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO localities (
			id,source,source_locality_id,name,country_code,timezone_name,
			centroid_latitude_e6,centroid_longitude_e6,boundary_wkt
		) VALUES ($1::uuid,'integration',$2,'Other City','UA','Europe/Kyiv',
			50510000,30510000,'MULTIPOLYGON(((30 50,31 50,31 51,30 51,30 50)))')`,
		otherLocalityID, "other-locality-"+suffix,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO canonical_places (
			id,source,source_place_id,name,category,locality,country_code,
			latitude_e6,longitude_e6,precision_m,locality_id
		) VALUES ($1::uuid,'integration',$2,'Integration Place','cafe','Kyiv','UA',50500000,30500000,50,$3::uuid)`,
		placeID, "integration-"+suffix, localityID,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO canonical_places (
			id,source,source_place_id,name,category,locality,country_code,
			latitude_e6,longitude_e6,precision_m,locality_id
		) VALUES ($1::uuid,'integration',$2,'Other Locality Place','cafe','Other City','UA',50502000,30502000,50,$3::uuid)`,
		otherPlaceID, "other-place-"+suffix, otherLocalityID,
	)
	if err != nil {
		t.Fatal(err)
	}

	placeService, err := places.NewService(NewPlaceStore(pool))
	if err != nil {
		t.Fatal(err)
	}
	placeItems, err := placeService.Search(ctx, places.SearchQuery{
		Text: "Integration", LocalityID: localityID, Limit: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(placeItems) != 1 || placeItems[0].ID != placeID || placeItems[0].LocalityID == nil || *placeItems[0].LocalityID != localityID {
		t.Fatalf("canonical locality-scoped place search mismatch: %#v", placeItems)
	}

	baseStore, err := NewSlotStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	v11Store, err := NewV11SlotStore(baseStore)
	if err != nil {
		t.Fatal(err)
	}
	slotService, err := slot.NewService(v11Store)
	if err != nil {
		t.Fatal(err)
	}

	waitlistMode := slot.AccessWaitlist
	draft, err := slotService.CreateDraft(ctx, viewer.User.ID, slot.CreateInput{
		Title: "Waitlist draft", Activity: "coffee", PlaceText: "Integration Place", Capacity: 3, AccessMode: &waitlistMode,
	}, "v11-hosting-draft-access-mode-0001")
	if err != nil {
		t.Fatal(err)
	}
	if draft.State != slot.StateDraft || draft.AccessMode != slot.AccessWaitlist {
		t.Fatalf("draft access mode was not persisted: %#v", draft)
	}
	instantMode := slot.AccessInstant
	editedDraft, err := slotService.Edit(ctx, viewer.User.ID, draft.ID, slot.EditInput{
		ExpectedVersion: draft.Version, AccessMode: &instantMode,
	}, "v11-hosting-edit-access-mode-0001")
	if err != nil {
		t.Fatal(err)
	}
	if editedDraft.AccessMode != slot.AccessInstant || editedDraft.Version != draft.Version+1 {
		t.Fatalf("draft access mode edit failed: %#v", editedDraft)
	}
	if _, err := slotService.PublishDraft(ctx, viewer.User.ID, draft.ID, editedDraft.Version, "v11-hosting-publish-future-mode-0001"); err != slot.ErrInvalidState {
		t.Fatalf("future access mode published before concurrency semantics: %v", err)
	}

	start := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	created, err := slotService.Create(ctx, viewer.User.ID, slot.CreateInput{
		Title: "Map integration", Activity: "coffee", PlaceText: "Integration Place",
		CanonicalPlaceID: &placeID, StartAt: &start, Capacity: 2,
	}, "v11-map-create-integration-0001")
	if err != nil {
		t.Fatal(err)
	}
	if created.CanonicalPlaceID == nil || *created.CanonicalPlaceID != placeID {
		t.Fatalf("canonical place missing from created slot: %#v", created)
	}

	fetched, err := slotService.Get(ctx, viewer.User.ID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fetched.CanonicalPlaceID == nil || *fetched.CanonicalPlaceID != placeID {
		t.Fatalf("canonical place missing from fetched slot: %#v", fetched)
	}

	_, err = slotService.Create(ctx, viewer.User.ID, slot.CreateInput{
		Title: "Other locality map integration", Activity: "coffee", PlaceText: "Other Locality Place",
		CanonicalPlaceID: &otherPlaceID, StartAt: &start, Capacity: 2,
	}, "v11-map-create-other-locality-0001")
	if err != nil {
		t.Fatal(err)
	}

	mapStore, err := NewCityMapStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	mapService, err := citymap.NewService(mapStore)
	if err != nil {
		t.Fatal(err)
	}
	clusters, err := mapService.Viewport(ctx, viewer.User.ID, localityID, citymap.Viewport{
		WestE6: 30000000, SouthE6: 50000000, EastE6: 31000000, NorthE6: 51000000,
		Zoom: 16, From: start.Add(-time.Hour), To: start.Add(time.Hour), Limit: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(clusters) != 1 || clusters[0].PlaceID == nil || *clusters[0].PlaceID != placeID || clusters[0].SlotCount != 1 {
		t.Fatalf("map leaked another locality or lost canonical place: %#v", clusters)
	}

	visibleSlots, err := mapService.PlaceSlots(ctx, viewer.User.ID, localityID, citymap.PlaceSlotsQuery{
		PlaceID: placeID, From: start.Add(-time.Hour), To: start.Add(time.Hour), Limit: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(visibleSlots) != 1 || visibleSlots[0].ID != created.ID {
		t.Fatalf("unexpected locality-scoped place slots: %#v", visibleSlots)
	}
	leakedSlots, err := mapService.PlaceSlots(ctx, viewer.User.ID, localityID, citymap.PlaceSlotsQuery{
		PlaceID: otherPlaceID, From: start.Add(-time.Hour), To: start.Add(time.Hour), Limit: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(leakedSlots) != 0 {
		t.Fatalf("map place detail leaked another locality: %#v", leakedSlots)
	}

	cleared, err := slotService.Edit(ctx, viewer.User.ID, created.ID, slot.EditInput{
		ExpectedVersion: created.Version, ClearCanonicalPlaceID: true,
	}, "v11-map-clear-place-0001")
	if err != nil {
		t.Fatal(err)
	}
	if cleared.CanonicalPlaceID != nil {
		t.Fatalf("canonical place was not cleared: %#v", cleared)
	}
}
