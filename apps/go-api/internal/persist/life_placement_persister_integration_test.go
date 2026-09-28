//go:build integration

// Package persist — life_placement_persister_integration_test.go : `match_life_placement` sur
// les VRAIES migrations (lot V2 du plan `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`).
//
// Ce que ces tests prouvent tient au SQL et au schema : l'aller-retour des colonnes (NULL compris),
// la vue `_latest` qui retient une PASSE ENTIERE, et les refus de la validation.
package persist

import (
	"context"
	"strings"
	"testing"
)

// placement : une ligne coherente — une vie de 10 s, 9 s mesurees, 1 100 ms d'exclusions.
func placement(xuid string, debut int64, mediane *float64, radar *float64, horsRadar *int64) LifePlacementInsert {
	return LifePlacementInsert{
		XUID: xuid, StartMS: debut, EndMS: debut + 10_000, DurationMS: 10_000,
		MeasuredMS: 9_000, MedianM: mediane, BeyondMS: horsRadar, RadarM: radar,
		CarrierMS: 300, TeamDownMS: 400, UnplacedMS: 200, TeammateUnplacedMS: 200, Kills: 2,
	}
}

func dureeMS(v int64) *int64 { return &v }

// TestLifePlacementPersister_AllerRetour — chaque colonne ecrite se relit par la vue, et un
// pointeur nil devient NULL (vie non mesuree, variante sans portee).
func TestLifePlacementPersister_AllerRetour(t *testing.T) {
	db := baseDeTest(t)
	err := NewLifePlacementPersister(db).PersistPass(context.Background(), LifePlacementBatch{
		MatchID: "m1", DecoderRev: "placement-rev-1",
		Rows: []LifePlacementInsert{
			placement("111", 0, metres(12.5), metres(18), dureeMS(3_000)),
			placement("222", 5_000, nil, nil, nil),
		},
	})
	if err != nil {
		t.Fatalf("PersistPass: %v", err)
	}
	type ligne struct {
		xuid, rev                                                        string
		debut, fin, duree, mesure, porteur, aTerre, nonSitue, coequipier int64
		mediane, radar                                                   *float64
		horsRadar                                                        *int64
		frags                                                            int
	}
	rows, err := db.Query(`SELECT xuid, decoder_rev, start_ms, end_ms, duration_ms, measured_ms,
		carrier_ms, team_down_ms, unplaced_ms, teammate_unplaced_ms, median_m, radar_m, beyond_ms, kills
		FROM match_life_placement_latest WHERE match_id = 'm1' ORDER BY xuid`)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	defer rows.Close() //nolint:errcheck
	var lues []ligne
	for rows.Next() {
		var l ligne
		if err := rows.Scan(&l.xuid, &l.rev, &l.debut, &l.fin, &l.duree, &l.mesure, &l.porteur,
			&l.aTerre, &l.nonSitue, &l.coequipier, &l.mediane, &l.radar, &l.horsRadar, &l.frags); err != nil {
			t.Fatalf("scan: %v", err)
		}
		lues = append(lues, l)
	}
	if len(lues) != 2 {
		t.Fatalf("%d lignes relues, attendu 2", len(lues))
	}
	a, b := lues[0], lues[1]
	if a.rev != "placement-rev-1" || a.debut != 0 || a.fin != 10_000 || a.duree != 10_000 ||
		a.mesure != 9_000 || a.porteur != 300 || a.aTerre != 400 || a.nonSitue != 200 ||
		a.coequipier != 200 || a.frags != 2 {
		t.Errorf("ligne 111 relue : %+v", a)
	}
	if a.mediane == nil || *a.mediane != 12.5 || a.radar == nil || *a.radar != 18 ||
		a.horsRadar == nil || *a.horsRadar != 3_000 {
		t.Errorf("ligne 111 : mediane %v, radar %v, hors radar %v", a.mediane, a.radar, a.horsRadar)
	}
	if b.mediane != nil || b.radar != nil || b.horsRadar != nil {
		t.Errorf("ligne 222 : les nil doivent se relire NULL (mediane %v, radar %v, hors radar %v)",
			b.mediane, b.radar, b.horsRadar)
	}
}

