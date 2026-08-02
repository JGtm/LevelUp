package migration

// steps_shared_kill_events_from_pairs.go — LA REPRISE : tout ce que porte `killer_victim_pairs`
// entre, DEDUPLIQUE, dans `match_kill_events`. Premier temps de la bascule.
//
// ─── CE QUE CETTE MIGRATION FAIT, ET CE QU ELLE NE FAIT PAS ENCORE ─────────────────────────
//
// ELLE FAIT : la reprise dedupliquee des couples dans la table canonique, la suppression de
// `v_killer_victim_full` (travail mort), et la recreation de `v_gamertag_lookup` sur la table
// canonique. Apres elle, `match_kill_events` porte le journal des morts de TOUS les matchs, et
// plus seulement des 949 dont le film a ete decode.
//
// ELLE NE FAIT PAS le retrait de `killer_victim_pairs`, ET C EST UNE MESURE QUI L A DECIDE, pas
// une prudence. Le retrait supposait que la table canonique dise au moins autant que la table
// historique. Elle en dit MOINS sur les matchs couverts par un film, et le manque est chiffre —
// oracle : les evenements `death` de `highlight_events`, mesures le 2026-08-02 sur les 949
// matchs a film de la base Halo Infinite :
//
//	morts a l API (oracle)          100 266
//	ancienne table (dedupliquee)     98 662   98,4 %
//	passe de film                    74 569   74,4 %
//
// La vue `_latest` retenant UNE passe par match, la passe de film — plus riche par ligne
// (source du degat, assistant) mais plus pauvre en couverture — serait devenue la generation
// servie et aurait EFFACE de la lecture un quart des morts de ces matchs. Sans erreur, sans
// compteur, sans qu aucun nom ni aucun instant ne change : seulement moins de lignes.
//
// CE QUE LE RETRAIT ATTEND : que la passe de film porte l INTEGRALITE du kill-feed — le
// collecteur doit partir de la liste officielle des morts et l ENRICHIR de ce qu il sait
// decoder, au lieu de publier la seule liste qu il sait decoder. Critere mesurable :
// `lignes_passe_film / morts_api` >= le ratio de l ancienne table sur le meme perimetre.
//
// ─── POURQUOI LA TABLE CANONIQUE, ET POURQUOI C EST LE POINT ───────────────────────────────
//
// Avant elle, le meme fait — « X a tue Y a l instant T » — vivait dans DEUX tables ecrites par
// des chemins differents, plus une vue posee sur l une d elles. Ce n est pas de la redondance
// inoffensive : c est LA CAUSE du defaut le plus cher du chantier. Mesure du 2026-08-02 sur la
// base Halo Infinite : **250 139 lignes pour 133 886 couples distincts, soit 46,5 % de
// doublons**, et des agregats carriere gonfles d un facteur **2,006** — un duo qui affiche
// 101 frags la ou il y en a 29. L origine est structurelle : le flux primaire INSERT sans
// supprimer pendant que la completion combat fait DELETE-then-INSERT, et rien dans le schema
// n interdisait au meme couple d exister deux fois.
//
// `match_kill_events` ne peut pas porter ce defaut : elle est append-only et sa vue `_latest`
// ne rend QU UNE passe par match. Deux ecritures ne s additionnent plus, la seconde remplace la
// premiere. Le doublon n est pas corrige — il est devenu impossible.
//
// Disparait donc ici la vue `v_killer_victim_full` : deux LEFT JOIN morts a chaque chargement
// de vue match, qui re-joignaient `v_gamertag_lookup` pour produire deux colonnes portant DEJA
// ces noms-la — elle ne « marchait » que par le renommage silencieux des homonymes par DuckDB.
// Son unique lecteur (Q20) lit la table directement, et rend les memes six colonnes.
//
// La TABLE `killer_victim_pairs`, elle, RESTE — avec sa double ecriture, datee et justifiee
// dans `persist/shared_persister.go`. Ses ~20 lecteurs continuent de la lire, doublons compris :
// tant que la table canonique en dit moins qu elle sur les matchs a film, la remplacer serait
// echanger 46,5 % de doublons contre 25 % de morts manquantes. Le doublon se voit et se corrige ;
// la mort manquante, non.

import (
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/analysis"
)

func init() {
	Register(Migration{
		Name:     "shared_kill_events_from_pairs_v1",
		TargetDB: TargetShared,
		Description: "Reprise dedupliquee de killer_victim_pairs dans match_kill_events + " +
			"suppression de v_killer_victim_full. La table source RESTE (cf. en-tete du fichier)",
		ApplySchema: applyKillEventsFromPairs,
	})
}

// PairsBackfillDecoderRev — la version du producteur « reprise de l ancienne table ».
//
// Distincte de celle du producteur live et de celle du decodeur de film : c est elle qui permet
// de savoir, plus tard, quelles lignes viennent d une reprise et lesquelles d une mesure fraiche.
const PairsBackfillDecoderRev = "pairs-backfill-2026-08-02"

