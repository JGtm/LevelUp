package filmdec

// weapon_hits_test.go — test unitaire PUR du numerateur de precision (Lot 2). Aucune DuckDB,
// aucun film : les tirs et degats sont construits a la main. Il verrouille la definition de la
// touche (tir apparie a >=1 degat du meme attaquant dans W ; un tir -> une touche) et le bucketing
// de distance. Contrairement aux instruments TestLot1* (gardes LOT1_TRAME_FILM), ce test tourne
// toujours.

import "testing"

const utW = uint64(1_000_000) // 1 s

// statFor retrouve la ligne de stats d'une cle (film index, weapon id).
func statFor(stats []WeaponHitStats, fidx int, wid uint64) (WeaponHitStats, bool) {
	for _, s := range stats {
		if s.FilmIndex == fidx && s.WeaponID == wid {
			return s, true
		}
	}
	return WeaponHitStats{}, false
}

func TestPairWeaponHitsPaired(t *testing.T) {
	shots := []WeaponShot{{TimestampUS: 10_000_000, Attacker: 5, WeaponID: 100, FilmIndex: 0, HasPair: true}}
	dmg := []WeaponDamage{{TimestampUS: 10_500_000, VictimIdx: 3, ResponsibleIdx: 5}}
	stats := PairWeaponHits(shots, dmg, utW, nil)
	s, ok := statFor(stats, 0, 100)
	if !ok {
		t.Fatalf("cle (0,100) absente : %+v", stats)
	}
	if s.ShotsPaired != 1 || s.Hits != 1 {
		t.Fatalf("tir apparie : ShotsPaired=%d Hits=%d, veut 1/1", s.ShotsPaired, s.Hits)
	}
}

func TestPairWeaponHitsNotPaired(t *testing.T) {
	shots := []WeaponShot{{TimestampUS: 10_000_000, Attacker: 5, WeaponID: 100, HasPair: true}}
	// Degat du MEME instant mais d'un AUTRE attaquant (responsable 9) : pas de touche.
	dmg := []WeaponDamage{{TimestampUS: 10_100_000, VictimIdx: 3, ResponsibleIdx: 9}}
	stats := PairWeaponHits(shots, dmg, utW, nil)
	s, _ := statFor(stats, 0, 100)
	if s.ShotsPaired != 1 || s.Hits != 0 {
		t.Fatalf("attaquant different : ShotsPaired=%d Hits=%d, veut 1/0", s.ShotsPaired, s.Hits)
	}
}

func TestPairWeaponHitsOutOfWindow(t *testing.T) {
	shots := []WeaponShot{{TimestampUS: 10_000_000, Attacker: 5, WeaponID: 100, HasPair: true}}
	// Meme attaquant mais degat a 2 s : hors de la fenetre de 1 s.
	dmg := []WeaponDamage{{TimestampUS: 12_000_000, VictimIdx: 3, ResponsibleIdx: 5}}
	stats := PairWeaponHits(shots, dmg, utW, nil)
	s, _ := statFor(stats, 0, 100)
	if s.ShotsPaired != 1 || s.Hits != 0 {
		t.Fatalf("hors fenetre : ShotsPaired=%d Hits=%d, veut 1/0", s.ShotsPaired, s.Hits)
	}
}

func TestPairWeaponHitsOneShotOneHit(t *testing.T) {
	shots := []WeaponShot{{TimestampUS: 10_000_000, Attacker: 5, WeaponID: 100, HasPair: true}}
	// TROIS degats du meme attaquant dans la fenetre : un seul tir ne compte qu'UNE touche.
	dmg := []WeaponDamage{
		{TimestampUS: 10_100_000, VictimIdx: 3, ResponsibleIdx: 5},
		{TimestampUS: 10_200_000, VictimIdx: 3, ResponsibleIdx: 5},
		{TimestampUS: 10_300_000, VictimIdx: 3, ResponsibleIdx: 5},
	}
	stats := PairWeaponHits(shots, dmg, utW, nil)
	s, _ := statFor(stats, 0, 100)
	if s.ShotsPaired != 1 || s.Hits != 1 {
		t.Fatalf("un tir -> une touche : ShotsPaired=%d Hits=%d, veut 1/1", s.ShotsPaired, s.Hits)
	}
}

