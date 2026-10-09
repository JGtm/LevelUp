// Package service — synthesis_weapon_records_test.go : la section « Records de distance par
// arme » vue du service (plan .ai/V7.5/PLAN_RECORDS_DISTANCE_2026-09-20.md).
//
// Ce que ces tests verrouillent :
//
//  1. le record est LE frag, avec sa clé (match, instant) et le contexte du match lu dans le
//     scope canonique — sans requête neuve ;
//  2. seuls les frags DU JOUEUR comptent (côté tueur) ; ses morts, jamais ;
//  3. les classes sans sens de distance sont écartées ET nommées avec leur effectif, et
//     comptent dans la couverture (elles sont mesurées, seulement pas tracées) ;
//  4. registre muet ou en panne, libellés en panne : la section survit ;
//  5. repo nil / scope vide / capability absente / erreur SQL / zéro frag : pas de section,
//     jamais de panique.
package service

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// wrecKill : un frag mesuré du côté tueur, dans un match nommé.
func wrecKill(weapon, match string, timeMS int64, dist float64) analysis.MeasuredKill {
	k := wrKill(weapon, analysis.SideKiller, timeMS, dist, 0)
	k.MatchID = match
	return k
}

// wrecCanonRow : un match canonique avec sa carte (FR + EN) et sa date.
func wrecCanonRow(matchID string, kills int, started time.Time, mapFR, mapEN string) canonical.PlayerMatchRow {
	return canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID: matchID, StartedAtUTC: started,
			Map: &canonical.AssetReference{DefaultLabel: mapEN, Labels: map[string]string{"fr": mapFR}},
		},
		Self: canonical.MatchParticipant{Kills: &kills},
	}
}

func wrecQuery(repo port.WeaponRangeRepository, rows []canonical.PlayerMatchRow) weaponRecordsQuery {
	return weaponRecordsQuery{Repo: repo, TitleSlug: "halo_infinite", Gamertag: "GT", Rows: rows}
}

func TestBuildWeaponRecordsSection_Nominal(t *testing.T) {
	t0 := time.Date(2026, 8, 14, 20, 0, 0, 0, time.UTC)
	rows := []canonical.PlayerMatchRow{
		wrecCanonRow("m1", 9, t0, "Béhémoth", "Behemoth"),
		wrecCanonRow("m2", 7, t0.Add(time.Hour), "Recharge", "Recharge"),
	}
	kills := []analysis.MeasuredKill{
		wrecKill("hinf_br75", "m1", 10, 12),
		wrecKill("hinf_br75", "m2", 20, 52.7), // le record du BR, en m2
		wrecKill("hinf_br75", "m1", 30, 14),
		wrecKill("hinf_s7_sniper", "m1", 40, 96.4),
		wrKill("hinf_s7_sniper", analysis.SideVictim, 50, 118.3, 0), // MA mort : ignorée
	}
	repo := &mockWeaponRangeRepo{
		kills: kills,
		dims: map[string]port.WeaponDimensions{
			"hinf_br75": {Class: "shoulder", Role: "precision"}, "hinf_s7_sniper": {Class: "heavy", Role: "sniper"},
		},
		labels: map[string]port.WeaponLabel{"hinf_br75": {Label: "BR75", LabelEN: "BR75"}},
	}

	block := buildWeaponRecordsSection(context.Background(), wrecQuery(repo, rows))
	if block == nil {
		t.Fatal("section nil, attendue peuplée")
	}
	if repo.killCalls != 1 || repo.dimCalls != 1 || repo.labelCalls != 1 || repo.openCalls != 0 {
		t.Errorf("appels repo kills/dims/labels/openings = %d/%d/%d/%d, attendu 1/1/1/0",
			repo.killCalls, repo.dimCalls, repo.labelCalls, repo.openCalls)
	}
	if repo.lastDimSlug != "halo_infinite" || len(repo.lastFilters.MatchIDs) != 2 || repo.lastFilters.Gamertag != "GT" {
		t.Errorf("scope = %+v (slug dims %q), attendu 2 match_id + gamertag + slug du titre",
			repo.lastFilters, repo.lastDimSlug)
	}
	if block.MeasuredKills != 4 || block.TotalKills != 16 {
		t.Errorf("couverture = %d/%d, attendu 4 frags mesurés (la mort ne compte pas) sur 16",
			block.MeasuredKills, block.TotalKills)
	}
	if len(block.Weapons) != 2 || block.Weapons[0].WeaponKey != "hinf_br75" || block.Weapons[1].WeaponKey != "hinf_s7_sniper" {
		t.Fatalf("armes = %+v, attendu BR puis sniper (record croissant)", block.Weapons)
	}
	br := block.Weapons[0]
	if br.Label != "BR75" || br.Class != "shoulder" || br.Measured != 3 {
		t.Errorf("BR = %+v, attendu libellé BR75, classe shoulder, 3 mesures", br)
	}
	if math.Abs(br.RecordM-52.7) > epsRange || math.Abs(br.MedianM-14) > epsRange {
		t.Errorf("BR record/médiane = %v/%v, attendu 52,7 / 14", br.RecordM, br.MedianM)
	}
	if br.Record.MatchID != "m2" || br.Record.TimeMS != 20 {
		t.Errorf("frag du record = %s/%d, attendu m2/20", br.Record.MatchID, br.Record.TimeMS)
	}
	if br.Record.MapLabel != "Recharge" || br.Record.StartedAt == nil || !br.Record.StartedAt.Equal(t0.Add(time.Hour)) {
		t.Errorf("contexte du record = %+v, attendu Recharge à t0+1h", br.Record)
	}
	sn := block.Weapons[1]
	if sn.Record.MapLabel != "Béhémoth" || sn.Record.MapLabelEN != "Behemoth" {
		t.Errorf("carte du sniper = %q / %q, attendu Béhémoth / Behemoth", sn.Record.MapLabel, sn.Record.MapLabelEN)
	}
	if sn.Label != "" || sn.LabelEN != "" {
		t.Errorf("libellé du sniper = %q, attendu vide (clé inconnue de la metadata, repli front)", sn.Label)
	}
	if block.Excluded != nil {
		t.Errorf("écartés = %+v, attendu aucun (omis du JSON)", block.Excluded)
	}
}

