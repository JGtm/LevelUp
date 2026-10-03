package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

// ---------------------------------------------------------------------------
// buildSoloMapBreakdown
// ---------------------------------------------------------------------------

// TestBuildSoloMapBreakdown_OrderedByFirstAppearance verrouille le tri des
// cartes par ordre CHRONOLOGIQUE de première apparition, calqué sur
// TestComputeMapBreakdown_OrderedByFirstAppearance (teammates_extra_test.go,
// commit 59e705690 — I12). "Early" (1 match) est jouée avant "Mid" (2 matchs)
// avant "Late" (3 matchs) : l'ordre fréquence (match_count DESC, ancien tri)
// serait l'inverse (Late>Mid>Early), donc ce test distingue les deux tris.
func TestBuildSoloMapBreakdown_OrderedByFirstAppearance(t *testing.T) {
	base := time.Date(2026, 5, 1, 18, 0, 0, 0, time.UTC)
	win := analysis.OutcomeWin
	current := []legacymatch.StatsMatchRow{
		{MatchID: "l1", StartTime: base.Add(2 * time.Hour), MapNameFR: "Late", Outcome: &win},
		{MatchID: "l2", StartTime: base.Add(150 * time.Minute), MapNameFR: "Late", Outcome: &win},
		{MatchID: "l3", StartTime: base.Add(3 * time.Hour), MapNameFR: "Late", Outcome: &win},
		{MatchID: "m1", StartTime: base.Add(1 * time.Hour), MapNameFR: "Mid", Outcome: &win},
		{MatchID: "m2", StartTime: base.Add(90 * time.Minute), MapNameFR: "Mid", Outcome: &win},
		{MatchID: "e1", StartTime: base, MapNameFR: "Early", Outcome: &win},
	}

	rows := buildSoloMapBreakdown(current, nil)

	want := []string{"Early", "Mid", "Late"}
	if len(rows) != len(want) {
		t.Fatalf("want %d maps, got %d (%v)", len(want), len(rows), rows)
	}
	for i, w := range want {
		if rows[i].MapUI != w {
			t.Errorf("rows[%d]: want %q, got %q (ordre chronologique cassé, got order: %v)",
				i, w, rows[i].MapUI, rows)
		}
	}
}

// TestBuildSoloMapBreakdown_TieBreakMapUI verrouille le tie-break déterministe
// (MapUI asc) quand deux cartes partagent le même firstSeen — l'itération
// d'une map Go n'étant pas ordonnée, un tri sans tie-break serait non
// déterministe d'un run à l'autre (même piège que computeMapBreakdown).
func TestBuildSoloMapBreakdown_TieBreakMapUI(t *testing.T) {
	same := time.Date(2026, 5, 1, 18, 0, 0, 0, time.UTC)
	win := analysis.OutcomeWin
	current := []legacymatch.StatsMatchRow{
		{MatchID: "z1", StartTime: same, MapNameFR: "Zanzibar", Outcome: &win},
		{MatchID: "a1", StartTime: same, MapNameFR: "Aquarius", Outcome: &win},
	}

	rows := buildSoloMapBreakdown(current, nil)

	if len(rows) != 2 || rows[0].MapUI != "Aquarius" || rows[1].MapUI != "Zanzibar" {
		t.Fatalf("want [Aquarius, Zanzibar] (tie-break MapUI asc), got %v", rows)
	}
}

// ---------------------------------------------------------------------------
// buildTopWeapons
// ---------------------------------------------------------------------------

// TestBuildTopWeapons_DepartageSurLeLibelle verrouille le passage de cette page à la
// doctrine canonique du top armes (topWeaponKillRows) : à FRAGS ÉGAUX le classement se
// départage sur le LIBELLÉ, et non plus sur l'identifiant d'arme. Ici l'identifiant et le
// libellé donnent des ordres OPPOSÉS (Bandit = id 20, Fusil = id 10), donc ce test
// distingue les deux doctrines. Il verrouille au passage les trois autres clauses du
// helper : grenade/mêlée écartée, ligne sans libellé écartée, agrégation par le COUPLE
// (identifiant, clé de registre).
func TestBuildTopWeapons_DepartageSurLeLibelle(t *testing.T) {
	rows := []port.WeaponKillRow{
		{WeaponID: 10, WeaponKey: "hinf_fusil", Label: "Fusil de combat", Kills: 3, Class: "shoulder"},
		{WeaponID: 20, WeaponKey: "hinf_bandit", Label: "Bandit", Kills: 2},
		{WeaponID: 20, WeaponKey: "hinf_bandit", Label: "Bandit", Kills: 1, Class: "shoulder"},
		{WeaponID: 30, WeaponKey: "hinf_grenade", Label: "Grenade", Kills: 9, IsGrenadeMelee: true},
		{WeaponID: 40, WeaponKey: "", Label: "", Kills: 5},
	}
	got := buildTopWeapons(rows, 10)
	if len(got) != 2 {
		t.Fatalf("attendu 2 armes (grenade/mêlée et ligne sans libellé écartées), obtenu %d : %+v",
			len(got), got)
	}
	if got[0].Label != "Bandit" || got[0].Kills != 3 {
		t.Errorf("1re place attendue Bandit/3 (départage alphabétique à 3 frags), obtenu %s/%d",
			got[0].Label, got[0].Kills)
	}
	if got[1].Label != "Fusil de combat" || got[1].Kills != 3 {
		t.Errorf("2e place attendue Fusil de combat/3, obtenu %s/%d", got[1].Label, got[1].Kills)
	}
	// Classe portée par la 1re valeur non vide du groupe agrégé.
	if got[0].Class != "shoulder" {
		t.Errorf("classe attendue shoulder sur Bandit (2e ligne du groupe), obtenu %q", got[0].Class)
	}
}

// TestBuildTopWeapons_PlafonneAuTopN : le cap est un plafond de COMPTE appliqué APRÈS tri.
func TestBuildTopWeapons_PlafonneAuTopN(t *testing.T) {
	rows := []port.WeaponKillRow{
		{WeaponID: 1, WeaponKey: "a", Label: "A", Kills: 1},
		{WeaponID: 2, WeaponKey: "b", Label: "B", Kills: 5},
		{WeaponID: 3, WeaponKey: "c", Label: "C", Kills: 3},
	}
	got := buildTopWeapons(rows, 2)
	if len(got) != 2 || got[0].Label != "B" || got[1].Label != "C" {
		t.Fatalf("attendu [B C], obtenu %+v", got)
	}
}
