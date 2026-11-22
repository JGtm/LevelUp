package duckdb

// medals_earned_fenetres_test.go — LES LECTURES DE MÉDAILLES BORNÉES À DES MATCHS PAYENT LA
// FENÊTRE DE LEURS MATCHS (ADR 0036 I2).
//
// La vue `v_medals_earned` fusionne `medals_earned` et le résumé d'usage du film (dernière passe de
// chaque match, décidée par la fenêtre de `match_usage_films_latest`). Une lecture bornée à des
// matchs qui la traverse doit pousser ses `match_id` constants (égalité ou liste) jusque sous cette
// fenêtre : sans quoi la page paie la fenêtre de tout l'historique du résumé d'usage pour afficher
// deux matchs, sans qu'aucun nombre ne change. Les lectures de tout l'historique
// (`Q36aMedalTotals`, totaux de carrière) gardent le coût de la fenêtre, ce que l'ADR admet.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

const (
	medailleVengeurFenetres = 9000000001
	medailleCibleFenetres   = 9000000002
	medailleAutreFenetres   = 55
	// generationsUsageFenetres : passes du résumé d'usage par match. Deux, pour que la fenêtre
	// ait quelque chose à départager : un match non borné la ferait voir deux fois plus de lignes.
	generationsUsageFenetres = 2
	// lignesParMatchFenetres : lignes qu'UN match tient dans la vue avant fusion (deux médailles
	// brutes, une valeur de Vengeur dérivée, une de Cible difficile). La fenêtre de fusion de la vue
	// les voit toutes ; la fenêtre du résumé d'usage, elle, voit `generationsUsageFenetres` passes,
	// donc moins. Hors périmètre, cette fenêtre voit la table entière (dix matchs : vingt passes).
	lignesParMatchFenetres = 4
)

// seedMedaillesFenetres pose, pour chacun des matchs, un Vengeur hérité, une autre médaille, et
// `generationsUsageFenetres` passes du résumé d'usage (une ligne film, une ligne joueur par
// passe : la dernière porte deux vengeances et trois recharges de bouclier).
func seedMedaillesFenetres(t *testing.T, b baseNotee, matchs []string) {
	t.Helper()
	for _, m := range matchs {
		if _, err := b.brute.Exec(`INSERT INTO medals_earned (match_id, xuid, medal_name_id, count) VALUES
			(?, ?, ?, 7), (?, ?, ?, 1)`,
			m, tacXUIDMoi, medailleVengeurFenetres, m, tacXUIDMoi, medailleAutreFenetres); err != nil {
			t.Fatalf("medals_earned %s: %v", m, err)
		}
		for g := 1; g <= generationsUsageFenetres; g++ {
			pass := "p" + string(rune('0'+g))
			if _, err := b.brute.Exec(`INSERT INTO match_usage_films
				(match_id, summary_pass, summary_rev, artifact_schema, frame_interval_ms, frame_count, duration_ms,
				 pad_occupancies, pad_named, pad_unnamed, weapon_pads_json, powerup_pickups_json)
				VALUES (?, ?, 'us7', 70, 100, 10, 1000, 0, 0, 0, '[]', '{}')`, m, pass); err != nil {
				t.Fatalf("film %s/%s: %v", m, pass, err)
			}
			if _, err := b.brute.Exec(`INSERT INTO match_usage_players
				(match_id, summary_pass, summary_rev, xuid, grapple_pulls, camo_episodes, camo_ms, camo_kills,
				 overshield_episodes, overshield_ms, overshield_kills, deployed_json, dropped_objects, grenades_thrown,
				 pad_pickups, pad_pickups_json, revenge_kills, shield_recharges)
				VALUES (?, ?, 'us7', ?, 0, 0, 0, 0, 0, 0, 0, '{}', 0, 0, 0, '{}', ?, ?)`,
				m, pass, tacXUIDMoi, g, g+1); err != nil {
				t.Fatalf("joueur %s/%s: %v", m, pass, err)
			}
		}
	}
}

// libellesDegrades : la lecture a servi ses médailles et dit seulement que leurs libellés n'ont pas
// pu être résolus (la base de métadonnées du corpus ne porte pas les définitions de médailles).
func libellesDegrades(err error) bool {
	var degraded *domain.SectionDegradedError
	return errors.As(err, &degraded)
}