// Les classes sans sens de distance sont écartées ET nommées ; elles restent dans la couverture.
func TestBuildWeaponRecordsSection_ClassesEcarteesNommees(t *testing.T) {
	kills := []analysis.MeasuredKill{
		wrecKill("hinf_br75", "m1", 1, 20),
		wrecKill("hinf_environment", "m1", 2, 3), wrecKill("hinf_environment", "m1", 3, 40),
		wrecKill("hinf_repulsor", "m1", 4, 2),
		wrecKill("h5_energy_sword", "m1", 5, 1.5), wrecKill("h5_energy_sword", "m1", 6, 1.7),
		wrecKill("hinf_warthog", "m1", 7, 4), // véhicule : GARDÉ, son record se lit
	}
	repo := &mockWeaponRangeRepo{
		kills: kills,
		dims: map[string]port.WeaponDimensions{
			"hinf_br75": {Class: "shoulder"}, "hinf_environment": {Class: "environmental"},
			"hinf_repulsor": {Class: "equipment"}, "h5_energy_sword": {Class: "melee"},
			"hinf_warthog": {Class: "vehicle"},
		},
		labels: map[string]port.WeaponLabel{"hinf_environment": {Label: "Chute et environnement", LabelEN: "Environment"}},
	}

	block := buildWeaponRecordsSection(context.Background(), wrecQuery(repo, wrCanonRows(1, 20, 5)))
	if block == nil {
		t.Fatal("section nil")
	}
	if len(block.Weapons) != 2 || block.Weapons[0].WeaponKey != "hinf_warthog" || block.Weapons[1].WeaponKey != "hinf_br75" {
		t.Errorf("armes = %+v, attendu Warthog (4 m) puis BR (20 m)", block.Weapons)
	}
	if block.MeasuredKills != 7 {
		t.Errorf("frags mesurés = %d, attendu 7 (les écartés sont mesurés, pas tracés)", block.MeasuredKills)
	}
	if len(block.Excluded) != 3 {
		t.Fatalf("écartés = %+v, attendu 3 armes", block.Excluded)
	}
	// Effectif décroissant, puis clé : environnement (2), épée (2) — « h5_… » < « hinf_… » —, répulseur (1).
	want := []struct {
		key, class, label string
		n                 int
	}{
		{"h5_energy_sword", "melee", "", 2},
		{"hinf_environment", "environmental", "Chute et environnement", 2},
		{"hinf_repulsor", "equipment", "", 1},
	}
	for i, w := range want {
		e := block.Excluded[i]
		if e.WeaponKey != w.key || e.Class != w.class || e.Measured != w.n || e.Label != w.label {
			t.Errorf("écarté[%d] = %+v, attendu %+v", i, e, w)
		}
	}
}

