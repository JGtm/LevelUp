package teammates

import (
	"context"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// fakeWeaponAccuracyRepo mocke port.WeaponAccuracyRepository pour les tests du builder
// buildSquadWeaponAccuracy (précision native par arme, comparatif escouade).
type fakeWeaponAccuracyRepo struct {
	rows []port.WeaponAccuracyRow
	err  error
}

func (f *fakeWeaponAccuracyRepo) LoadWeaponAccuracyAggregated(
	_ context.Context, _ string, _ port.WeaponAccuracyFilters,
) ([]port.WeaponAccuracyRow, error) {
	return f.rows, f.err
}

func teammateWithXUID(gt, xuid string) domain.TeammateRow {
	x := xuid
	return domain.TeammateRow{Gamertag: gt, XUID: &x}
}

// TestBuildSquadWeaponAccuracy_AggregatesByRole vérifie le regroupement PAR RÔLE : deux
// armes du même rôle (precision) sont sommées en 1 barre, la précision par joueur = RAW
// ΣShotsLanded/ΣShotsFired (jamais la moyenne des ratios par arme), le pivot xuid→gamertag
// et le tri ASC par total de tirs.
func TestBuildSquadWeaponAccuracy_AggregatesByRole(t *testing.T) {
	svc := &TeammatesService{
		titleSlug: "halo_5",
		weaponAccuracyRepo: &fakeWeaponAccuracyRepo{
			rows: []port.WeaponAccuracyRow{
				// Rôle precision = 2 armes (BR75 id 10, DMR id 11), classe shoulder.
				// main : BR75 40/150 + DMR 10/50 → Σ 50/200 = 0.25 RAW (la moyenne des
				// ratios donnerait ~0.233 → l'assertion 0.25 prouve le calcul RAW).
				{XUID: "x_main", WeaponID: 10, ShotsFired: 150, ShotsLanded: 40, Class: "shoulder", Role: "precision"},
				{XUID: "x_main", WeaponID: 11, ShotsFired: 50, ShotsLanded: 10, Class: "shoulder", Role: "precision"},
				// friend1 : BR75 30/100 → 0.3 sur le rôle precision.
				{XUID: "x_f1", WeaponID: 10, ShotsFired: 100, ShotsLanded: 30, Class: "shoulder", Role: "precision"},
				// Rôle sniper : main seul 8/10=0.8 → total tirs 10.
				{XUID: "x_main", WeaponID: 20, ShotsFired: 10, ShotsLanded: 8, Class: "shoulder", Role: "sniper"},
			},
		},
	}
	allSquadRows := []domain.SquadMatchRow{{MatchID: "m1"}}
	teammates := []domain.TeammateRow{teammateWithXUID("friend1", "x_f1")}

	got := svc.buildSquadWeaponAccuracy(context.Background(), allSquadRows, "main", "x_main", teammates)
	if got == nil {
		t.Fatal("want non-nil accuracy")
	}
	if len(got.Players) != 2 || got.Players[0] != "main" || got.Players[1] != "friend1" {
		t.Fatalf("Players want [main friend1] (ordre canonique), got %v", got.Players)
	}
	if len(got.Bars) != 2 {
		t.Fatalf("want 2 role bars, got %d", len(got.Bars))
	}
	// Tri ASC par total de tirs : sniper (10) avant precision (300).
	if got.Bars[0].Role != "sniper" || got.Bars[1].Role != "precision" {
		t.Errorf("tri ASC total tirs attendu [sniper, precision], got [%s, %s]", got.Bars[0].Role, got.Bars[1].Role)
	}
	prec := got.Bars[1]
	if prec.AccuracyByPlayer["main"] != 0.25 {
		t.Errorf("main precision RAW (ΣLanded/ΣFired = 50/200) want 0.25, got %v", prec.AccuracyByPlayer["main"])
	}
	if prec.AccuracyByPlayer["friend1"] != 0.3 {
		t.Errorf("friend1 precision want 0.3, got %v", prec.AccuracyByPlayer["friend1"])
	}
	if prec.ShotsFiredByPlayer["main"] != 200 || prec.ShotsFiredByPlayer["friend1"] != 100 {
		t.Errorf("precision tirs par joueur inattendus: %+v", prec.ShotsFiredByPlayer)
	}
	if prec.TotalShotsSquad != 300 {
		t.Errorf("precision total tirs want 300, got %d", prec.TotalShotsSquad)
	}
	// Sniper : main uniquement (friend1 n'a pas tiré → pas d'entrée).
	sn := got.Bars[0]
	if sn.AccuracyByPlayer["main"] != 0.8 {
		t.Errorf("main sniper précision want 0.8, got %v", sn.AccuracyByPlayer["main"])
	}
	if _, ok := sn.AccuracyByPlayer["friend1"]; ok {
		t.Errorf("friend1 ne devrait pas avoir de précision sniper (aucun tir)")
	}
}

// TestBuildSquadWeaponAccuracy_NilRepo : sans repo câblé → nil (best-effort).
func TestBuildSquadWeaponAccuracy_NilRepo(t *testing.T) {
	svc := &TeammatesService{titleSlug: "halo_5"} // weaponAccuracyRepo == nil
	got := svc.buildSquadWeaponAccuracy(context.Background(),
		[]domain.SquadMatchRow{{MatchID: "m1"}}, "main", "x_main",
		[]domain.TeammateRow{teammateWithXUID("friend1", "x_f1")})
	if got != nil {
		t.Errorf("repo nil → want nil, got %+v", got)
	}
}

// TestBuildSquadWeaponAccuracy_CapabilityAbsent : ErrCapabilityNotSupported (Halo Infinite,
// pas de table weapon_accuracy) → nil sans erreur remontée.
func TestBuildSquadWeaponAccuracy_CapabilityAbsent(t *testing.T) {
	svc := &TeammatesService{
		titleSlug:          "halo_infinite",
		weaponAccuracyRepo: &fakeWeaponAccuracyRepo{err: games.ErrCapabilityNotSupported},
	}
	got := svc.buildSquadWeaponAccuracy(context.Background(),
		[]domain.SquadMatchRow{{MatchID: "m1"}}, "main", "x_main",
		[]domain.TeammateRow{teammateWithXUID("friend1", "x_f1")})
	if got != nil {
		t.Errorf("ErrCapabilityNotSupported → want nil, got %+v", got)
	}
}

// TestBuildSquadWeaponAccuracy_ExcludesNonAccuracyClasses : une grenade lancée
// (shots_fired > 0) mais de classe grenade ne doit PAS apparaître (pas de « tir au but »),
// tandis qu'une arme à tir valide subsiste (regroupée sous son rôle).
func TestBuildSquadWeaponAccuracy_ExcludesNonAccuracyClasses(t *testing.T) {
	svc := &TeammatesService{
		titleSlug: "halo_5",
		weaponAccuracyRepo: &fakeWeaponAccuracyRepo{
			rows: []port.WeaponAccuracyRow{
				{XUID: "x_main", WeaponID: 0, ShotsFired: 20, ShotsLanded: 5, Class: domain.FragClassGrenade, Role: "grenade"},
				{XUID: "x_main", WeaponID: 10, ShotsFired: 100, ShotsLanded: 40, Class: "shoulder", Role: "precision"},
			},
		},
	}
	got := svc.buildSquadWeaponAccuracy(context.Background(),
		[]domain.SquadMatchRow{{MatchID: "m1"}}, "main", "x_main",
		[]domain.TeammateRow{teammateWithXUID("friend1", "x_f1")})
	if got == nil {
		t.Fatal("want non-nil (precision valide)")
	}
	for _, b := range got.Bars {
		if b.Role == "grenade" {
			t.Errorf("la classe grenade ne doit pas apparaître dans la précision par rôle")
		}
	}
	if len(got.Bars) != 1 || got.Bars[0].Role != "precision" {
		t.Errorf("want [precision] uniquement, got %+v", got.Bars)
	}
}

// TestBuildSquadWeaponAccuracy_ExcludesEmptyRole : une arme à tir (classe à précision
// pertinente) mais SANS rôle résolu (registre incomplet) est écartée — pas de barre à clé
// de rôle vide (comptée + loggée côté builder, jamais silencieuse) et ses tirs ne gonflent
// aucun rôle.
func TestBuildSquadWeaponAccuracy_ExcludesEmptyRole(t *testing.T) {
	svc := &TeammatesService{
		titleSlug: "halo_5",
		weaponAccuracyRepo: &fakeWeaponAccuracyRepo{
			rows: []port.WeaponAccuracyRow{
				{XUID: "x_main", WeaponID: 99, ShotsFired: 50, ShotsLanded: 25, Class: "shoulder", Role: ""},
				{XUID: "x_main", WeaponID: 10, ShotsFired: 100, ShotsLanded: 40, Class: "shoulder", Role: "precision"},
			},
		},
	}
	got := svc.buildSquadWeaponAccuracy(context.Background(),
		[]domain.SquadMatchRow{{MatchID: "m1"}}, "main", "x_main",
		[]domain.TeammateRow{teammateWithXUID("friend1", "x_f1")})
	if got == nil {
		t.Fatal("want non-nil (precision valide)")
	}
	if len(got.Bars) != 1 || got.Bars[0].Role != "precision" {
		t.Errorf("want [precision] uniquement (rôle vide ignoré), got %+v", got.Bars)
	}
	if got.Bars[0].ShotsFiredByPlayer["main"] != 100 {
		t.Errorf("l'arme sans rôle ne doit pas gonfler les tirs precision: %+v", got.Bars[0].ShotsFiredByPlayer)
	}
}

// TestSquadScopeCentralized est le garde-rail de la factorisation resolveSquadScope
// (règle CLAUDE.md ≤2 copies) : la dérivation du périmètre par-arme (matchs partagés +
// xuids alignés sur les gamertags ordonnés via xuidByPlayer) ne doit exister qu'UNE fois,
// dans resolveSquadScope. buildSquadWeaponKills / buildSquadKillMechanics /
// buildSquadWeaponAccuracy passent tous par le helper. Le littéral distinctif ci-dessous
// n'existe QUE dans le scope par-arme (buildSquadFirstEvents a sa propre dérivation
// partielle, sans slice xuids) → toute réintroduction inline le fait réapparaître.
func TestSquadScopeCentralized(t *testing.T) {
	const sentinel = "xuids = append(xuids, xuidByPlayer[p])"
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du répertoire package: %v", err)
	}
	count := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("lecture %s: %v", name, err)
		}
		count += strings.Count(string(data), sentinel)
	}
	if count != 1 {
		t.Errorf("dérivation du scope squad par-arme dupliquée : littéral %q trouvé %d fois "+
			"(attendu 1, uniquement dans resolveSquadScope). Passer par resolveSquadScope.", sentinel, count)
	}
}
