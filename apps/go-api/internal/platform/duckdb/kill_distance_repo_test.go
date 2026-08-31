//go:build integration

// Package duckdb — kill_distance_repo_test.go : tests KillDistanceRepo.
//
// Round-trip sur DB :memory: avec les VRAIES migrations shared (dont les vues
// match_kill_events_latest et kill_positions_latest) et le VRAI registre
// d'armes — même harnais que killsource_class_repo_test.go
// (newKillSourceTestPlayerDB, fakeKillSourceClassifier, insertKill : réutilisés
// tels quels, même package).
//
// Ce que ces tests verrouillent, dans l'ordre d'importance :
//
//  1. distance moyenne EXACTE par (xuid, weapon_key), sur des coordonnées
//     choisies pour que le calcul se vérifie de tête ;
//  2. une mort sans position (aucune ligne kill_positions, ou une ligne
//     PARTIELLE — un seul côté connu) est EXCLUE proprement, jamais approchée ;
//  3. un double kill au même (tueur, instant) dont les deux lignes ne
//     s'accordent pas sur l'arme est EXCLU en entier (garde d'unanimité,
//     même doctrine que Q21b) ;
//  4. une passe NON PUBLIABLE est EXCLUE (contrairement à KillSourceClassRepo :
//     cette lecture est PAR KILL, pas un agrégat qui tolère l'individuel faux) ;
//  5. une clé D'ARSENAL (qui porte un id numérique, hinf_br75) REMONTE — LA
//     différence de comportement avec KillSourceClassRepo, qui la filtre ;
//  6. classificateur nil / table vide / matchID vide dégradent proprement.
//
// Lancer avec : go test -tags=integration -run KillDistance ./internal/platform/duckdb/ -v
package duckdb