// TestLifePlacementPersister_LaVueRendLaDernierePasseEntiere — une passe B plus courte que la
// passe A ne laisse survivre AUCUNE ligne de A (l'unite de generation est la passe).
func TestLifePlacementPersister_LaVueRendLaDernierePasseEntiere(t *testing.T) {
	db := baseDeTest(t)
	p := NewLifePlacementPersister(db)
	ctx := context.Background()
	if err := p.PersistPass(ctx, LifePlacementBatch{MatchID: "m1", DecoderRev: "a", Rows: []LifePlacementInsert{
		placement("111", 0, nil, nil, nil), placement("111", 20_000, nil, nil, nil),
		placement("222", 0, nil, nil, nil),
	}}); err != nil {
		t.Fatalf("passe A: %v", err)
	}
	if err := p.PersistPass(ctx, LifePlacementBatch{MatchID: "m2", DecoderRev: "a", Rows: []LifePlacementInsert{
		placement("333", 0, nil, nil, nil),
	}}); err != nil {
		t.Fatalf("passe m2: %v", err)
	}
	if err := p.PersistPass(ctx, LifePlacementBatch{MatchID: "m1", DecoderRev: "b", Rows: []LifePlacementInsert{
		placement("111", 0, nil, nil, nil),
	}}); err != nil {
		t.Fatalf("passe B: %v", err)
	}
	var m1, m1RevA, m2, brutes int
	if err := db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM match_life_placement_latest WHERE match_id = 'm1'),
		(SELECT COUNT(*) FROM match_life_placement_latest WHERE match_id = 'm1' AND decoder_rev = 'a'),
		(SELECT COUNT(*) FROM match_life_placement_latest WHERE match_id = 'm2'),
		(SELECT COUNT(*) FROM match_life_placement)`).Scan(&m1, &m1RevA, &m2, &brutes); err != nil {
		t.Fatalf("select: %v", err)
	}
	if m1 != 1 || m1RevA != 0 || m2 != 1 || brutes != 5 {
		t.Errorf("vue : m1=%d (attendu 1), m1 rev a=%d (attendu 0), m2=%d (attendu 1), brutes=%d (attendu 5)",
			m1, m1RevA, m2, brutes)
	}
}

// TestLifePlacementPersister_PasseVide_RienNEstEcrit — aucune ligne, aucune passe.
func TestLifePlacementPersister_PasseVide_RienNEstEcrit(t *testing.T) {
	db := baseDeTest(t)
	if err := NewLifePlacementPersister(db).PersistPass(context.Background(),
		LifePlacementBatch{MatchID: "m1", DecoderRev: "a"}); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_life_placement`).Scan(&n); err != nil {
		t.Fatalf("select: %v", err)
	}
	if n != 0 {
		t.Errorf("%d lignes ecrites pour une passe vide", n)
	}
}

// TestLifePlacementPersister_Refus — chaque invariant refuse, et RIEN n'est ecrit (la validation
// precede la transaction).
func TestLifePlacementPersister_Refus(t *testing.T) {
	db := baseDeTest(t)
	p := NewLifePlacementPersister(db)
	casser := func(f func(*LifePlacementInsert)) LifePlacementInsert {
		r := placement("111", 0, metres(10), metres(18), dureeMS(1_000))
		f(&r)
		return r
	}
	for _, c := range []struct {
		nom, motif string
		batch      LifePlacementBatch
	}{
		{"match vide", "matchID vide", LifePlacementBatch{DecoderRev: "a"}},
		{"revision vide", "DecoderRev vide", LifePlacementBatch{MatchID: "m1"}},
		{"xuid vide", "xuid vide", LifePlacementBatch{MatchID: "m1", DecoderRev: "a",
			Rows: []LifePlacementInsert{casser(func(r *LifePlacementInsert) { r.XUID = "" })}}},
		{"duree fausse", "bornes", LifePlacementBatch{MatchID: "m1", DecoderRev: "a",
			Rows: []LifePlacementInsert{casser(func(r *LifePlacementInsert) { r.DurationMS = 1 })}}},
		{"cumuls courts", "ne couvrent pas", LifePlacementBatch{MatchID: "m1", DecoderRev: "a",
			Rows: []LifePlacementInsert{casser(func(r *LifePlacementInsert) { r.MeasuredMS = 1_000 })}}},
		{"radar sans hors radar", "ENSEMBLE", LifePlacementBatch{MatchID: "m1", DecoderRev: "a",
			Rows: []LifePlacementInsert{casser(func(r *LifePlacementInsert) { r.BeyondMS = nil })}}},
		{"hors radar au-dela du mesure", "incoherents", LifePlacementBatch{MatchID: "m1", DecoderRev: "a",
			Rows: []LifePlacementInsert{casser(func(r *LifePlacementInsert) { r.BeyondMS = dureeMS(9_001) })}}},
		{"mediane sans mesure", "sans instant mesure", LifePlacementBatch{MatchID: "m1", DecoderRev: "a",
			Rows: []LifePlacementInsert{casser(func(r *LifePlacementInsert) {
				r.CarrierMS += r.MeasuredMS
				r.MeasuredMS, r.BeyondMS = 0, dureeMS(0)
			})}}},
		{"frags negatifs", "frags negatifs", LifePlacementBatch{MatchID: "m1", DecoderRev: "a",
			Rows: []LifePlacementInsert{casser(func(r *LifePlacementInsert) { r.Kills = -1 })}}},
	} {
		t.Run(c.nom, func(t *testing.T) {
			err := p.PersistPass(context.Background(), c.batch)
			if err == nil || !strings.Contains(err.Error(), c.motif) {
				t.Fatalf("erreur %v, attendu un refus contenant %q", err, c.motif)
			}
		})
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_life_placement`).Scan(&n); err != nil {
		t.Fatalf("select: %v", err)
	}
	if n != 0 {
		t.Errorf("%d lignes ecrites malgre les refus", n)
	}
}