// TestMedailles_LecturesBornees_FenetresBornees : dix matchs, deux demandés ; chaque fenêtre de la
// vue des médailles ne voit que les passes des matchs demandés, et les médailles servies sont les
// dérivées (Vengeur de la dernière passe à la place de l'hérité, Cible difficile).
func TestMedailles_LecturesBornees_FenetresBornees(t *testing.T) {
	b := newBaseNotee(t)
	ids := seedFenetresTactiques(t, b.pdb, 10)
	seedMedaillesFenetres(t, b, ids)
	ctx := context.Background()
	deux := ids[:2]
	b.carnet.vider()

	// Dernière passe : Vengeur 2, Cible difficile 3 ; l'autre médaille reste.
	attendu := map[int64]int{medailleVengeurFenetres: 2, medailleCibleFenetres: 3, medailleAutreFenetres: 1}
	exigerMedailles := func(nom string, got map[int64]int) {
		t.Helper()
		if len(got) != len(attendu) {
			t.Fatalf("%s : %v, attendu %v", nom, got, attendu)
		}
		for id, n := range attendu {
			if got[id] != n {
				t.Fatalf("%s : médaille %d = %d, attendu %d (%v)", nom, id, got[id], n, got)
			}
		}
	}

	t.Run("Q14 un match", func(t *testing.T) {
		got, err := NewMatchViewRepo(b.pdb, tacXUIDMoi).GetMatchMedals(ctx, tacXUIDMoi, ids[0])
		if err != nil && !libellesDegrades(err) {
			t.Fatalf("GetMatchMedals: %v", err)
		}
		parID := map[int64]int{}
		for _, m := range got {
			parID[m.MedalID] += m.Count
		}
		exigerMedailles("GetMatchMedals", parID)
		exigerFenetresBornees(t, b, "GetMatchMedals (Q14)", lignesParMatchFenetres, 1)
	})

	t.Run("Q27 un match", func(t *testing.T) {
		got, err := NewMatchViewRepo(b.pdb, tacXUIDMoi).GetMatchBulkMedals(ctx, ids[0])
		if err != nil && !libellesDegrades(err) {
			t.Fatalf("GetMatchBulkMedals: %v", err)
		}
		parID := map[int64]int{}
		for _, m := range got {
			parID[m.MedalID] += m.Count
		}
		exigerMedailles("GetMatchBulkMedals", parID)
		exigerFenetresBornees(t, b, "GetMatchBulkMedals (Q27)", lignesParMatchFenetres, 1)
	})

	t.Run("Q26h Accueil", func(t *testing.T) {
		got, err := NewHomeRepo(b.pdb).LoadMatchMedals(ctx, deux)
		if err != nil {
			t.Fatalf("LoadMatchMedals: %v", err)
		}
		if len(got) != len(deux) {
			t.Fatalf("LoadMatchMedals : %d matchs, attendu %d", len(got), len(deux))
		}
		exigerFenetresBornees(t, b, "LoadMatchMedals (Q26h)", len(deux)*lignesParMatchFenetres, 1)
	})

	t.Run("Explorateur", func(t *testing.T) {
		repo := NewExplorerRepo(b.pdb, tacXUIDMoi)
		agg, err := repo.GetMedalCountsForMatches(ctx, tacXUIDMoi, deux)
		if err != nil {
			t.Fatalf("GetMedalCountsForMatches: %v", err)
		}
		// Par match : 2 Vengeur + 3 Cible difficile + 1 autre ; trois identifiants distincts.
		if agg.Total != len(deux)*(2+3+1) || agg.Unique != 3 {
			t.Fatalf("GetMedalCountsForMatches = %+v", agg)
		}
		exigerFenetresBornees(t, b, "GetMedalCountsForMatches", len(deux)*lignesParMatchFenetres, 1)

		top, err := repo.GetTopMedalsForMatches(ctx, tacXUIDMoi, deux, 5)
		if err != nil {
			t.Fatalf("GetTopMedalsForMatches: %v", err)
		}
		if len(top) != 3 || top[0].NameID != medailleCibleFenetres || top[0].Count != len(deux)*3 {
			t.Fatalf("GetTopMedalsForMatches = %+v", top)
		}
		exigerFenetresBornees(t, b, "GetTopMedalsForMatches", len(deux)*lignesParMatchFenetres, 1)
	})

	t.Run("Médailles par joueur", func(t *testing.T) {
		rows, err := NewMedalsByXUIDRepo(b.pdb).LoadMedalsForMatchesByXUID(ctx, "halo_infinite",
			port.MedalsByXUIDFilters{MatchIDs: deux, XUIDs: []string{tacXUIDMoi, tacXUIDAdv}})
		if err != nil {
			t.Fatalf("LoadMedalsForMatchesByXUID: %v", err)
		}
		if len(rows) != 3*len(deux) {
			t.Fatalf("LoadMedalsForMatchesByXUID : %d lignes, attendu %d", len(rows), 3*len(deux))
		}
		exigerFenetresBornees(t, b, "LoadMedalsForMatchesByXUID", len(deux)*lignesParMatchFenetres, 1)
	})
}
