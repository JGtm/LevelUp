//go:build integration

// Package duckdb — weapon_range_repo_test.go : tests WeaponRangeRepo (lot 3 du plan
// .ai/PLAN_DUELS_PORTEE_2026-09-06.md).
//
// Round-trip sur DB `:memory:` montée par les VRAIES migrations shared (fixture partagée
// `newKillSourceTestPlayerDB` — jamais une DDL recopiée : une DDL de test qui diverge de la
// production ne rougit nulle part, c'est le piège le plus cher du dépôt).
//
// Ce que ces tests verrouillent, dans l'ordre d'importance :
//
//  1. LE SIGNE DU DÉNIVELÉ N'EST PAS INVERSÉ PAR LE REPO, y compris côté victime. C'est
//     l'erreur la plus facile et la moins détectable de tout le lot : `analysis` inverse
//     déjà pour le côté victime, une seconde inversion ici annulerait la première et le
//     produit répondrait « d'en haut » quand la vérité est « d'en bas ».
//  2. les deux côtés sont lus, et l'arme d'une mort est celle DU TUEUR ;
//  3. la garde d'unanimité et les positions partielles/absentes excluent proprement ;
//  4. table de positions absente -> games.ErrCapabilityNotSupported ; table VIDE -> zéro
//     ligne SANS erreur (deux états distincts, jamais confondus) ;
//  5. un filtre trop large est refusé avant toute requête (jamais de scan complet).
//
// Lancer avec : go test -tags=integration -run WeaponRange ./internal/platform/duckdb/ -v
package duckdb

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// wrJoueur / wrAdverse : les deux protagonistes. wrJoueur est CELUI QU'ON INTERROGE — il
// frague une fois et meurt une fois, pour que les deux côtés se lisent d'un seul appel.
const (
	wrJoueur   = "xuid(2533274000000061)"
	wrAdverse  = "xuid(2533274000000062)"
	wrGamertag = "PorteeTest"
)

