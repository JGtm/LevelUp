package duckdb

// resumes_parite_corpus_test.go — LE CORPUS DE LA PARITÉ ANCIEN / NOUVEAU des lecteurs basculés
// sur les résumés du journal (ADR 0039, plan PLAN_PERF_PAGES_V801 étape 9).
//
// Un shared `:memory:` aux vraies migrations, dont le journal est posé par l'aide unique
// ([poserJournal]) : les résumés sont calculés par la fonction pure de l'écrivain réel (l'égalité
// aide / persist est tenue par resumes_aide_persist_test.go).
//
// Les cas qui font diverger deux lecteurs d'un même chiffre : tueur et victime bots (xuid NULL),
// assistant anonyme (« mesuré, personne n'a assisté »), assistant bot nommé par son gamertag
// seul, victime bot, source inconnue, assistance non lue, passe non publiable, match sans
// journal, match de Campagne, passe supplantée.

import (
	"database/sql"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
)

const (
	pJ1, pJ2, pJ3 = "xj1", "xj2", "xj3" // trois joueurs humains
	pTagA, pTagB  = uint32(0x0a000001), uint32(0x0a000002)
)

// corpusPariteMatchs : les matchs du corpus, tous inscrits au registre.
var corpusPariteMatchs = []string{"pA", "pB", "pC", "pD", "pE", "pF", "pG"}

// corpusParite monte le corpus. pE est un match de Campagne de Halo 5 ; pF n'a pas de journal ; pG a
// deux passes (l'ancienne, supplantée, ne doit compter nulle part).
func corpusParite(t *testing.T) *sql.DB {
	t.Helper()
	db := ouvrirSharedMigree(t)
	corpusPariteSur(t, db)
	return db
}

// corpusPariteSur pose le corpus sur un shared déjà migré (la base d'une baseNotee, dont les
// lectures sont notées).
func corpusPariteSur(t *testing.T, db *sql.DB) {
	t.Helper()
	camp := analysis.CampaignExcludedVariantIDs("halo_5")[0]
	for _, m := range corpusPariteMatchs {
		variante := "aaaaaaaa-0000-0000-0000-000000000001"
		if m == "pE" {
			variante = camp
		}
		if _, err := db.Exec(`INSERT INTO match_registry (match_id, start_time, start_time_utc, game_variant_id, is_firefight)
			VALUES (?, '2026-09-01 10:00:00', '2026-09-01 10:00:00+00', ?, FALSE)`, m, variante); err != nil {
			t.Fatalf("registre %s : %v", m, err)
		}
		for i, x := range []string{pJ1, pJ2, pJ3} {
			if _, err := db.Exec(`INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome, kills)
				VALUES (?, ?, ?, ?, 2, 5)`, m, x, "GT"+x, i%2); err != nil {
				t.Fatalf("participant %s : %v", m, err)
			}
		}
	}
	poserJournal(t, db, lignesVariees("pA", true)...)
	poserJournal(t, db, lignesVariees("pB", false)...)
	poserJournal(t, db, lignesSansAssistance("pC")...)
	poserJournal(t, db, lignesBotsSeuls("pD")...)
	poserJournal(t, db, lignesVariees("pE", true)...)
	// pG : l'ancienne passe (un seul frag de J1 sur J2), puis la passe courante.
	poserJournal(t, db, ligneJournal{match: "pG", publiable: true, passe: "ancienne", tueur: pJ1, victime: pJ2,
		tueurGT: "GT" + pJ1, tag: pTagA, timeMS: 1000})
	courante := lignesVariees("pG", true)
	for i := range courante {
		courante[i].passe, courante[i].ecart = "courante", time.Hour
	}
	poserJournalAJour(t, db, courante...)
}

// lignesVariees : une passe qui traverse chaque grain des trois familles.
func lignesVariees(match string, publiable bool) []ligneJournal {
	l := func(ms int, tueur, victime string, assistX, assistGT string, known bool, tag uint32, pK, pA *uint8) ligneJournal {
		g := ligneJournal{match: match, publiable: publiable, timeMS: ms, tueur: tueur, victime: victime,
			assistant: assistX, assistantGT: assistGT, assistKnown: known, tag: tag, pctTueur: pK, pctAssist: pA}
		if tueur != "" {
			g.tueurGT = "GT" + tueur
		} else {
			g.tueurGT = "Bot K"
		}
		return g
	}
	return []ligneJournal{
		l(1000, pJ1, pJ2, "", "", true, pTagA, pct(100), nil),            // mesuré, personne n'a assisté
		l(2000, pJ1, pJ2, pJ3, "GT"+pJ3, true, pTagA, pct(40), pct(24)),  // basse (24)
		l(3000, pJ1, pJ2, pJ3, "GT"+pJ3, true, pTagA, pct(40), pct(25)),  // moyenne (25, borne incluse)
		l(4000, pJ1, pJ3, pJ2, "GT"+pJ2, true, pTagB, pct(30), pct(50)),  // moyenne (50, borne incluse)
		l(5000, pJ1, pJ3, pJ2, "GT"+pJ2, true, pTagB, pct(30), pct(51)),  // haute (51), volée
		l(6000, pJ1, pJ3, pJ2, "GT"+pJ2, true, pTagB, nil, pct(70)),      // volée sans part du tueur : non
		l(7000, pJ2, pJ1, pJ3, "GT"+pJ3, true, 0, pct(60), pct(228)),     // source inconnue, part hors échelle
		l(8000, pJ2, "", "", "Bot A", true, pTagA, pct(20), pct(35)),     // victime bot, assistant bot nommé
		l(9000, pJ3, pJ1, "", "", false, 0, nil, nil),                    // assistance non lue
		l(10000, "", pJ1, pJ2, "GT"+pJ2, true, pTagA, pct(10), pct(10)),  // tueur bot, assisté par un joueur
		l(11000, "", pJ2, "", "Bot A", true, pTagB, pct(10), pct(80)),    // tueur bot assisté par un bot
		l(12000, "", pJ3, "", "Bot A", true, pTagB, pct(10), nil),        // même paire de bots, autre victime, sans part
		l(13000, pJ1, "", "", "", true, 0, nil, nil),                     // victime bot, source inconnue, anonyme
		l(14000, pJ3, pJ2, pJ1, "GT"+pJ1, true, pTagA, pct(10), pct(10)), // basse
	}
}

// lignesSansAssistance : le film ne lit aucune assistance (assist_known = FALSE partout).
func lignesSansAssistance(match string) []ligneJournal {
	return []ligneJournal{
		{match: match, publiable: true, tueur: pJ1, tueurGT: "GT" + pJ1, victime: pJ2, tag: pTagA, timeMS: 1000},
		{match: match, publiable: true, tueur: pJ2, tueurGT: "GT" + pJ2, victime: pJ1, timeMS: 2000},
	}
}

// lignesBotsSeuls : aucun humain ne tue ; tout est xuid NULL.
func lignesBotsSeuls(match string) []ligneJournal {
	return []ligneJournal{
		{match: match, publiable: true, tueurGT: "Bot K", victime: pJ1, assistKnown: true, timeMS: 1000, tag: pTagA},
		{match: match, publiable: true, tueurGT: "Bot K", victime: "", assistKnown: true, assistantGT: "Bot A",
			pctAssist: pct(50), timeMS: 2000},
	}
}
