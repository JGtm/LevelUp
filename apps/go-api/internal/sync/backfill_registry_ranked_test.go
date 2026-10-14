// Package sync — backfill_registry_ranked_test.go : le rattrapage de match_registry.is_ranked
// depuis le CSR par match (A4), sur le schéma shared réel en mémoire. Aucune base réelle.
package sync

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
)

// seedRankedRepairScenario : cinq matchs couvrant chaque cas.
//
//	m1 is_ranked FALSE + CSR   -> corrigé
//	m2 is_ranked NULL  + CSR   -> corrigé
//	m3 is_ranked TRUE  + CSR   -> déjà classé
//	m4 is_ranked TRUE  sans CSR -> classé sans CSR (jamais déclassé)
//	m5 is_ranked FALSE sans CSR -> intact
func seedRankedRepairScenario(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, q := range []string{
		`INSERT INTO match_registry (match_id, start_time, is_ranked) VALUES
			('m1', TIMESTAMP '2026-10-01 10:00:00', FALSE),
			('m2', TIMESTAMP '2026-10-02 10:00:00', NULL),
			('m3', TIMESTAMP '2026-10-03 10:00:00', TRUE),
			('m4', TIMESTAMP '2026-10-04 10:00:00', TRUE),
			('m5', TIMESTAMP '2026-10-05 10:00:00', FALSE)`,
		`INSERT INTO match_csrs (match_id, xuid, rating_type, rating_value, tier, sub_tier, tier_label)
			VALUES ('m1', 'x1', 'CSR', 1000, 'Gold', 1, 'Or I'),
			       ('m1', 'x2', 'CSR', 1100, 'Gold', 2, 'Or II'),
			       ('m2', 'x1', 'CSR', 1050, 'Gold', 1, 'Or I'),
			       ('m3', 'x1', 'CSR', 1500, 'Diamond', 1, 'Diamant I')`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
}

func isRankedOf(t *testing.T, db *sql.DB, matchID string) sql.NullBool {
	t.Helper()
	var v sql.NullBool
	if err := db.QueryRow(`SELECT is_ranked FROM match_registry WHERE match_id = ?`, matchID).Scan(&v); err != nil {
		t.Fatalf("lecture is_ranked %s: %v", matchID, err)
	}
	return v
}

func TestBackfillRegistryIsRanked_DryRunThenWriteThenIdempotent(t *testing.T) {
	db := openSharedForCSRBackfill(t)
	seedRankedRepairScenario(t, db)
	ctx := context.Background()

	dry, err := BackfillRegistryIsRanked(ctx, db, RegistryRankedOptions{DryRun: true})
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if dry.Candidates != 2 || dry.Fixed != 2 || dry.AlreadyRanked != 1 || dry.RankedWithoutCSR != 1 {
		t.Errorf("dry-run stats = %+v, want candidates=2 fixed=2 already=1 sans_csr=1", dry)
	}
	if isRankedOf(t, db, "m1").Bool || isRankedOf(t, db, "m2").Bool {
		t.Fatal("--dry-run a écrit")
	}

	res, err := BackfillRegistryIsRanked(ctx, db, RegistryRankedOptions{})
	if err != nil {
		t.Fatalf("écriture: %v", err)
	}
	if res.Candidates != 2 || res.Fixed != 2 || res.Errors != 0 || res.AlreadyRanked != 1 || res.RankedWithoutCSR != 1 {
		t.Errorf("stats = %+v, want candidates=2 fixed=2 errors=0 already=1 sans_csr=1", res)
	}
	for id, want := range map[string]bool{"m1": true, "m2": true, "m3": true, "m4": true, "m5": false} {
		if got := isRankedOf(t, db, id); got.Bool != want {
			t.Errorf("%s is_ranked = %v, want %v", id, got.Bool, want)
		}
	}

	again, err := BackfillRegistryIsRanked(ctx, db, RegistryRankedOptions{})
	if err != nil {
		t.Fatalf("seconde passe: %v", err)
	}
	if again.Candidates != 0 || again.Fixed != 0 {
		t.Errorf("seconde passe = %+v, want candidates=0 fixed=0 (idempotent)", again)
	}
	// Le garde ne touche rien d'autre que is_ranked.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_registry`).Scan(&n); err != nil || n != 5 {
		t.Errorf("match_registry : %d lignes (err %v), want 5", n, err)
	}
}

// Un CSR ancien d'un autre type (LUSR) ne prouve pas un match classé.
func TestBackfillRegistryIsRanked_OnlyCSRRatingType(t *testing.T) {
	db := openSharedForCSRBackfill(t)
	if _, err := db.Exec(`INSERT INTO match_registry (match_id, start_time, is_ranked)
		VALUES ('m1', TIMESTAMP '2026-10-01 10:00:00', FALSE)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO match_csrs (match_id, xuid, rating_type, rating_value)
		VALUES ('m1', 'x1', 'LUSR', 1000)`); err != nil {
		t.Fatal(err)
	}
	res, err := BackfillRegistryIsRanked(context.Background(), db, RegistryRankedOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Candidates != 0 || isRankedOf(t, db, "m1").Bool {
		t.Errorf("une ligne non-CSR ne doit pas classer le match : %+v", res)
	}
}

func TestBackfillRegistryIsRanked_NilDB(t *testing.T) {
	if _, err := BackfillRegistryIsRanked(context.Background(), nil, RegistryRankedOptions{}); err == nil {
		t.Error("sharedDB nil : erreur attendue")
	}
}

// Candidature des rattrapages CSR : un match dont le nom n'est pas résolu est repêché par le
// playlist_id de la référence (sans décider de son classement).
func TestRankedCandidates_ByReferencePlaylistID(t *testing.T) {
	db := openSharedForCSRBackfill(t)
	refID := rankedplaylists.All()[0].AssetID
	for _, q := range []string{
		`INSERT INTO match_registry (match_id, start_time, playlist_id, playlist_name, is_ranked) VALUES
			('m-ref', TIMESTAMP '2026-10-01 10:00:00', '` + refID + `', NULL, FALSE),
			('m-name', TIMESTAMP '2026-10-02 10:00:00', 'autre', 'Ranked Arena', FALSE),
			('m-flag', TIMESTAMP '2026-10-03 10:00:00', 'autre', 'Quick Play', TRUE),
			('m-none', TIMESTAMP '2026-10-04 10:00:00', 'autre', 'Quick Play', FALSE)`,
		`INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome)
			SELECT match_id, 'xuid-A', 'A', 0, 2 FROM match_registry`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	want := map[string]bool{"m-ref": true, "m-name": true, "m-flag": true}

	player, err := loadRankedMatchesForCSRBackfill(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	shared, err := loadRankedMatchesForSharedCSRBackfill(context.Background(), db, "xuid-A")
	if err != nil {
		t.Fatal(err)
	}
	for label, got := range map[string][]MatchRegistryRow{"joueur": player, "partagé": shared} {
		ids := map[string]bool{}
		for _, r := range got {
			ids[r.MatchID] = true
			if r.IsRanked {
				t.Errorf("%s : un candidat ne porte pas de verdict de classement (%s)", label, r.MatchID)
			}
		}
		if len(ids) != len(want) {
			t.Errorf("%s : candidats %v, want %v", label, ids, want)
		}
		for id := range want {
			if !ids[id] {
				t.Errorf("%s : %s doit être candidat", label, id)
			}
		}
	}
}