import (
	"context"
	"math"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// kdXUIDA / kdXUIDB : xuids de test. Le match_id, lui, réutilise kscMatchID
// (killsource_class_repo_test.go, même package) : insertKill (réutilisé tel
// quel ci-dessous) l'a EN DUR dans son propre INSERT — un match_id différent
// ici ferait deux voisines qui ne se joignent jamais (kill_positions vs
// match_kill_events sur deux match_id distincts).
const (
	kdXUIDA = "xuid(2533274000000051)"
	kdXUIDB = "xuid(2533274000000052)"
)

// insertKillPos pose une ligne kill_positions brute. Les coordonnées acceptent
// `nil` (position partielle — cf. TestKillDistance_PositionPartielle_Exclue) :
// signature `any` pour ça, pas `*float64`, afin de rester un simple passe-plat
// vers le driver SQL (même convention que insertKill pour killerXUID).
func insertKillPos(t *testing.T, pdb *PlayerDB, matchID, killerXUID string, timeMS int,
	kx, ky, kz, vx, vy, vz any,
) {
	t.Helper()
	_, err := pdb.Shared.Exec(context.Background(), `
		INSERT INTO kill_positions
			(match_id, killer_xuid, time_ms, killer_x, killer_y, killer_z, victim_x, victim_y, victim_z)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		matchID, killerXUID, timeMS, kx, ky, kz, vx, vy, vz)
	if err != nil {
		t.Fatalf("insert kill_positions: %v", err)
	}
}

func loadKD(t *testing.T, pdb *PlayerDB, classifier port.KillSourceClassifier) []domain.MatchKillDistancePlayer {
	t.Helper()
	repo := NewKillDistanceRepo(pdb, classifier)
	rows, err := repo.LoadMatch(context.Background(), kscMatchID)
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	return rows
}

func findKDPlayer(rows []domain.MatchKillDistancePlayer, xuid string) *domain.MatchKillDistancePlayer {
	for i := range rows {
		if rows[i].XUID == xuid {
			return &rows[i]
		}
	}
	return nil
}

func findKDWeapon(p *domain.MatchKillDistancePlayer, weaponKey string) *domain.MatchKillDistanceWeapon {
	if p == nil {
		return nil
	}
	for i := range p.Weapons {
		if p.Weapons[i].WeaponKey == weaponKey {
			return &p.Weapons[i]
		}
	}
	return nil
}

func almostEqual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// TestKillDistance_MoyennesExactes : LE test du lot — deux joueurs, deux armes,
// distances choisies pour un calcul de tête. A/hinf_br75 : 3 m puis 5 m (avg 4,
// min 3, max 5). B/hinf_repulsor : 10 m (un seul kill, avg = valeur unique).
func TestKillDistance_MoyennesExactes(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)

	insertKill(t, pdb, kscDecodeV1, true, kdXUIDA, kscTagRifle, 1000)
	insertKillPos(t, pdb, kscMatchID, kdXUIDA, 1000, 0.0, 0.0, 0.0, 3.0, 0.0, 0.0) // distance 3

	insertKill(t, pdb, kscDecodeV1, true, kdXUIDA, kscTagRifle, 2000)
	insertKillPos(t, pdb, kscMatchID, kdXUIDA, 2000, 0.0, 0.0, 0.0, 0.0, 5.0, 0.0) // distance 5

	insertKill(t, pdb, kscDecodeV1, true, kdXUIDB, kscTagRepulsor, 1000)
	insertKillPos(t, pdb, kscMatchID, kdXUIDB, 1000, 0.0, 0.0, 0.0, 0.0, 0.0, 10.0) // distance 10

	rows := loadKD(t, pdb, fakeKillSourceClassifier{})

	a := findKDWeapon(findKDPlayer(rows, kdXUIDA), "hinf_br75")
	if a == nil {
		t.Fatalf("hinf_br75 absent pour A, got %+v", rows)
	}
	if a.MeasuredKills != 2 {
		t.Errorf("A MeasuredKills = %d, want 2", a.MeasuredKills)
	}
	if !almostEqual(a.AvgDistanceM, 4.0) {
		t.Errorf("A AvgDistanceM = %v, want 4.0", a.AvgDistanceM)
	}
	if !almostEqual(a.MinDistanceM, 3.0) || !almostEqual(a.MaxDistanceM, 5.0) {
		t.Errorf("A Min/Max = %v/%v, want 3.0/5.0", a.MinDistanceM, a.MaxDistanceM)
	}

	b := findKDWeapon(findKDPlayer(rows, kdXUIDB), "hinf_repulsor")
	if b == nil {
		t.Fatalf("hinf_repulsor absent pour B, got %+v", rows)
	}
	if b.MeasuredKills != 1 || !almostEqual(b.AvgDistanceM, 10.0) {
		t.Errorf("B = %d kills / %v m, want 1 / 10.0", b.MeasuredKills, b.AvgDistanceM)
	}
}

// TestKillDistance_CleArsenalRemonte : hinf_br75 PORTE un id numérique
// (weapon_ids, registre réel) — KillSourceClassRepo l'écarterait
// (anti-double-comptage weapon_kills). Ce repo-ci n'a pas cette raison
// d'écarter : il doit la publier, avec son libellé.
func TestKillDistance_CleArsenalRemonte(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Metadata.Exec(ctx, `CREATE TABLE weapon_name_labels (
		title_slug VARCHAR, weapon_key VARCHAR, name_en VARCHAR, name_fr VARCHAR,
		PRIMARY KEY (title_slug, weapon_key))`); err != nil {
		t.Fatalf("create weapon_name_labels: %v", err)
	}
	if _, err := pdb.Metadata.Exec(ctx,
		"INSERT INTO weapon_name_labels VALUES ('halo_infinite', 'hinf_br75', 'BR75', 'BR75')"); err != nil {
		t.Fatalf("seed weapon_name_labels: %v", err)
	}

	insertKill(t, pdb, kscDecodeV1, true, kdXUIDA, kscTagRifle, 1000)
	insertKillPos(t, pdb, kscMatchID, kdXUIDA, 1000, 0.0, 0.0, 0.0, 7.0, 0.0, 0.0)

	rows := loadKD(t, pdb, fakeKillSourceClassifier{})
	w := findKDWeapon(findKDPlayer(rows, kdXUIDA), "hinf_br75")
	if w == nil {
		t.Fatalf("hinf_br75 absent (devrait remonter, contrairement à KillSourceClassRepo) : %+v", rows)
	}
	if w.Label != "BR75" || w.LabelEN != "BR75" {
		t.Errorf("Label/LabelEN = %q/%q, want BR75/BR75", w.Label, w.LabelEN)
	}
}

// TestKillDistance_SansPosition_Exclue : un kill_event sans AUCUNE ligne
// kill_positions correspondante n'est jamais compté.
func TestKillDistance_SansPosition_Exclue(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, true, kdXUIDA, kscTagRifle, 1000)
	insertKillPos(t, pdb, kscMatchID, kdXUIDA, 1000, 0.0, 0.0, 0.0, 3.0, 0.0, 0.0)
	// Deuxième kill, AUCUNE position.
	insertKill(t, pdb, kscDecodeV1, true, kdXUIDA, kscTagRifle, 2000)

	rows := loadKD(t, pdb, fakeKillSourceClassifier{})
	w := findKDWeapon(findKDPlayer(rows, kdXUIDA), "hinf_br75")
	if w == nil || w.MeasuredKills != 1 {
		t.Fatalf("want exactement 1 kill mesuré (le second, sans position, exclu) : %+v", rows)
	}
}

// TestKillDistance_PositionPartielle_Exclue : une ligne kill_positions dont un
// SEUL côté est connu (victime NULL) n'est pas approchée — exclue comme une
// absence totale.
func TestKillDistance_PositionPartielle_Exclue(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, true, kdXUIDA, kscTagRifle, 1000)
	insertKillPos(t, pdb, kscMatchID, kdXUIDA, 1000, 0.0, 0.0, 0.0, nil, nil, nil)

	rows := loadKD(t, pdb, fakeKillSourceClassifier{})
	if w := findKDWeapon(findKDPlayer(rows, kdXUIDA), "hinf_br75"); w != nil {
		t.Errorf("position partielle comptée à tort : %+v", w)
	}
}

// TestKillDistance_DoubleKillArmesDivergentes_Exclu : deux morts au même
// (tueur, instant) qui ne s'accordent pas sur l'arme ne publient RIEN — même
// garde d'unanimité que Q21b (accrocher une position à la mauvaise arme serait
// indétectable à l'écran).
func TestKillDistance_DoubleKillArmesDivergentes_Exclu(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, true, kdXUIDA, kscTagRifle, 4000)
	insertKill(t, pdb, kscDecodeV1, true, kdXUIDA, kscTagRepulsor, 4000)
	insertKillPos(t, pdb, kscMatchID, kdXUIDA, 4000, 0.0, 0.0, 0.0, 3.0, 0.0, 0.0)

	rows := loadKD(t, pdb, fakeKillSourceClassifier{})
	if p := findKDPlayer(rows, kdXUIDA); p != nil && len(p.Weapons) != 0 {
		t.Errorf("double kill ambigu compté à tort : %+v", p)
	}
}

// TestKillDistance_NonPublishable_Exclue : contrairement à KillSourceClassRepo
// (qui compte les passes non publiables, cf. son doctrine), cette lecture est
// PAR KILL — une passe non publiable est justesse en AGRÉGAT et fausse
// individuellement (attribution arme+distance nommée) : exclue.
func TestKillDistance_NonPublishable_Exclue(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, false, kdXUIDA, kscTagRifle, 1000)
	insertKillPos(t, pdb, kscMatchID, kdXUIDA, 1000, 0.0, 0.0, 0.0, 3.0, 0.0, 0.0)

	rows := loadKD(t, pdb, fakeKillSourceClassifier{})
	if w := findKDWeapon(findKDPlayer(rows, kdXUIDA), "hinf_br75"); w != nil {
		t.Errorf("passe non publiable comptée à tort : %+v", w)
	}
}

// TestKillDistance_TableVide_ZeroLigneZeroErreur : match jamais décodé, aucune
// ligne dans les deux tables — état NOMINAL, pas une panne.
func TestKillDistance_TableVide_ZeroLigneZeroErreur(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	repo := NewKillDistanceRepo(pdb, fakeKillSourceClassifier{})
	rows, err := repo.LoadMatch(context.Background(), kscMatchID)
	if err != nil {
		t.Fatalf("LoadMatch sur table vide: err = %v, want nil", err)
	}
	if rows != nil {
		t.Errorf("rows = %+v, want nil", rows)
	}
}

// TestKillDistance_ClassifierNil_RienCharge : même doctrine que
// KillSourceClassRepo — un titre sans classificateur ne peut rien traduire,
// donc rien ne remonte, sans erreur, même avec des données présentes.
func TestKillDistance_ClassifierNil_RienCharge(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, true, kdXUIDA, kscTagRifle, 1000)
	insertKillPos(t, pdb, kscMatchID, kdXUIDA, 1000, 0.0, 0.0, 0.0, 3.0, 0.0, 0.0)

	repo := NewKillDistanceRepo(pdb, nil)
	rows, err := repo.LoadMatch(context.Background(), kscMatchID)
	if err != nil || rows != nil {
		t.Errorf("classifier nil: rows=%+v err=%v, want nil/nil", rows, err)
	}
}

// TestKillDistance_MatchIDVide_Erreur : jamais de scan complet — un matchID
// vide est un refus, pas un balayage de shared.kill_positions entier.
func TestKillDistance_MatchIDVide_Erreur(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	repo := NewKillDistanceRepo(pdb, fakeKillSourceClassifier{})
	if _, err := repo.LoadMatch(context.Background(), ""); err == nil {
		t.Fatal("attendu un refus (matchID vide), obtenu nil")
	}
}