// applyKillEventsFromPairs : reprise, suppressions, vue. Idempotente — si `killer_victim_pairs`
// n est plus une table (migration deja passee), tout est deja fait.
func applyKillEventsFromPairs(db *sql.DB) error {
	if err := EnsureMatchKillEvents(db); err != nil {
		return fmt.Errorf("bascule kill events: dependance: %w", err)
	}

	estTable, err := killerVictimPairsIsTable(db)
	if err != nil {
		return err
	}
	if estTable {
		if err := reprendreCouplesDansKillEvents(db); err != nil {
			return err
		}
	}

	// v_killer_victim_full disparait : ses deux LEFT JOIN etaient du travail mort a chaque
	// chargement de vue match (elle re-joignait `v_gamertag_lookup` pour produire deux colonnes
	// qui portaient DEJA ces noms-la, et ne « marchait » que par le renommage silencieux des
	// homonymes par DuckDB). Son unique lecteur, Q20, lit desormais la table directement.
	if _, err := db.ExecContext(bootCtx(), `DROP VIEW IF EXISTS v_killer_victim_full`); err != nil {
		return fmt.Errorf("bascule kill events: %w", err)
	}
	// v_gamertag_lookup est recreee : elle lit desormais `match_kill_events_latest`, que la
	// reprise ci-dessus vient de remplir. La reposer ICI garantit qu aucune base ne garde
	// l ancienne definition apres la reprise.
	if _, err := db.ExecContext(bootCtx(), analysis.GamertagLookupViewSQL()); err != nil {
		return fmt.Errorf("bascule kill events: recreation v_gamertag_lookup: %w", err)
	}
	return nil
}

// killerVictimPairsIsTable distingue la table de la vue de compatibilite. `tableExists` ne
// suffirait pas : le catalogue DuckDB ne separe pas les deux espaces de noms, et une vue
// repondrait « oui ».
func killerVictimPairsIsTable(db *sql.DB) (bool, error) {
	var n int
	err := db.QueryRowContext(bootCtx(), `
		SELECT COUNT(*) FROM duckdb_tables() WHERE table_name = 'killer_victim_pairs'`).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("bascule kill events: nature de killer_victim_pairs: %w", err)
	}
	return n > 0, nil
}

// reprendreCouplesDansKillEvents recopie l ancienne table dans la nouvelle, DEDUPLIQUEE.
//
// TROIS PROPRIETES, chacune indispensable :
//
//  1. `SELECT DISTINCT` : c est LA deduplication. Le meme couple (match, tueur, victime,
//     instant) ecrit deux fois par deux chemins ne rend qu une mort.
//  2. UNE PASSE PAR MATCH (`decode_pass` derive du match_id) : l unite de generation de la vue
//     `_latest` est le match entier. Une passe unique pour toute la reprise ferait rendre a la
//     vue la reprise ENTIERE ou rien, et le premier decodage de film suivant ferait disparaitre
//     les couples de TOUS les autres matchs.
//  3. AUCUN MATCH DEJA COUVERT n est repris : ni par un film (sa passe porte la source du degat,
//     que cette reprise ne mesure pas), ni par le producteur live (memes lignes, deja ecrites).
//     Sans ce filtre, la reprise deviendrait la generation servie et EFFACERAIT la source du
//     degat fatal de la lecture — sans changer un seul nom ni un seul compte, donc sans symptome.
//
// Les colonnes non mesurees par le kill-feed (source, categorie, divergence, parts de degats)
// restent NULL et l assistant reste INCONNU (`assist_known = FALSE`) : ecrire « pas d assistant »
// fabriquerait un fait jamais observe.
func reprendreCouplesDansKillEvents(db *sql.DB) error {
	res, err := db.ExecContext(bootCtx(), `
		INSERT INTO match_kill_events (
			match_id, decode_pass, decoder_rev, written_at, publishable,
			time_ms, victim_gamertag, victim_xuid,
			feed_killer_gamertag, feed_killer_xuid, feed_present,
			assist_known, assist_extra_count, read_path, read_origin
		)
		SELECT DISTINCT
			kvp.match_id,
			'pairs-' || kvp.match_id,
			?, now(), TRUE,
			COALESCE(kvp.time_ms, 0),
			kvp.victim_gamertag, NULLIF(kvp.victim_xuid, ''),
			kvp.killer_gamertag, NULLIF(kvp.killer_xuid, ''), TRUE,
			FALSE, 0, ?, ?
		FROM killer_victim_pairs kvp
		WHERE kvp.victim_gamertag IS NOT NULL AND kvp.victim_gamertag <> ''
		  AND kvp.match_id NOT IN (SELECT match_id FROM match_kill_events_latest)`,
		PairsBackfillDecoderRev, readPathLiveFeed, readOriginCreditOnly)
	if err != nil {
		return fmt.Errorf("bascule kill events: reprise des couples: %w", err)
	}
	if n, errRows := res.RowsAffected(); errRows == nil {
		slog.InfoContext(bootCtx(), "bascule kill events: couples repris depuis killer_victim_pairs",
			"lignes", n, "decoder_rev", PairsBackfillDecoderRev)
	}
	return nil
}

// Portee des lignes reprises. Dupliquee depuis `persist` (que `migration` ne peut pas importer
// sans inverser la dependance) et verrouillee par un test d egalite — cf.
// steps_shared_kill_events_from_pairs_test.go.
const (
	readPathLiveFeed     = "kill-feed"
	readOriginCreditOnly = "credit-seul"
)