// insertKillOpening pose une ligne kill_openings brute.
//
// SŒUR d'insertKillPos (kill_distance_repo_test.go) : deux tables, deux INSERT. Les
// factoriser derrière un paramètre de table rouvrirait exactement la confusion que les deux
// types `persist` ferment (une passe d'entames écrite dans kill_positions serait des
// positions fausses de 1,5 s présentées comme celles du coup fatal).
func insertKillOpening(t *testing.T, pdb *PlayerDB, matchID, killerXUID string, timeMS int,
	kx, ky, kz, vx, vy, vz any,
) {
	t.Helper()
	_, err := pdb.Shared.Exec(context.Background(), `
		INSERT INTO kill_openings
			(match_id, killer_xuid, time_ms, killer_x, killer_y, killer_z, victim_x, victim_y, victim_z)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		matchID, killerXUID, timeMS, kx, ky, kz, vx, vy, vz)
	if err != nil {
		t.Fatalf("insert kill_openings: %v", err)
	}
}

// wrFilters : le scope nominal — un match, un joueur par xuid.
func wrFilters() port.WeaponRangeFilters {
	return port.WeaponRangeFilters{MatchIDs: []string{kscMatchID}, XUIDs: []string{wrJoueur}}
}

// seedDeuxCotes pose LES DEUX morts du scénario nominal :
//
//	t=1000 — wrJoueur tue wrAdverse au BR75. Tueur en (0,0,2), victime en (0,0,-1) :
//	         distance 3 m, dénivelé BRUT killer_z - victim_z = +3.
//	t=2000 — wrAdverse tue wrJoueur au répulseur. Tueur en (0,0,0), victime en (0,4,3) :
//	         distance 5 m, dénivelé BRUT = -3 (le tueur était EN DESSOUS).
func seedDeuxCotes(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	insertKillEvent(t, pdb, killEventFixture{pass: kscDecodeV1, publishable: true,
		killerXUID: wrJoueur, victimXUID: wrAdverse, tag: kscTagRifle, timeMS: 1000})
	insertKillPos(t, pdb, kscMatchID, wrJoueur, 1000, 0.0, 0.0, 2.0, 0.0, 0.0, -1.0)

	insertKillEvent(t, pdb, killEventFixture{pass: kscDecodeV1, publishable: true,
		killerXUID: wrAdverse, victimXUID: wrJoueur, tag: kscTagRepulsor, timeMS: 2000})
	insertKillPos(t, pdb, kscMatchID, wrAdverse, 2000, 0.0, 0.0, 0.0, 0.0, 4.0, 3.0)
}

func loadRange(t *testing.T, pdb *PlayerDB, f port.WeaponRangeFilters) []analysis.MeasuredKill {
	t.Helper()
	rows, err := NewWeaponRangeRepo(pdb, fakeKillSourceClassifier{}).
		LoadWeaponRange(context.Background(), pdb.TitleSlug, f)
	if err != nil {
		t.Fatalf("LoadWeaponRange: %v", err)
	}
	return rows
}

func findMeasured(rows []analysis.MeasuredKill, side analysis.Side) *analysis.MeasuredKill {
	for i := range rows {
		if rows[i].Side == side {
			return &rows[i]
		}
	}
	return nil
}

// TestWeaponRange_DeuxCotes_ArmeDistanceEtDeniveleBrut : LE test du lot.
func TestWeaponRange_DeuxCotes_ArmeDistanceEtDeniveleBrut(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	seedDeuxCotes(t, pdb)

	rows := loadRange(t, pdb, wrFilters())
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2 (un frag + une mort) : %+v", len(rows), rows)
	}

	frag := findMeasured(rows, analysis.SideKiller)
	if frag == nil {
		t.Fatalf("aucune ligne côté tueur : %+v", rows)
	}
	if frag.WeaponKey != "hinf_br75" {
		t.Errorf("frag WeaponKey = %q, want hinf_br75", frag.WeaponKey)
	}
	if !almostEqual(frag.DistanceM, 3.0) {
		t.Errorf("frag DistanceM = %v, want 3.0", frag.DistanceM)
	}
	if !almostEqual(frag.DeltaZ, 3.0) {
		t.Errorf("frag DeltaZ = %v, want +3.0 (killer_z - victim_z)", frag.DeltaZ)
	}
	if frag.MatchID != kscMatchID || frag.KillerXUID != wrJoueur || frag.TimeMS != 1000 {
		t.Errorf("clé du frag = (%q,%q,%d), want (%q,%q,1000)",
			frag.MatchID, frag.KillerXUID, frag.TimeMS, kscMatchID, wrJoueur)
	}

	mort := findMeasured(rows, analysis.SideVictim)
	if mort == nil {
		t.Fatalf("aucune ligne côté victime : %+v", rows)
	}
	// L'arme d'une mort est celle DU TUEUR, jamais celle que la victime tenait.
	if mort.WeaponKey != "hinf_repulsor" {
		t.Errorf("mort WeaponKey = %q, want hinf_repulsor (l'arme du tueur)", mort.WeaponKey)
	}
	if !almostEqual(mort.DistanceM, 5.0) {
		t.Errorf("mort DistanceM = %v, want 5.0", mort.DistanceM)
	}
	// LE POINT CRITIQUE : le repo rend la grandeur PHYSIQUE, jamais le point de vue de la
	// victime. C'est analysis.WeaponRangeAggregate qui inverse ; inverser ici aussi
	// annulerait l'inversion et le produit dirait l'exact contraire de la vérité.
	if !almostEqual(mort.DeltaZ, -3.0) {
		t.Errorf("mort DeltaZ = %v, want -3.0 (killer_z - victim_z BRUT, JAMAIS inversé ici)",
			mort.DeltaZ)
	}
	// La clé du frag reste celle du TUEUR, des deux côtés : c'est la clé de la mort, pas
	// celle du point de vue.
	if mort.KillerXUID != wrAdverse || mort.TimeMS != 2000 {
		t.Errorf("clé de la mort = (%q,%d), want (%q,2000)", mort.KillerXUID, mort.TimeMS, wrAdverse)
	}
}

// TestWeaponRange_ResolutionParGamertag : le scope peut désigner le joueur par son gamertag,
// résolu via xuid_aliases — même chemin que tous les lecteurs du paquet.
func TestWeaponRange_ResolutionParGamertag(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	seedDeuxCotes(t, pdb)
	if _, err := pdb.Shared.Exec(context.Background(),
		`INSERT INTO xuid_aliases (xuid, gamertag) VALUES (?, ?)`, wrJoueur, wrGamertag); err != nil {
		t.Fatalf("seed xuid_aliases: %v", err)
	}

	rows := loadRange(t, pdb, port.WeaponRangeFilters{
		MatchIDs: []string{kscMatchID}, Gamertag: wrGamertag,
	})
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2 (gamertag non résolu en xuid ?) : %+v", len(rows), rows)
	}
}

// TestWeaponRange_UnanimiteViolee_Exclue : deux morts au même (tueur, instant) qui ne
// s'accordent pas sur l'arme ne publient RIEN côté tueur — accrocher une position à la
// mauvaise arme serait indétectable à l'écran.
func TestWeaponRange_UnanimiteViolee_Exclue(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKillEvent(t, pdb, killEventFixture{pass: kscDecodeV1, publishable: true,
		killerXUID: wrJoueur, victimXUID: wrAdverse, tag: kscTagRifle, timeMS: 4000})
	insertKillEvent(t, pdb, killEventFixture{pass: kscDecodeV1, publishable: true,
		killerXUID: wrJoueur, victimXUID: "xuid(2533274000000063)", tag: kscTagRepulsor, timeMS: 4000})
	insertKillPos(t, pdb, kscMatchID, wrJoueur, 4000, 0.0, 0.0, 0.0, 3.0, 0.0, 0.0)

	if rows := loadRange(t, pdb, wrFilters()); len(rows) != 0 {
		t.Errorf("double kill ambigu publié à tort : %+v", rows)
	}
}

// TestWeaponRange_DoubleFragMemeArme_ExcluDesDeuxCotes — LE constat C1 de la revue
// adversariale du 2026-09-06.
//
// Deux morts au MÊME (match, tueur, instant) et à la MÊME arme : l'unanimité de `source_tag`
// est vérifiée, mais la ligne de positions — clée par ce seul triplet — ne dit pas LAQUELLE
// des deux victimes elle place. Une distance en sortirait, plausible et potentiellement
// fausse d'un joueur entier.
//
// ET LE FILTRE DE CÔTÉ PASSAIT AUTOUR DE LA GARDE : `WHERE e.victim_xuid = ?` s'applique AVANT
// le `GROUP BY`, donc la lecture côté victime ne voyait qu'UNE ligne et jugeait l'unanimité
// d'un singleton — toujours vérifiée. La garde vit désormais dans la sous-requête `fragSolo`,
// qui compte le groupe sur la vue ENTIÈRE. Le test lit les DEUX côtés : côté victime, il
// rougirait sur la version d'avant.
func TestWeaponRange_DoubleFragMemeArme_ExcluDesDeuxCotes(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	// wrAdverse tue wrJoueur ET un tiers, au même instant, à la même arme.
	insertKillEvent(t, pdb, killEventFixture{pass: kscDecodeV1, publishable: true,
		killerXUID: wrAdverse, victimXUID: wrJoueur, tag: kscTagRifle, timeMS: 6000})
	insertKillEvent(t, pdb, killEventFixture{pass: kscDecodeV1, publishable: true,
		killerXUID: wrAdverse, victimXUID: "xuid(2533274000000065)", tag: kscTagRifle, timeMS: 6000})
	insertKillPos(t, pdb, kscMatchID, wrAdverse, 6000, 0.0, 0.0, 0.0, 3.0, 0.0, 0.0)
	// Et le symétrique, pour que le côté TUEUR soit couvert par le même test.
	insertKillEvent(t, pdb, killEventFixture{pass: kscDecodeV1, publishable: true,
		killerXUID: wrJoueur, victimXUID: wrAdverse, tag: kscTagRifle, timeMS: 7000})
	insertKillEvent(t, pdb, killEventFixture{pass: kscDecodeV1, publishable: true,
		killerXUID: wrJoueur, victimXUID: "xuid(2533274000000066)", tag: kscTagRifle, timeMS: 7000})
	insertKillPos(t, pdb, kscMatchID, wrJoueur, 7000, 0.0, 0.0, 0.0, 4.0, 0.0, 0.0)

	if rows := loadRange(t, pdb, wrFilters()); len(rows) != 0 {
		t.Errorf("double frag publié à tort (la victime placée est indéterminée) : %+v", rows)
	}
}

// TestWeaponRange_PositionAbsenteOuPartielle_Exclue : une mort sans ligne de positions, et
// une ligne dont un seul côté est localisé, sont écartées — jamais approchées.
func TestWeaponRange_PositionAbsenteOuPartielle_Exclue(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	// (a) aucune position pour ce kill.
	insertKillEvent(t, pdb, killEventFixture{pass: kscDecodeV1, publishable: true,
		killerXUID: wrJoueur, victimXUID: wrAdverse, tag: kscTagRifle, timeMS: 1000})
	// (b) position PARTIELLE (victime inconnue) pour celui-ci.
	insertKillEvent(t, pdb, killEventFixture{pass: kscDecodeV1, publishable: true,
		killerXUID: wrJoueur, victimXUID: wrAdverse, tag: kscTagRifle, timeMS: 2000})
	insertKillPos(t, pdb, kscMatchID, wrJoueur, 2000, 0.0, 0.0, 0.0, nil, nil, nil)

	if rows := loadRange(t, pdb, wrFilters()); len(rows) != 0 {
		t.Errorf("position absente ou partielle comptée à tort : %+v", rows)
	}
}

// TestWeaponRange_NonPublishable_Exclue : cette lecture est PAR KILL (arme ET distance
// nommées) — une passe non publiable est juste en agrégat et fausse individuellement.
func TestWeaponRange_NonPublishable_Exclue(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKillEvent(t, pdb, killEventFixture{pass: kscDecodeV1, publishable: false,
		killerXUID: wrJoueur, victimXUID: wrAdverse, tag: kscTagRifle, timeMS: 1000})
	insertKillPos(t, pdb, kscMatchID, wrJoueur, 1000, 0.0, 0.0, 0.0, 3.0, 0.0, 0.0)

	if rows := loadRange(t, pdb, wrFilters()); len(rows) != 0 {
		t.Errorf("passe non publiable comptée à tort : %+v", rows)
	}
}

// TestWeaponRange_HorsScope_NonLu : un autre joueur du même match n'entre pas dans le scope.
func TestWeaponRange_HorsScope_NonLu(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	seedDeuxCotes(t, pdb)
	// Une troisième mort, entre deux tiers : jamais lue pour wrJoueur.
	autre := "xuid(2533274000000064)"
	insertKillEvent(t, pdb, killEventFixture{pass: kscDecodeV1, publishable: true,
		killerXUID: autre, victimXUID: wrAdverse, tag: kscTagRifle, timeMS: 3000})
	insertKillPos(t, pdb, kscMatchID, autre, 3000, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0)

	rows := loadRange(t, pdb, wrFilters())
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2 (une mort hors scope a fuité) : %+v", len(rows), rows)
	}
}

// TestWeaponOpening_Nominal : l'entame se lit avec la MÊME jointure, sur l'autre table, et
// la ligne porte le time_ms DU KILL (sinon rien ne se joindrait).
func TestWeaponOpening_Nominal(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	seedDeuxCotes(t, pdb)
	// Entame du frag de wrJoueur : les deux joueurs étaient à 10 m, au même niveau.
	insertKillOpening(t, pdb, kscMatchID, wrJoueur, 1000, 0.0, 0.0, 0.0, 10.0, 0.0, 0.0)

	rows, err := NewWeaponRangeRepo(pdb, fakeKillSourceClassifier{}).
		LoadWeaponOpening(context.Background(), pdb.TitleSlug, wrFilters())
	if err != nil {
		t.Fatalf("LoadWeaponOpening: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1 (seul le frag a une entame) : %+v", len(rows), rows)
	}
	if rows[0].Side != analysis.SideKiller || rows[0].WeaponKey != "hinf_br75" {
		t.Errorf("entame = %+v, want côté tueur / hinf_br75", rows[0])
	}
	if !almostEqual(rows[0].DistanceM, 10.0) || !almostEqual(rows[0].DeltaZ, 0.0) {
		t.Errorf("entame distance/dénivelé = %v/%v, want 10.0/0.0", rows[0].DistanceM, rows[0].DeltaZ)
	}
	if rows[0].TimeMS != 1000 {
		t.Errorf("entame TimeMS = %d, want 1000 (l'instant DU KILL, pas l'instant mesuré)", rows[0].TimeMS)
	}
}

// TestWeaponOpening_TableVide_ZeroLigneZeroErreur : la table existe et ne porte rien (le
// backfill n'a pas tourné) — état NOMINAL, jamais une panne. C'est ce qui permet à la
// section de dire « N frags mesurés » au lieu d'un zéro trompeur.
func TestWeaponOpening_TableVide_ZeroLigneZeroErreur(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	seedDeuxCotes(t, pdb)

	rows, err := NewWeaponRangeRepo(pdb, fakeKillSourceClassifier{}).
		LoadWeaponOpening(context.Background(), pdb.TitleSlug, wrFilters())
	if err != nil {
		t.Fatalf("LoadWeaponOpening sur table vide: err = %v, want nil", err)
	}
	if len(rows) != 0 {
		t.Errorf("rows = %+v, want aucune", rows)
	}
}

// TestWeaponOpening_TableAbsente_CapabilityNotSupported : une base sans la vue d'entames
// (titre sans décodeur de film, base non migrée) dégrade en capability absente — pas en
// erreur technique remontée à l'utilisateur.
func TestWeaponOpening_TableAbsente_CapabilityNotSupported(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	seedDeuxCotes(t, pdb)
	if _, err := pdb.Shared.Exec(context.Background(), `DROP VIEW kill_openings_latest`); err != nil {
		t.Fatalf("drop vue: %v", err)
	}

	_, err := NewWeaponRangeRepo(pdb, fakeKillSourceClassifier{}).
		LoadWeaponOpening(context.Background(), pdb.TitleSlug, wrFilters())
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Fatalf("err = %v, want games.ErrCapabilityNotSupported", err)
	}
}

// TestWeaponRange_FiltreTropLarge_Refuse : jamais de scan complet. Les deux formes de refus
// (aucun match, aucun joueur) sont vérifiées — une seule des deux laisserait passer l'autre.
func TestWeaponRange_FiltreTropLarge_Refuse(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	repo := NewWeaponRangeRepo(pdb, fakeKillSourceClassifier{})
	cas := map[string]port.WeaponRangeFilters{
		"aucun match":  {XUIDs: []string{wrJoueur}},
		"aucun joueur": {MatchIDs: []string{kscMatchID}},
	}
	for nom, f := range cas {
		if _, err := repo.LoadWeaponRange(context.Background(), pdb.TitleSlug, f); !errors.Is(err, port.ErrWeaponRangeFiltersTooBroad) {
			t.Errorf("%s: err = %v, want ErrWeaponRangeFiltersTooBroad", nom, err)
		}
		if _, err := repo.LoadWeaponOpening(context.Background(), pdb.TitleSlug, f); !errors.Is(err, port.ErrWeaponRangeFiltersTooBroad) {
			t.Errorf("%s (entame): err = %v, want ErrWeaponRangeFiltersTooBroad", nom, err)
		}
	}
}

// TestWeaponRange_ClassifierNil_RienCharge : un titre sans classificateur ne peut traduire
// aucune source de dégât — zéro ligne, sans erreur, même avec des données présentes.
func TestWeaponRange_ClassifierNil_RienCharge(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	seedDeuxCotes(t, pdb)

	rows, err := NewWeaponRangeRepo(pdb, nil).
		LoadWeaponRange(context.Background(), pdb.TitleSlug, wrFilters())
	if err != nil || rows != nil {
		t.Errorf("classifier nil: rows=%+v err=%v, want nil/nil", rows, err)
	}
}

// TestWeaponRange_SourceHorsRegistre_Ecartee : une source que le classificateur ne connaît
// pas est écartée, jamais devinée — et la mort ne remonte pas sous une arme approchée.
func TestWeaponRange_SourceHorsRegistre_Ecartee(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKillEvent(t, pdb, killEventFixture{pass: kscDecodeV1, publishable: true,
		killerXUID: wrJoueur, victimXUID: wrAdverse, tag: kscTagInconnu, timeMS: 1000})
	insertKillPos(t, pdb, kscMatchID, wrJoueur, 1000, 0.0, 0.0, 0.0, 3.0, 0.0, 0.0)

	if rows := loadRange(t, pdb, wrFilters()); len(rows) != 0 {
		t.Errorf("source hors registre publiée à tort : %+v", rows)
	}
}