// Registre muet (map vide) ou en panne : rien n'est écarté, la classe reste vide, la section survit.
func TestBuildWeaponRecordsSection_RegistreMuetOuEnPanne(t *testing.T) {
	kills := []analysis.MeasuredKill{wrecKill("hinf_environment", "m1", 1, 40), wrecKill("hinf_br75", "m1", 2, 20)}
	for _, c := range []struct {
		nom  string
		repo *mockWeaponRangeRepo
	}{
		{"registre muet", &mockWeaponRangeRepo{kills: kills}},
		{"registre en panne", &mockWeaponRangeRepo{kills: kills, dimsErr: errors.New("metadata indisponible")}},
	} {
		t.Run(c.nom, func(t *testing.T) {
			block := buildWeaponRecordsSection(context.Background(), wrecQuery(c.repo, wrCanonRows(1, 9, 5)))
			if block == nil || len(block.Weapons) != 2 || len(block.Excluded) != 0 {
				t.Fatalf("section = %+v, attendue peuplée de 2 armes sans exclusion", block)
			}
			if block.Weapons[0].Class != "" {
				t.Errorf("classe = %q, attendu vide (le web retombe sur la couleur neutre)", block.Weapons[0].Class)
			}
		})
	}
}

func TestBuildWeaponRecordsSection_LibellesEnPanne_LaSectionSurvit(t *testing.T) {
	repo := &mockWeaponRangeRepo{
		kills: []analysis.MeasuredKill{wrecKill("hinf_br75", "m1", 1, 20)}, labelsErr: errors.New("boom"),
	}
	block := buildWeaponRecordsSection(context.Background(), wrecQuery(repo, wrCanonRows(1, 9, 5)))
	if block == nil || len(block.Weapons) != 1 || block.Weapons[0].Label != "" || block.Weapons[0].WeaponKey != "hinf_br75" {
		t.Fatalf("section = %+v, attendue peuplée avec la clé conservée et le libellé vide", block)
	}
}

// Les chemins qui ne publient rien : jamais de panique, jamais une section vide.
func TestBuildWeaponRecordsSection_DegradationsSansSection(t *testing.T) {
	unFrag := []analysis.MeasuredKill{wrecKill("hinf_br75", "m1", 1, 20)}
	sansGamertag := wrecQuery(&mockWeaponRangeRepo{kills: unFrag}, wrCanonRows(1, 9, 5))
	sansGamertag.Gamertag = ""
	cas := []struct {
		nom string
		q   weaponRecordsQuery
	}{
		{"repo non câblé", wrecQuery(nil, wrCanonRows(1, 9, 5))},
		{"scope vide", wrecQuery(&mockWeaponRangeRepo{kills: unFrag}, nil)},
		{"gamertag vide", sansGamertag},
		{"capability absente", wrecQuery(&mockWeaponRangeRepo{killsErr: games.ErrCapabilityNotSupported}, wrCanonRows(1, 9, 5))},
		{"erreur SQL", wrecQuery(&mockWeaponRangeRepo{killsErr: errors.New("boom")}, wrCanonRows(1, 9, 5))},
		{"zéro frag mesuré", wrecQuery(&mockWeaponRangeRepo{}, wrCanonRows(1, 9, 5))},
		{"seulement mes morts", wrecQuery(&mockWeaponRangeRepo{kills: wrKills("hinf_br75", analysis.SideVictim, 3, 20, 0)}, wrCanonRows(1, 9, 5))},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if block := buildWeaponRecordsSection(context.Background(), c.q); block != nil {
				t.Errorf("section = %+v, attendu nil", block)
			}
		})
	}
}
