//go:build research

package duckdb

// tactical_repo_coequipiers_research_test.go — LES QUATRE SONDES DU LOT L2.0 du plan
// `.ai/V7.5/PLAN_TACTIQUE_COEQUIPIERS_2026-10-14.md`, qui fixent ce que « coequipier » et « vengeur »
// veulent dire pour un BOT :
//
//	1. des bots dans match_participants ? (xuid vide ou NULL) et avec quelle equipe ;
//	2. le gamertag d'un bot est-il unique dans un match ? (sinon il ne designe pas un joueur) ;
//	3. des doublons (match_id, victim_xuid, time_ms) dans match_kill_events_latest ? (la cle
//	   d'une mort, sur laquelle les coequipiers se lisent) ;
//	4. la part des morts dont le tueur est un bot (feed_killer_xuid NULL, gamertag renseigne).
//
// LECTURE SEULE, SUR UNE COPIE : jamais le fichier que le serveur ou un backfill tient.
//
//	SONDE_SHARED=<copie de shared_matches_v2.duckdb> \
//	  go test -tags research -run TestSondesCoequipiers -v ./internal/platform/duckdb/

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestSondesCoequipiers(t *testing.T) {
	chemin := os.Getenv("SONDE_SHARED")
	if chemin == "" {
		t.Skip("SONDE_SHARED non pose : sondes du parc (copie de shared_matches_v2.duckdb)")
	}
	db, err := sql.Open("duckdb", chemin+"?access_mode=read_only")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	sondes := []struct{ nom, requete string }{
		{"1. participants sans xuid (bots) et leur equipe", `
			SELECT count(*) AS lignes, count(DISTINCT match_id) AS matchs,
			       count(*) FILTER (WHERE team_id IS NOT NULL) AS avec_equipe
			FROM match_participants WHERE xuid IS NULL OR xuid = ''`},
		{"2. gamertags de bot dupliques dans un match", `
			SELECT count(*) AS groupes_dupliques FROM (
			  SELECT match_id, gamertag FROM match_participants
			  WHERE xuid IS NULL OR xuid = '' GROUP BY 1, 2 HAVING count(*) > 1)`},
		{"3. doublons de mort (match, victime, instant)", `
			SELECT count(*) AS cles_dupliquees FROM (
			  SELECT match_id, victim_xuid, time_ms FROM match_kill_events_latest
			  WHERE victim_xuid IS NOT NULL GROUP BY 1, 2, 3 HAVING count(*) > 1)`},
		{"4. morts dont le tueur est un bot", `
			SELECT count(*) AS morts,
			       count(*) FILTER (WHERE feed_killer_xuid IS NULL AND feed_killer_gamertag IS NOT NULL) AS tueur_bot
			FROM match_kill_events_latest WHERE publishable`},
	}
	for _, s := range sondes {
		rows, err := db.Query(s.requete)
		if err != nil {
			t.Fatalf("%s: %v", s.nom, err)
		}
		cols, _ := rows.Columns()
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		for rows.Next() {
			if err := rows.Scan(ptrs...); err != nil {
				t.Fatal(err)
			}
			t.Logf("%s : %v = %v", s.nom, cols, vals)
		}
		rows.Close()
	}
}
