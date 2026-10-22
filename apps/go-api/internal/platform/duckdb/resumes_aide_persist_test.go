package duckdb_test

// resumes_aide_persist_test.go — L'AIDE DE TEST DES RÉSUMÉS ÉGALE L'ÉCRIVAIN RÉEL (ADR 0039).
//
// Les fixtures du paquet `duckdb` posent leurs résumés par une aide ([ecrireResumes]) qui relit la
// passe en base et la résume par `analysis/killsummary`, parce que le paquet ne peut pas importer
// `persist`. Ce test écrit un corpus par `persist` (journal ET résumés, chemin réel), relève les
// vues `_latest`, laisse l'aide les réécrire, relève de nouveau : les deux relevés sont égaux
// champ pour champ. Une divergence de l'aide (adaptateur de ligne, grain) fait mentir toutes les
// fixtures du paquet.

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"

	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/persist"
	ddb "levelup/go-api/internal/platform/duckdb"
)

func pct8(v uint8) *uint8 { return &v }

func mort(ms int, victimeGT, victimeX, tueurGT, tueurX string) persist.KillEventInsert {
	return persist.KillEventInsert{TimeMS: ms, VictimGamertag: victimeGT, VictimXUID: victimeX,
		FeedKillerGamertag: tueurGT, FeedKillerXUID: tueurX, FeedPresent: true,
		ReadPath: killscope.ReadPathFilmWalk, ReadOrigin: "credit-concordant"}
}

func passeVariee(match string, publiable bool) persist.KillSourceBatch {
	a := mort(1000, "Bob", "2", "Alice", "1")
	a.SourceTag, a.SourceCategory, a.AssistKnown = 0x0a000001, "None", true
	a.AssistGamertag, a.AssistXUID = "Carol", "3"
	a.KillerDamagePct, a.AssistDamagePct = pct8(40), pct8(59)
	b := mort(2000, "Bot1", "", "Alice", "1")
	b.SourceTag, b.SourceCategory, b.AssistKnown = 0x0a000002, "Headshot", true
	c := mort(3000, "Bob", "2", "BotK", "")
	c.SourceTag, c.SourceCategory = 0x0a000001, "None"
	c.AssistKnown, c.AssistGamertag, c.AssistDamagePct = true, "BotA", pct8(20)
	d := mort(4000, "Alice", "1", "Bob", "2")
	d.AssistKnown, d.KillerDamagePct = true, pct8(100)
	e := mort(5000, "Bob", "2", "Alice", "1") // assistance non lue
	return persist.KillSourceBatch{MatchID: match, DecoderRev: "test", Publishable: publiable,
		Deaths: []persist.KillEventInsert{a, b, c, d, e}}
}

// releve : les trois vues, sans les colonnes techniques (id, génération).
func releve(t *testing.T, db *sql.DB) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	for nom, q := range map[string]string{
		"S0": `SELECT match_id, summary_rev, src_decode_pass, CAST(written_at AS VARCHAR), publishable,
			n_rows, n_assist_known, n_killer_xuid, n_src_killer FROM match_kill_heads_latest`,
		"S1": `SELECT match_id, summary_rev, src_decode_pass, CAST(written_at AS VARCHAR), publishable,
			feed_killer_xuid, victim_xuid, source_tag, kills FROM match_kill_pairs_latest`,
		"S2": `SELECT match_id, summary_rev, src_decode_pass, CAST(written_at AS VARCHAR), publishable,
			assist_xuid, assist_gamertag, feed_killer_xuid, feed_killer_gamertag, victim_xuid,
			n, n_stolen, sum_assist_pct, n_assist_pct, n_low, n_mid, n_high FROM match_assist_pairs_latest`,
	} {
		rows, err := db.Query(q + ` ORDER BY ALL`)
		if err != nil {
			t.Fatalf("%s : %v", nom, err)
		}
		cols, _ := rows.Columns()
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				t.Fatalf("scan %s : %v", nom, err)
			}
			out[nom] = append(out[nom], fmt.Sprint(vals...))
		}
		_ = rows.Close()
	}
	return out
}

func TestAideDeTest_EgaleAPersist(t *testing.T) {
	db := ddb.OuvrirSharedMigreePourTest(t)
	p := persist.NewKillSourcePersister(db)
	ctx := context.Background()
	for _, b := range []persist.KillSourceBatch{passeVariee("m1", true), passeVariee("m2", false)} {
		if err := p.PersistPass(ctx, b); err != nil {
			t.Fatalf("PersistPass %s : %v", b.MatchID, err)
		}
	}
	// m1 reçoit une seconde passe : l'aide doit résumer la COURANTE.
	seconde := passeVariee("m1", true)
	seconde.Deaths = seconde.Deaths[:3]
	if err := p.PersistPass(ctx, seconde); err != nil {
		t.Fatalf("PersistPass m1 bis : %v", err)
	}
	reel := releve(t, db)
	if len(reel["S0"]) != 2 || len(reel["S1"]) == 0 || len(reel["S2"]) == 0 {
		t.Fatalf("corpus trop pauvre : %v", reel)
	}
	ddb.EcrireResumesAide(t, db)
	if aide := releve(t, db); !reflect.DeepEqual(reel, aide) {
		t.Fatalf("l'aide de test diverge de l'écrivain réel\nréel : %v\naide : %v", reel, aide)
	}
}
