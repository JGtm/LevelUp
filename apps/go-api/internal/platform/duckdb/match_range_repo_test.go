//go:build integration

// Package duckdb — match_range_repo_test.go : tests du lecteur de portée « tout le lobby »
// (lot N2 du plan .ai/V7.5/PLAN_AJUSTEMENTS_SUPP_PRE_V75_2026-09-21.md, réserve R3).
//
// Round-trip sur DB `:memory:` montée par les VRAIES migrations shared (fixture partagée
// `newKillSourceTestPlayerDB`) — jamais une DDL recopiée, qui diverge sans rougir nulle part.
//
// Ce que ces tests verrouillent, dans l'ordre d'importance :
//
//  1. LA LECTURE RAMÈNE LES FRAGS DE TOUS LES JOUEURS, sans désignant, et chaque frag n'est
//     compté QU'UNE FOIS (côté tueur seulement — le compter des deux côtés doublerait chaque
//     mesure dans la médiane du lobby, silencieusement) ;
//  2. `AllPlayers` sans MatchIDs reste REFUSÉ : la garde anti-scan n'est pas désarmée ;
//  3. le dénominateur de couverture compte les frags publiables, positions ou non ;
//  4. table de positions absente -> games.ErrCapabilityNotSupported (dégradation, pas panne).
//
// Lancer avec : go test -tags=integration -run MatchRange ./internal/platform/duckdb/ -v
package duckdb

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// mrTroisieme : un joueur du lobby que RIEN ne désigne — c'est lui qui prouve que la lecture
// ne dépend d'aucun filtre de joueur.
const mrTroisieme = "xuid(2533274000000063)"

func mrFilters() port.WeaponRangeFilters {
	return port.WeaponRangeFilters{MatchIDs: []string{kscMatchID}, AllPlayers: true}
}

func loadMatchRange(t *testing.T, pdb *PlayerDB) port.MatchRangeRead {
	t.Helper()
	read, err := NewWeaponRangeRepo(pdb, fakeKillSourceClassifier{}).
		LoadMatchRangeKills(context.Background(), pdb.TitleSlug, mrFilters())
	if err != nil {
		t.Fatalf("LoadMatchRangeKills: %v", err)
	}
	return read
}

// TestMatchRange_TousLesJoueursUneSeuleFois : LE test du lot.
//
// Trois frags, trois tueurs différents, dont un (mrTroisieme) qui n'est désigné nulle part.
// Un QUATRIÈME frag est publiable mais SANS positions : il ne peut pas être mesuré, et il
// compte au dénominateur — c'est exactement ce que publie la couverture.
func TestMatchRange_TousLesJoueursUneSeuleFois(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	seedDeuxCotes(t, pdb) // wrJoueur tue à 3 m (t=1000), wrAdverse tue à 5 m (t=2000)
	insertKillEvent(t, pdb, killEventFixture{pass: kscDecodeV1, publishable: true,
		killerXUID: mrTroisieme, victimXUID: wrAdverse, tag: kscTagRifle, timeMS: 3000})
	insertKillPos(t, pdb, kscMatchID, mrTroisieme, 3000, 0.0, 0.0, 0.0, 0.0, 12.0, 0.0)
	// Frag publiable SANS ligne de positions : absent du numérateur, présent au dénominateur.
	insertKillEvent(t, pdb, killEventFixture{pass: kscDecodeV1, publishable: true,
		killerXUID: wrJoueur, victimXUID: mrTroisieme, tag: kscTagRifle, timeMS: 4000})

	read := loadMatchRange(t, pdb)
	if len(read.Kills) != 3 {
		t.Fatalf("frags mesures = %d, want 3 (un par tueur, chacun UNE fois) : %+v",
			len(read.Kills), read.Kills)
	}
	parTueur := map[string]int{}
	for _, k := range read.Kills {
		parTueur[k.KillerXUID]++
		if k.MatchID != kscMatchID {
			t.Errorf("match_id = %q, want %q", k.MatchID, kscMatchID)
		}
	}
	for _, xuid := range []string{wrJoueur, wrAdverse, mrTroisieme} {
		if parTueur[xuid] != 1 {
			t.Errorf("tueur %s : %d frags, want 1 (deux = le cote victime compte double)",
				xuid, parTueur[xuid])
		}
	}
	// Le dénominateur porte les QUATRE frags publiables, celui sans positions compris.
	if read.KillsTotal != 4 {
		t.Errorf("KillsTotal = %d, want 4 (les frags publiables, positions ou non)",
			read.KillsTotal)
	}
}

// TestMatchRange_AllPlayersSansMatchIDsRefuse : la garde anti-scan n'est pas désarmée par
// AllPlayers — elle n'a jamais protégé un joueur, elle protège la table partagée.
func TestMatchRange_AllPlayersSansMatchIDsRefuse(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	_, err := NewWeaponRangeRepo(pdb, fakeKillSourceClassifier{}).LoadMatchRangeKills(
		context.Background(), pdb.TitleSlug, port.WeaponRangeFilters{AllPlayers: true})
	if !errors.Is(err, port.ErrWeaponRangeFiltersTooBroad) {
		t.Fatalf("err = %v, want ErrWeaponRangeFiltersTooBroad", err)
	}
}

// TestMatchRange_TablePositionsAbsente : un titre sans décodeur de film dégrade proprement.
func TestMatchRange_TablePositionsAbsente(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	seedDeuxCotes(t, pdb)
	if _, err := pdb.Shared.Exec(context.Background(), `DROP VIEW kill_positions_latest`); err != nil {
		t.Fatalf("drop view: %v", err)
	}
	_, err := NewWeaponRangeRepo(pdb, fakeKillSourceClassifier{}).
		LoadMatchRangeKills(context.Background(), pdb.TitleSlug, mrFilters())
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Fatalf("err = %v, want games.ErrCapabilityNotSupported", err)
	}
}

// TestMatchRange_ScopeVideSansErreur : un scope dont aucun film n'est décodé rend zéro frag
// SANS erreur — l'état nominal d'une session récente, jamais une panne.
func TestMatchRange_ScopeVideSansErreur(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	read := loadMatchRange(t, pdb)
	if len(read.Kills) != 0 || read.KillsTotal != 0 {
		t.Fatalf("read = %+v, want vide sans erreur", read)
	}
}
