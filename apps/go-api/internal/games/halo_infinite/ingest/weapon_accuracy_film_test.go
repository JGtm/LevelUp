package ingest

// weapon_accuracy_film_test.go — le mapper film -> weapon_accuracy + distance, sans film ni DuckDB.
// Verrouille : shots_fired = ShotsPaired, shots_landed = Hits, drops = 0 ; le pont FilmIndex->xuid
// (indice non resolu ecarte) ; l agregation de deux indices sur un meme xuid ; la ligne distance
// absente quand aucune touche n a de distance resolue ; l ordre stable.

import (
	"encoding/json"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const testDecoderRev = "whd-test"

// bucketsWith construit un histogramme de la bonne longueur avec les comptes donnes en tete.
func bucketsWith(counts ...int) []int {
	b := make([]int, filmdec.WeaponHitBucketCount())
	copy(b, counts)
	return b
}

// resolveFixed rend un pont FilmIndex->xuid a partir d une table ; "" pour un indice absent.
func resolveFixed(m map[int]string) func(int) string {
	return func(i int) string { return m[i] }
}

func TestMapWeaponAccuracyFilmSurfaces(t *testing.T) {
	stats := []filmdec.WeaponHitStats{
		{FilmIndex: 3, WeaponID: 1000, ShotsPaired: 10, Hits: 6, DistBuckets: bucketsWith(2, 1)},
		{FilmIndex: 5, WeaponID: 2000, ShotsPaired: 4, Hits: 4, DistBuckets: bucketsWith(0, 0, 4)},
	}
	acc, dist := MapWeaponAccuracyFilm("m1", stats, resolveFixed(map[int]string{3: "xuid(1)", 5: "xuid(2)"}), testDecoderRev)

	if len(acc) != 2 {
		t.Fatalf("attendu 2 lignes weapon_accuracy, obtenu %d", len(acc))
	}
	if acc[0].XUID != "xuid(1)" || acc[0].WeaponID != 1000 || acc[0].ShotsFired != 10 || acc[0].ShotsLanded != 6 || acc[0].Drops != 0 {
		t.Errorf("ligne 0 inattendue : %+v", acc[0])
	}
	if acc[0].MatchID != "m1" {
		t.Errorf("match_id non propage : %q", acc[0].MatchID)
	}
	if dist.MatchID != "m1" || dist.DecoderRev != testDecoderRev {
		t.Errorf("batch distance mal estampille : %+v", dist)
	}
	if len(dist.Rows) != 2 {
		t.Fatalf("attendu 2 lignes distance, obtenu %d", len(dist.Rows))
	}
	if dist.Rows[0].DistN != 3 {
		t.Errorf("dist_n cle 1 attendu 3 (2+1), obtenu %d", dist.Rows[0].DistN)
	}
	var hist []int
	if err := json.Unmarshal([]byte(dist.Rows[0].DistBucketJSON), &hist); err != nil {
		t.Fatalf("dist_bucket_json illisible : %v", err)
	}
	if len(hist) != filmdec.WeaponHitBucketCount() || hist[0] != 2 || hist[1] != 1 {
		t.Errorf("histogramme serialise inattendu : %v", hist)
	}
}

// TestMapWeaponAccuracyFilmSkips — indice non resolu, arme nulle, zero tir appariable : ecartes.
func TestMapWeaponAccuracyFilmSkips(t *testing.T) {
	stats := []filmdec.WeaponHitStats{
		{FilmIndex: 1, WeaponID: 1000, ShotsPaired: 9, Hits: 5, DistBuckets: bucketsWith(3)}, // indice non resolu
		{FilmIndex: 2, WeaponID: 0, ShotsPaired: 9, Hits: 5, DistBuckets: bucketsWith(3)},    // arme nulle
		{FilmIndex: 3, WeaponID: 1000, ShotsPaired: 0, Hits: 0, DistBuckets: bucketsWith()},  // zero tir appariable
	}
	acc, dist := MapWeaponAccuracyFilm("m1", stats, resolveFixed(map[int]string{2: "xuid(2)", 3: "xuid(3)"}), testDecoderRev)
	if len(acc) != 0 {
		t.Errorf("aucune ligne attendue, obtenu %d : %+v", len(acc), acc)
	}
	if len(dist.Rows) != 0 {
		t.Errorf("aucune ligne distance attendue, obtenu %d", len(dist.Rows))
	}
}

// TestMapWeaponAccuracyFilmAggregatesIndexCollision — deux FilmIndex sur le meme xuid+arme somment.
func TestMapWeaponAccuracyFilmAggregatesIndexCollision(t *testing.T) {
	stats := []filmdec.WeaponHitStats{
		{FilmIndex: 3, WeaponID: 1000, ShotsPaired: 6, Hits: 4, DistBuckets: bucketsWith(2)},
		{FilmIndex: 4, WeaponID: 1000, ShotsPaired: 5, Hits: 2, DistBuckets: bucketsWith(1)},
	}
	acc, dist := MapWeaponAccuracyFilm("m1", stats, resolveFixed(map[int]string{3: "xuid(1)", 4: "xuid(1)"}), testDecoderRev)
	if len(acc) != 1 {
		t.Fatalf("attendu 1 ligne agregee, obtenu %d", len(acc))
	}
	if acc[0].ShotsFired != 11 || acc[0].ShotsLanded != 6 {
		t.Errorf("agregation incorrecte : %+v (attendu 11/6)", acc[0])
	}
	if len(dist.Rows) != 1 || dist.Rows[0].DistN != 3 {
		t.Errorf("distance agregee incorrecte : %+v", dist.Rows)
	}
}

// TestMapWeaponAccuracyFilmNoDistanceRowWhenUnresolved — accuracy presente, distance absente quand
// aucune touche n a ses deux positions (dist_n = 0).
func TestMapWeaponAccuracyFilmNoDistanceRowWhenUnresolved(t *testing.T) {
	stats := []filmdec.WeaponHitStats{
		{FilmIndex: 3, WeaponID: 1000, ShotsPaired: 10, Hits: 6, DistBuckets: bucketsWith()},
	}
	acc, dist := MapWeaponAccuracyFilm("m1", stats, resolveFixed(map[int]string{3: "xuid(1)"}), testDecoderRev)
	if len(acc) != 1 {
		t.Fatalf("attendu 1 ligne accuracy, obtenu %d", len(acc))
	}
	if len(dist.Rows) != 0 {
		t.Errorf("aucune ligne distance attendue (dist_n=0), obtenu %d", len(dist.Rows))
	}
}
