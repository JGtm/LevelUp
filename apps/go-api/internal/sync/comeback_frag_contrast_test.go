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
func seedKillTimeline(t *testing.T, db *sql.DB, matchID string, myFrags, enemyFrags int, dup bool) {
	t.Helper()
	ctx := context.Background()
	// Table minimale sans contrainte d'unicité : les doublons exacts existent en base réelle.
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS highlight_events (
		match_id VARCHAR, event_type VARCHAR, time_ms INTEGER, xuid VARCHAR)`); err != nil {
		t.Fatalf("create highlight_events: %v", err)
	}
	insert := func(xuid string, ms int64, copies int) {
		for i := 0; i < copies; i++ {
			if _, err := db.ExecContext(ctx,
				`INSERT INTO highlight_events (match_id, event_type, time_ms, xuid) VALUES (?, 'kill', ?, ?)`,
				matchID, ms, xuid); err != nil {
				t.Fatalf("insert kill: %v", err)
			}
		}
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
