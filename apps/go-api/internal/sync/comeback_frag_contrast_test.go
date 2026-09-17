package sync

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/analysis"
)

// seedKillTimeline insère une frag toutes les 10 s : l'équipe qui a le plus de frags
// prend d'entrée toute son avance, puis les deux équipes alternent (elle reste devant
// tout le match) ; dup=true insère
// deux fois chaque frag de l'équipe dominée (doublons exacts observés dans
// highlight_events) : sans dédoublonnage elle passerait devant aux frags.
// ensureHighlightEvents crée la table minimale des events, SANS contrainte d'unicité : les
// doublons exacts existent en base réelle (cf. loadDistinctTeamKillEvents).
func ensureHighlightEvents(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS highlight_events (
		match_id VARCHAR, event_type VARCHAR, time_ms INTEGER, xuid VARCHAR)`); err != nil {
		t.Fatalf("create highlight_events: %v", err)
	}
}

// insertKillEvent insère `copies` fois la même frag (xuid, ms) sur le match.
func insertKillEvent(t *testing.T, db *sql.DB, matchID, xuid string, ms int64, copies int) {
	t.Helper()
	for i := 0; i < copies; i++ {
		if _, err := db.ExecContext(context.Background(),
			`INSERT INTO highlight_events (match_id, event_type, time_ms, xuid) VALUES (?, 'kill', ?, ?)`,
			matchID, ms, xuid); err != nil {
			t.Fatalf("insert kill: %v", err)
		}
	}
}

func seedKillTimeline(t *testing.T, db *sql.DB, matchID string, myFrags, enemyFrags int, dup bool) {
	t.Helper()
	ensureHighlightEvents(t, db)
	insert := func(xuid string, ms int64, copies int) {
		insertKillEvent(t, db, matchID, xuid, ms, copies)
	}
	ms := int64(1_000)
	lead, trail := "me", "enemy_xuid"
	leadN, trailN := myFrags, enemyFrags
	if enemyFrags > myFrags {
		lead, trail, leadN, trailN = trail, lead, trailN, leadN
	}
	trailCopies := 1
	if dup {
		trailCopies = 2
	}
	for i := 0; i < leadN-trailN; i++ {
		insert(lead, ms, 1)
		ms += 10_000
	}
	for i := 0; i < trailN; i++ {
		insert(trail, ms, trailCopies)
		insert(lead, ms+5_000, 1)
		ms += 10_000
	}
}

func TestBackfillDominanceFlags_FragContrast(t *testing.T) {
	cases := []struct {
		name, variant     string
		outcome           int
		myFrags, enemyFrs int
		dup               bool
		withSteaktacular  bool
		want              int
	}{
		{"sabordage en Strongholds", "Strongholds:Arena", 3, 30, 20, false, false, analysis.DominanceFlagSabordage},
		{"abnegation en Oddball", "Oddball:Arena", 2, 20, 30, false, false, analysis.DominanceFlagAbnegation},
		{"doublons dedoublonnes : resultat identique", "Strongholds:Arena", 3, 30, 20, true, false, analysis.DominanceFlagSabordage},
		{"ecart final sous 15 %", "King of the Hill:Arena", 3, 30, 27, false, false, analysis.DominanceFlagNone},
		{"Slayer exclu", "Slayer:Arena", 3, 30, 20, false, false, analysis.DominanceFlagNone},
		{"badge existant prioritaire", "Strongholds:Arena", 2, 20, 30, false, true, analysis.DominanceFlagDomination},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			playerDB, sharedDB := newInMemoryDBs(t)
			ensureDominanceFlagColumn(t, playerDB)
			const myXUID = "me"
			matchID := "m_frag_contrast_" + string(rune('a'+i))
			seedComebackMatch(t, sharedDB, matchID, c.variant, myXUID, 0, c.outcome, 1)
			seedKillTimeline(t, sharedDB, matchID, c.myFrags, c.enemyFrs, c.dup)
			if c.withSteaktacular {
				seedSteaktacularMedal(t, sharedDB, matchID, myXUID)
			}
			if err := BackfillDominanceFlags(context.Background(), sharedDB, playerDB, myXUID, []string{matchID}); err != nil {
				t.Fatalf("BackfillDominanceFlags: %v", err)
			}
			if got := readDominanceFlag(t, playerDB, matchID); got != c.want {
				t.Errorf("flag = %d, attendu %d", got, c.want)
			}
		})
	}
}

// setMatchDuration écrit duration_seconds sur un match déjà seedé : seedComebackMatch laisse
// la colonne NULL, or c'est elle qui donne la FIN DE MATCH au critère « en tête ≥ 75 % du
// temps ». Sans ce helper, la lecture de duration_seconds n'était testée qu'en NULL.
func setMatchDuration(t *testing.T, sharedDB *sql.DB, matchID string, seconds int) {
	t.Helper()
	if _, err := sharedDB.ExecContext(context.Background(),
		`UPDATE match_registry SET duration_seconds = ? WHERE match_id = ?`,
		seconds, matchID); err != nil {
		t.Fatalf("update duration_seconds: %v", err)
	}
}

// seedLateLeadTimeline : l'ennemi mène de 0 à 100 s, puis mon équipe passe devant et finit à
// 12 frags contre 2, dernière frag à 120 s.
//
//   - fin = dernière frag (120 s) : je ne suis devant que 9 s, soit 7,5 % → aucun badge ;
//   - fin = durée du match (900 s) : je suis devant 789 s, soit 87,7 % → SABORDAGE.
//
// La timeline est donc un DÉTECTEUR de la fin de match retenue.
func seedLateLeadTimeline(t *testing.T, db *sql.DB, matchID string) {
	t.Helper()
	ensureHighlightEvents(t, db)
	insertKillEvent(t, db, matchID, "enemy_xuid", 0, 1)
	insertKillEvent(t, db, matchID, "enemy_xuid", 50_000, 1)
	insertKillEvent(t, db, matchID, "me", 100_000, 1)
	for ms := int64(110_000); ms <= 120_000; ms += 1_000 {
		insertKillEvent(t, db, matchID, "me", ms, 1)
	}
}

// TestBackfillDominanceFlags_FragContrast_Duree — la durée absente NE DOIT PAS disparaître
// en silence : un duration_seconds NULL ne lève aucune erreur, donc sans WARN explicite le
// repli « fin = dernière frag » (qui change ici le verdict) serait invisible aux ops.
func TestBackfillDominanceFlags_FragContrast_Duree(t *testing.T) {
	cases := []struct {
		name      string
		durationS int // 0 = colonne laissée NULL par seedComebackMatch
		want      int
		wantWarn  bool
	}{
		{"duree absente : repli sur la derniere frag, trace", 0, analysis.DominanceFlagNone, true},
		{"duree presente : la fin du match vient de duration_seconds", 900, analysis.DominanceFlagSabordage, false},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			playerDB, sharedDB := newInMemoryDBs(t)
			ensureDominanceFlagColumn(t, playerDB)
			const myXUID = "me"
			matchID := "m_frag_duree_" + string(rune('a'+i))
			seedComebackMatch(t, sharedDB, matchID, "Strongholds:Arena", myXUID, 0, 3 /* loss */, 1)
			seedLateLeadTimeline(t, sharedDB, matchID)
			if c.durationS > 0 {
				setMatchDuration(t, sharedDB, matchID, c.durationS)
			}
			buf := captureSlog(t)
			if err := BackfillDominanceFlags(
				context.Background(), sharedDB, playerDB, myXUID, []string{matchID}); err != nil {
				t.Fatalf("BackfillDominanceFlags: %v", err)
			}
			if got := readDominanceFlag(t, playerDB, matchID); got != c.want {
				t.Errorf("flag = %d, attendu %d", got, c.want)
			}
			if got := hasWarn(t, buf, "durée absente, fin = dernière frag"); got != c.wantWarn {
				t.Errorf("WARN « durée absente » présent = %v, attendu %v — journal :\n%s",
					got, c.wantWarn, buf.String())
			}
		})
	}
}

// seedThirdTeamParticipant ajoute un joueur d'une TROISIÈME équipe au match (Multi Team).
func seedThirdTeamParticipant(t *testing.T, sharedDB *sql.DB, matchID string, teamID int) {
	t.Helper()
	if _, err := sharedDB.ExecContext(context.Background(), `
		INSERT INTO match_participants (match_id, xuid, team_id, outcome)
		VALUES (?, 'third_xuid', ?, 3)`, matchID, teamID); err != nil {
		t.Fatalf("insert participant troisieme equipe: %v", err)
	}
}

// TestBackfillDominanceFlags_FragContrast_TroisEquipes — SABORDAGE / ABNÉGATION opposent
// DEUX camps. Sur un Multi Team à objectifs, le filtre `team_id IN (0, 1)` de la timeline
// retirait les équipes 2+ au lieu d'écarter le match : le verdict se rendait alors sur une
// FRACTION du match. Le comptage des équipes doit l'écarter entièrement.
func TestBackfillDominanceFlags_FragContrast_TroisEquipes(t *testing.T) {
	cases := []struct {
		name      string
		thirdTeam bool
		want      int
	}{
		{"deux equipes : badge rendu", false, analysis.DominanceFlagSabordage},
		{"trois equipes : match ecarte", true, analysis.DominanceFlagNone},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			playerDB, sharedDB := newInMemoryDBs(t)
			ensureDominanceFlagColumn(t, playerDB)
			const myXUID = "me"
			matchID := "m_frag_equipes_" + string(rune('a'+i))
			seedComebackMatch(t, sharedDB, matchID, "Strongholds:Arena", myXUID, 0, 3 /* loss */, 1)
			if c.thirdTeam {
				seedThirdTeamParticipant(t, sharedDB, matchID, 2)
			}
			// Timeline identique dans les deux cas : 30 frags contre 20 entre les équipes 0 et 1.
			seedKillTimeline(t, sharedDB, matchID, 30, 20, false)
			if err := BackfillDominanceFlags(
				context.Background(), sharedDB, playerDB, myXUID, []string{matchID}); err != nil {
				t.Fatalf("BackfillDominanceFlags: %v", err)
			}
			if got := readDominanceFlag(t, playerDB, matchID); got != c.want {
				t.Errorf("flag = %d, attendu %d", got, c.want)
			}
		})
	}
}
