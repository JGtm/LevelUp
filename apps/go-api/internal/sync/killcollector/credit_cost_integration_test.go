//go:build integration

// Package killcollector — credit_cost_integration_test.go : LE COUT PAR MATCH DE LA PASSE
// CREDIT, MESURE SUR LES VRAIES MIGRATIONS (lot 5.12, 2026-09-21).
//
// # CE QUI A ETE OBSERVE EN PRODUCTION, ET CE QUE LA MESURE A TROUVE
//
// `levelup backfill-killsource` du 2026-09-21 : la passe des FILMS a decode 1 598 films en
// 4 h 14 (journalisee film par film), puis la passe CREDIT-SEUL (« 9 144 matchs a examiner »,
// SQL -> SQL) a tourne PLUS DE 22 HEURES sans terminer ni journaliser une ligne.
//
// L hypothese de depart etait la lecture de l enrichissement par match
// (`persist.FilmPassForMatch` sur la vue `match_kill_events_latest`, dont le `QUALIFY` porte une
// fenetre sur toute la table append-only). ELLE EST FAUSSE, ET C EST MESURE :
// `kill_events_film_pass_cost_integration_test.go` colle le plan DuckDB — le predicat
// `match_id = ?` est POUSSE JUSQU AU SEQ_SCAN, sous la fenetre — et le cout est de ~4 ms par
// appel sur 180 000 lignes, soit ~40 s pour 9 144 matchs. Ce n est pas la cause des 22 heures.
//
// LA CAUSE EST `evenementsDuMatch` : sa jointure `LEFT JOIN v_gamertag_lookup`. Cette vue
// canonique d identite agrege, a CHAQUE evaluation, `match_participants` GROUP BY, DEUX
// balayages fenetres de `match_kill_events_latest` et deux de `killer_victim_pairs`, joints en
// FULL OUTER JOIN. Le filtre du lecteur porte sur `highlight_events.match_id` : il ne peut donc
// RIEN pousser dans la vue, qui est materialisee ENTIEREMENT — une fois PAR MATCH, 9 144 fois.
// C est la cette quadratique, et le present fichier la mesure.
package killcollector

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/persist"
)

// Les dimensions du banc : l ordre de grandeur d une campagne de production (~160 000 lignes de
// `match_kill_events` ajoutees par backfill, 248 566 lignes de `killer_victim_pairs` en prod).
const (
	nbMatchsFilmBanc  = 2000
	nbPassesFilmBanc  = 3
	nbMortsFilmBanc   = 30
	nbPairesBanc      = 50000
	nbMatchsCreditMes = 20
	// budgetDesLecturesParMatch : LE BUDGET des lectures par match (l ecriture est hors budget,
	// voir TestPasseCredit_BudgetDesLecturesParMatch). 10 ms tient les 9 144 matchs du registre en
	// moins de deux minutes de lecture ; le defaut mesure, lui, coutait 55 a 77 ms de lecture par
	// match sur ce banc, et des secondes sur la base de production — d ou les 22 heures.
	budgetDesLecturesParMatch = 10 * time.Millisecond
)