func TestPairWeaponHitsSkipsUnpairable(t *testing.T) {
	shots := []WeaponShot{
		{TimestampUS: 10_000_000, Attacker: 5, WeaponID: 100, HasPair: true},
		{TimestampUS: 10_000_000, Attacker: 5, WeaponID: 100, HasPair: false}, // illisible : ecarte
	}
	dmg := []WeaponDamage{{TimestampUS: 10_100_000, VictimIdx: 3, ResponsibleIdx: 5}}
	stats := PairWeaponHits(shots, dmg, utW, nil)
	s, _ := statFor(stats, 0, 100)
	if s.ShotsPaired != 1 {
		t.Fatalf("tir non appariable compte : ShotsPaired=%d, veut 1", s.ShotsPaired)
	}
}

func TestPairWeaponHitsDistanceBucket(t *testing.T) {
	shots := []WeaponShot{{TimestampUS: 10_000_000, Attacker: 5, WeaponID: 100, HasPair: true}}
	dmg := []WeaponDamage{{TimestampUS: 10_100_000, VictimIdx: 3, ResponsibleIdx: 5}}
	// Resolveur constant : 12 m -> tranche de WeaponHitBucket(12).
	dist := func(WeaponDamage) (float64, bool) { return 12.0, true }
	stats := PairWeaponHits(shots, dmg, utW, dist)
	s, _ := statFor(stats, 0, 100)
	if len(s.DistBuckets) != WeaponHitBucketCount() {
		t.Fatalf("DistBuckets len=%d, veut %d", len(s.DistBuckets), WeaponHitBucketCount())
	}
	if got := s.DistBuckets[WeaponHitBucket(12)]; got != 1 {
		t.Fatalf("tranche de 12 m : compte=%d, veut 1", got)
	}
	total := 0
	for _, c := range s.DistBuckets {
		total += c
	}
	if total != 1 {
		t.Fatalf("total buckets=%d, veut 1", total)
	}
}

func TestPairWeaponHitsDistanceUnresolved(t *testing.T) {
	shots := []WeaponShot{{TimestampUS: 10_000_000, Attacker: 5, WeaponID: 100, HasPair: true}}
	dmg := []WeaponDamage{{TimestampUS: 10_100_000, VictimIdx: 3, ResponsibleIdx: 5}}
	// Position non resoluble : la touche est comptee, aucune tranche remplie.
	dist := func(WeaponDamage) (float64, bool) { return 0, false }
	stats := PairWeaponHits(shots, dmg, utW, dist)
	s, _ := statFor(stats, 0, 100)
	if s.Hits != 1 {
		t.Fatalf("touche non comptee sans position : Hits=%d, veut 1", s.Hits)
	}
	for i, c := range s.DistBuckets {
		if c != 0 {
			t.Fatalf("tranche %d remplie sans position : %d", i, c)
		}
	}
}

func TestPairWeaponHitsKeysBySlot(t *testing.T) {
	// Deux tireurs (film index) et deux armes : quatre cles distinctes.
	shots := []WeaponShot{
		{TimestampUS: 1_000_000, Attacker: 1, WeaponID: 10, FilmIndex: 0, HasPair: true},
		{TimestampUS: 2_000_000, Attacker: 2, WeaponID: 10, FilmIndex: 1, HasPair: true},
		{TimestampUS: 3_000_000, Attacker: 1, WeaponID: 20, FilmIndex: 0, HasPair: true},
	}
	stats := PairWeaponHits(shots, nil, utW, nil)
	if len(stats) != 3 {
		t.Fatalf("cles distinctes : %d, veut 3 (%+v)", len(stats), stats)
	}
}

func TestWeaponHitBucket(t *testing.T) {
	// Bornes {2,5,10,15,25,40} -> 7 tranches (0..6).
	cases := []struct {
		d    float64
		want int
	}{
		{0, 0}, {1.9, 0}, {2, 1}, {4.9, 1}, {5, 2}, {12, 3}, {24, 4}, {30, 5}, {40, 6}, {1000, 6},
	}
	for _, c := range cases {
		if got := WeaponHitBucket(c.d); got != c.want {
			t.Errorf("WeaponHitBucket(%.1f)=%d, veut %d", c.d, got, c.want)
		}
	}
	if WeaponHitBucketCount() != len(WeaponHitDistanceEdges)+1 {
		t.Fatalf("WeaponHitBucketCount=%d incoherent avec %d bornes", WeaponHitBucketCount(), len(WeaponHitDistanceEdges))
	}
}
