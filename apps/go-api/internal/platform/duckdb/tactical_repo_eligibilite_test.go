package duckdb

// tactical_repo_eligibilite_test.go — LA COLONNE `eligible_cuisson` DE `QTacticalUnivers`,
// SUR UNE VRAIE BASE.
//
// Elle n'etait exercee par AUCUN test (revue de 7.10, P1) : le double du service rendait le
// booleen a la main, si bien que le SQL qui le calcule n'a jamais ete execute. C'est ainsi
// qu'un predicat rendant NULL a pu etre livre.

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/sync/matchflags"
)

// semerMatchEligibilite inscrit un match au registre et y place le joueur.
//
// `debut` NUL pose les DEUX horodatages a NULL — ce que la DDL autorise
// (`steps_shared_core.go`, `start_time` et `start_time_utc` nullables) et ce qui faisait
// echouer le scan avant la correction.
func semerMatchEligibilite(t *testing.T, pdb *PlayerDB, matchID string, debut *time.Time, flags int64) {
	t.Helper()
	ctx := context.Background()
	if debut == nil {
		if _, err := pdb.Shared.Exec(ctx,
			`INSERT INTO match_registry (match_id, map_id, backfill_completed) VALUES (?, ?, ?)`,
			matchID, tacCarteA, flags); err != nil {
			t.Fatalf("seed registre %s: %v", matchID, err)
		}
	} else if _, err := pdb.Shared.Exec(ctx,
		`INSERT INTO match_registry (match_id, map_id, start_time, start_time_utc, backfill_completed)
		 VALUES (?, ?, ?, ?, ?)`, matchID, tacCarteA, *debut, *debut, flags); err != nil {
		t.Fatalf("seed registre %s: %v", matchID, err)
	}
	if _, err := pdb.Shared.Exec(ctx,
		`INSERT INTO match_participants (match_id, xuid, team_id, outcome) VALUES (?, ?, 0, 2)`,
		matchID, tacXUIDMoi); err != nil {
		t.Fatalf("seed participant %s: %v", matchID, err)
	}
}

// TestUnivers_EligibiliteALaCuisson_QuatreCas — LE PREDICAT DE LA FILE, EXECUTE.
//
// Quatre matchs, un par situation, et un seul est eligible :
//
//	recent        dans la fenetre, film pas perdu, datable  -> ELIGIBLE
//	vieux         plus ancien que la fenetre                -> non
//	film_perdu    marqueur terminal `MBitFilmAbsent` pose   -> non
//	indatable     les deux horodatages NULL                 -> non, ET SANS PLANTER
//
// LE QUATRIEME EST LE P0. `COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC') >= ?`
// vaut NULL quand les deux colonnes sont NULL, et le scanner dans un `bool` nu rendait
// `converting NULL to bool` : 500 sur TOUTE la lecture d'artefact de la carte, pour une
// seule ligne de registre mal datee.
func TestUnivers_EligibiliteALaCuisson_QuatreCas(t *testing.T) {
	_ = migration.All()
	pdb := newTacticalTestPlayerDB(t)
	recent := time.Now().UTC().AddDate(0, 0, -3)
	vieux := time.Now().UTC().AddDate(0, -6, 0)

	semerMatchEligibilite(t, pdb, "recent", &recent, 0)
	semerMatchEligibilite(t, pdb, "vieux", &vieux, 0)
	semerMatchEligibilite(t, pdb, "film_perdu", &recent, int64(matchflags.MBitFilmAbsent))
	semerMatchEligibilite(t, pdb, "indatable", nil, 0)

	repo := NewTacticalRepo(pdb)
	q := domain.TacticalQuery{PlayerXUID: tacXUIDMoi, MapID: tacCarteA, RetentionMois: 3}
	univ, err := repo.Univers(context.Background(), q)
	if err != nil {
		t.Fatalf("Univers avec RetentionMois=3 : %v — un horodatage NULL ne doit pas faire "+
			"echouer la lecture", err)
	}
	if len(univ.Matchs) != 4 {
		t.Fatalf("univers = %d matchs, attendu 4 : la retention NE FILTRE RIEN, elle classe",
			len(univ.Matchs))
	}
	attendu := map[string]bool{"recent": true, "vieux": false, "film_perdu": false, "indatable": false}
	eligibles := 0
	for _, m := range univ.Matchs {
		veut, connu := attendu[m.MatchID]
		if !connu {
			t.Fatalf("match inattendu %q", m.MatchID)
		}
		if m.EligibleALaCuisson != veut {
			t.Errorf("%s : eligible = %v, attendu %v", m.MatchID, m.EligibleALaCuisson, veut)
		}
		if m.EligibleALaCuisson {
			eligibles++
		}
	}
	if eligibles != 1 {
		t.Fatalf("eligibles = %d, attendu 1 sur 4 (ventilation 1 / 3)", eligibles)
	}
}

// TestUnivers_EligibiliteALaCuisson_FenetreIllimitee — sans fenetre, l'AGE n'ecarte plus
// personne, mais le film perdu et l'absence d'horodatage restent disqualifiants.
//
// C'est la convention partagee avec la purge et la file : 0 = illimitee, jamais « zero
// mois ». Un match ancien redevient « en attente » — ce qui est vrai, la file le reprendra.
func TestUnivers_EligibiliteALaCuisson_FenetreIllimitee(t *testing.T) {
	_ = migration.All()
	pdb := newTacticalTestPlayerDB(t)
	recent := time.Now().UTC().AddDate(0, 0, -3)
	vieux := time.Now().UTC().AddDate(0, -6, 0)

	semerMatchEligibilite(t, pdb, "recent", &recent, 0)
	semerMatchEligibilite(t, pdb, "vieux", &vieux, 0)
	semerMatchEligibilite(t, pdb, "film_perdu", &recent, int64(matchflags.MBitFilmAbsent))
	semerMatchEligibilite(t, pdb, "indatable", nil, 0)

	repo := NewTacticalRepo(pdb)
	univ, err := repo.Univers(context.Background(),
		domain.TacticalQuery{PlayerXUID: tacXUIDMoi, MapID: tacCarteA, RetentionMois: 0})
	if err != nil {
		t.Fatalf("Univers sans fenetre : %v", err)
	}
	attendu := map[string]bool{"recent": true, "vieux": true, "film_perdu": false, "indatable": false}
	for _, m := range univ.Matchs {
		if m.EligibleALaCuisson != attendu[m.MatchID] {
			t.Errorf("%s : eligible = %v, attendu %v (fenetre illimitee)", m.MatchID,
				m.EligibleALaCuisson, attendu[m.MatchID])
		}
	}
}