// peuplerBancCredit fabrique la base du banc :
//
//   - `match_kill_events` : `nbMatchsFilmBanc` matchs x `nbPassesFilmBanc` passes de film ;
//   - `killer_victim_pairs` : `nbPairesBanc` couples (la seconde source de `v_gamertag_lookup`) ;
//   - `highlight_events` + `match_participants` : les matchs que la passe credit va examiner.
//
// Les insertions sont des `INSERT … SELECT` de fixture : INSERT purs, la seule ecriture qu ADR
// 0026 autorise sur une table append-only.
func peuplerBancCredit(t *testing.T, db *sql.DB) {
	t.Helper()
	debut := time.Now()
	if _, err := db.Exec(`
		INSERT INTO match_kill_events (
			match_id, decode_pass, decoder_rev, written_at, publishable, time_ms,
			victim_gamertag, victim_xuid, feed_killer_gamertag, feed_killer_xuid, feed_present,
			assist_known, assist_extra_count, killer_damage_pct, read_path, read_origin)
		SELECT
			printf('film-%05d', m.i), printf('film-%05d-passe-%d', m.i, p.i), 'rev-banc',
			TIMESTAMP '2026-01-01 00:00:00' + to_minutes(CAST(p.i AS INTEGER)),
			TRUE, CAST(d.i AS INTEGER) * 1000,
			printf('victime-%02d', d.i), printf('9%011d', d.i),
			printf('tueur-%02d', d.i), printf('8%011d', d.i), TRUE,
			FALSE, 0, 70, ?, 'credit-concordant'
		FROM range(0, ?) AS m(i), range(0, ?) AS p(i), range(0, ?) AS d(i)`,
		persist.FilmReadPaths[0], nbMatchsFilmBanc, nbPassesFilmBanc, nbMortsFilmBanc); err != nil {
		t.Fatalf("banc match_kill_events: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO killer_victim_pairs (match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag)
		SELECT printf('film-%05d', CAST(i / 100 AS INTEGER)),
			printf('8%011d', i % 200), printf('tueur-%03d', i % 200),
			printf('9%011d', (i + 7) % 200), printf('victime-%03d', (i + 7) % 200)
		FROM range(0, ?) AS t(i)`, nbPairesBanc); err != nil {
		t.Fatalf("banc killer_victim_pairs: %v", err)
	}
	// Les matchs de la passe credit : 3 morts chacun, deux joueurs — la forme de `duelSimple`,
	// produite en masse.
	for i := 0; i < nbMatchsCreditMes; i++ {
		insererEvenements(t, db, fmt.Sprintf("credit-%05d", i), duelSimple())
	}
	t.Logf("banc peuple en %s : %d lignes match_kill_events, %d couples, %d matchs credit",
		time.Since(debut).Round(time.Millisecond),
		nbMatchsFilmBanc*nbPassesFilmBanc*nbMortsFilmBanc, nbPairesBanc, nbMatchsCreditMes)
}

// collerPlanCredit colle le plan DuckDB d une requete — LA PIECE qui dit ce qui est materialise.
func collerPlanCredit(t *testing.T, db *sql.DB, titre, requete string, args ...any) {
	t.Helper()
	rows, err := db.Query("EXPLAIN "+requete, args...)
	if err != nil {
		t.Fatalf("EXPLAIN %s: %v", titre, err)
	}
	defer func() { _ = rows.Close() }()
	var plan strings.Builder
	for rows.Next() {
		var cle, valeur string
		if err := rows.Scan(&cle, &valeur); err != nil {
			t.Fatalf("EXPLAIN %s (scan): %v", titre, err)
		}
		plan.WriteString(valeur + "\n")
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("EXPLAIN %s (rows): %v", titre, err)
	}
	t.Logf("PLAN — %s\n%s", titre, plan.String())
}

// requeteDefautJointureParMatch : LA FORME D AVANT LE LOT 5.12, gardee ICI et nulle part ailleurs.
//
// C est la piece du defaut : la jointure de la vue canonique d identite DANS la lecture par
// match. Elle ne sert qu a etre chronometree a cote de la forme de production — aucun chemin de
// production ne la contient plus.
const requeteDefautJointureParMatch = `
		SELECT he.xuid, LOWER(he.event_type), COALESCE(he.time_ms, 0), COALESCE(g.gamertag, '')
		FROM highlight_events he
		LEFT JOIN v_gamertag_lookup g ON g.xuid = he.xuid
		WHERE he.match_id = ?
		  AND LOWER(he.event_type) IN (?, ?)
		  AND he.xuid IS NOT NULL AND he.xuid <> ''
		ORDER BY he.time_ms, he.xuid, he.event_type`

// facteurDeDefautAttendu : l ecart MINIMAL exige entre la forme du defaut et celle de
// production. Mesure du 2026-09-21 sur ce banc : 77,2 ms contre 0,9 ms, soit 84x. Exiger 5x
// laisse toute la marge du bruit de machine tout en gardant la reproduction du defaut vraie.
const facteurDeDefautAttendu = 5

// TestPasseCredit_OuVaLeTemps — LA DECOMPOSITION, et c est elle qui NOMME le coupable.
//
// Elle chronometre separement, sur le meme banc, les trois lectures par match de la passe credit
// et la FORME DU DEFAUT (la jointure d identite par match). L ecart entre les deux premieres
// lignes est le cout de `v_gamertag_lookup` materialisee a chaque match — la cause des 22 heures.
func TestPasseCredit_OuVaLeTemps(t *testing.T) {
	db := openSharedTestDB(t)
	peuplerBancCredit(t, db)
	ctx := context.Background()

	mesurer := func(nom string, appel func(matchID string) error) time.Duration {
		debut := time.Now()
		for i := 0; i < nbMatchsCreditMes; i++ {
			if err := appel(fmt.Sprintf("credit-%05d", i)); err != nil {
				t.Fatalf("%s: %v", nom, err)
			}
		}
		d := time.Since(debut) / nbMatchsCreditMes
		t.Logf("%-52s %s par match", nom, d.Round(time.Microsecond))
		return d
	}

	viderLignes := func(requete string, args ...any) error {
		rows, err := db.QueryContext(ctx, requete, args...)
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() { //nolint:revive // on ne veut que le cout de la lecture
		}
		return rows.Err()
	}

	defaut := mesurer("DEFAUT — evenements AVEC v_gamertag_lookup jointe par match",
		func(m string) error {
			return viderLignes(requeteDefautJointureParMatch, m, "kill", "death")
		})
	production := mesurer("production — evenements du match (annuaire en memoire)",
		func(m string) error {
			return viderLignes(requeteEvenementsDuMatch, m, "kill", "death")
		})
	_ = mesurer("preseance — COUNT(*) sur match_kill_events_latest", func(m string) error {
		var n int
		return db.QueryRowContext(ctx, `SELECT COUNT(*) FROM match_kill_events_latest
			WHERE match_id = ? AND read_path IN (?, ?)`, m, persist.FilmReadPaths[0], persist.FilmReadPaths[1]).Scan(&n)
	})
	_ = mesurer("enrichissement — persist.FilmPassForMatch", func(m string) error {
		_, err := persist.FilmPassForMatch(ctx, db, m)
		return err
	})
	// L ECRITURE, sur le meme banc : ce qui reste du cout d un match quand la lecture ne coute
	// plus rien. Un match ecrit 3 morts dans sa propre transaction.
	credit := NewCreditCollector(db, sharedWriter(db))
	_ = mesurer("ecriture — la passe d un match (CollectMatch entier)", func(m string) error {
		_, _, err := credit.CollectMatch(ctx, m)
		return err
	})

	// L annuaire, une fois pour la passe — la lecture qui remplace les 9 144 jointures.
	debutAnnuaire := time.Now()
	noms, err := NewCreditCollector(db, sharedWriter(db)).annuaire(ctx)
	if err != nil {
		t.Fatalf("annuaire: %v", err)
	}
	t.Logf("%-52s %s pour %d xuids", "annuaire de la passe — UNE fois",
		time.Since(debutAnnuaire).Round(time.Microsecond), len(noms.parXUID))

	if defaut < facteurDeDefautAttendu*production {
		t.Errorf("la jointure par match ne coute que %s contre %s pour la lecture de production "+
			"(facteur attendu >= %d) : soit le banc ne reproduit plus le defaut (revoir ses "+
			"dimensions), soit DuckDB pousse desormais le filtre dans la vue d identite — "+
			"re-mesurer avant de conclure quoi que ce soit", defaut, production, facteurDeDefautAttendu)
	}
}

// TestPasseCredit_BudgetDesLecturesParMatch — LE BUDGET, ET IL PORTE SUR LES LECTURES.
//
// POURQUOI PAS SUR `CollectMatch` ENTIER, qui serait plus parlant : parce que la passe d un match
// ECRIT, et que l ecriture d une transaction par match est bornee par le disque, pas par le
// volume de la base. Mesure du 2026-09-21 sur ce banc : 20,8 ms pour un `CollectMatch` entier
// dont ~2,8 ms de lecture — et cette ecriture varie de 9 a 75 ms selon la charge de la machine.
// Un budget pose dessus serait soit instable en CI, soit si large qu il laisserait repasser le
// defaut (108 ms par match avant correction). LES LECTURES, ELLES, SONT LA QUANTITE QUE CE LOT
// CHANGE et la seule qui grossit avec la base : c est sur elles que le garde-fou tient.
//
// Rouge = la passe credit est redevenue quadratique (une vue globale evaluee par match).
func TestPasseCredit_BudgetDesLecturesParMatch(t *testing.T) {
	db := openSharedTestDB(t)
	peuplerBancCredit(t, db)
	ctx := context.Background()

	collerPlanCredit(t, db, "AVANT (defaut) — les evenements d un match, vue d identite jointe",
		requeteDefautJointureParMatch, "credit-00000", "kill", "death")
	collerPlanCredit(t, db, "APRES (production) — les evenements d un match",
		requeteEvenementsDuMatch, "credit-00000", "kill", "death")

	credit := NewCreditCollector(db, sharedWriter(db))
	// L annuaire charge d abord : il appartient a la passe, pas au match, et son cout ne doit
	// donc pas etre impute au premier match.
	if _, err := credit.annuaire(ctx); err != nil {
		t.Fatalf("annuaire: %v", err)
	}

	// La boucle fait EXACTEMENT ce que fait la production sur cette population : les matchs du
	// banc credit n ont pas de film, donc la sonde de preseance repond non et l enrichissement
	// n est pas relu. Son cout propre (1,2 ms par match) est mesure a part par
	// [TestPasseCredit_OuVaLeTemps] — les deux tests se completent, aucun des deux ne devine.
	debut := time.Now()
	for i := 0; i < nbMatchsCreditMes; i++ {
		m := fmt.Sprintf("credit-%05d", i)
		if _, err := credit.evenementsDuMatch(ctx, m); err != nil {
			t.Fatalf("evenements %s: %v", m, err)
		}
		couvert, err := credit.covertParUnFilm(ctx, m)
		if err != nil {
			t.Fatalf("preseance %s: %v", m, err)
		}
		if couvert {
			if _, err := persist.FilmPassForMatch(ctx, db, m); err != nil {
				t.Fatalf("enrichissement %s: %v", m, err)
			}
		}
	}
	moyenne := time.Since(debut) / nbMatchsCreditMes
	t.Logf("lectures par match : %s (%d matchs, %d lignes de film en base) — budget %s",
		moyenne.Round(time.Microsecond), nbMatchsCreditMes,
		nbMatchsFilmBanc*nbPassesFilmBanc*nbMortsFilmBanc, budgetDesLecturesParMatch)
	if moyenne > budgetDesLecturesParMatch {
		t.Fatalf("les lectures d un match coutent %s, budget %s : une vue globale est evaluee "+
			"par match (lot 5.12)", moyenne, budgetDesLecturesParMatch)
	}

	// ET LA PASSE REELLE, SUR LA MEME BASE : le budget ne vaut rien si le producteur ne produit
	// plus. Les 20 matchs doivent tous s ecrire avec leurs 3 morts.
	sum := credit.CollectMatches(ctx, matchsDuBanc())
	if sum.Written != nbMatchsCreditMes || sum.Deaths != 3*nbMatchsCreditMes || sum.Errors != 0 {
		t.Fatalf("CollectMatches = %+v, attendu %d ecrits / %d morts / 0 erreur",
			sum, nbMatchsCreditMes, 3*nbMatchsCreditMes)
	}
	t.Logf("passe reelle (ecriture incluse) : %s par match — hors budget, bornee par le disque",
		(sum.ElapsedTime / nbMatchsCreditMes).Round(time.Microsecond))
}

// matchsDuBanc : les identifiants des matchs credit du banc.
func matchsDuBanc() []string {
	ids := make([]string, 0, nbMatchsCreditMes)
	for i := 0; i < nbMatchsCreditMes; i++ {
		ids = append(ids, fmt.Sprintf("credit-%05d", i))
	}
	return ids
}

// TestProgressionJournaliseeMemeQuandLesMatchsEchouent — LE JALON N EST PAS SAUTE PAR UNE ERREUR.
//
// Constat de la revue adversariale du 2026-09-21 : la journalisation de progression etait placee
// APRES un `continue` qui traitait le cas d erreur, alors que le compteur de matchs examines,
// lui, comptait les erreurs. Un jalon tombant sur un match en echec etait donc perdu, et une
// passe dont TOUS les matchs echouent ne journalisait aucune progression — exactement le silence
// que le lot supprime.
//
// Le test fabrique la panne la plus franche (la table source retiree) et exige les jalons.
func TestProgressionJournaliseeMemeQuandLesMatchsEchouent(t *testing.T) {
	db := openSharedTestDB(t)
	if _, err := db.Exec(`DROP TABLE highlight_events`); err != nil {
		t.Fatalf("retrait de la table source: %v", err)
	}

	var journal bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&journal, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	// 1 200 matchs : deux jalons attendus (500 et 1 000), tous les deux sur des matchs en echec.
	ids := make([]string, 0, 1200)
	for i := 0; i < 1200; i++ {
		ids = append(ids, fmt.Sprintf("echec-%05d", i))
	}
	sum := NewCreditCollector(db, sharedWriter(db)).CollectMatches(context.Background(), ids)
	if sum.Errors != len(ids) {
		t.Fatalf("CollectMatches: %d erreurs, attendu %d (la panne doit toucher tous les matchs)",
			sum.Errors, len(ids))
	}

	jalons := strings.Count(journal.String(), "killsource: credit — progression")
	if jalons != 2 {
		t.Errorf("%d ligne(s) de progression, attendu 2 (matchs 500 et 1 000) — une passe qui "+
			"echoue reste muette", jalons)
	}
	if !strings.Contains(journal.String(), "killsource: credit — passe terminee") {
		t.Error("bilan final absent du journal")
	}
}
