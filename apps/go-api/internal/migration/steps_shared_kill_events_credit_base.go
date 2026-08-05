package migration

// steps_shared_kill_events_credit_base.go — L INVERSION DE PRESEANCE, APPLIQUEE A L HISTORIQUE.
//
// ─── CE QU ELLE FAIT, EN UNE PHRASE ────────────────────────────────────────────────────────
//
// Pour chaque match, elle ecrit UNE passe recomposee : la BASE CREDIT (les couples de
// `killer_victim_pairs`, dedupliques sur l identite) ENRICHIE de ce que la passe de film courante
// a mesure (arme, assistant nomme, parts de degats, divergence), plus les lignes de film que le
// credit ne peut pas representer (les orphelins).
//
// ─── POURQUOI UNE MIGRATION ET PAS UN BACKFILL ─────────────────────────────────────────────
//
// ELLE NE DEMANDE AUCUN FILM. L enrichissement des 949 matchs decodes est DEJA en base, indexe par
// `(match_id, time_ms)` : la recomposition est SQL -> SQL. Ni les 23 Go de films en cache, ni les
// 3 a 11 heures de decodage, ni un token. C est ce qui la rend jouable en production telle quelle,
// et c est l argument operationnel qui decide de sa forme.
//
// Le backfill par la CLI (`levelup backfill-killsource`) reste utile pour les matchs dont le film
// n a JAMAIS ete decode — il n est PAS un prealable a cette bascule.
//
// ─── L APPARIEMENT : CLEF `(match_id, time_ms)`, TOLERANCE ZERO ────────────────────────────
//
// Les deux cotes sont le MEME FLUX lu a deux endroits (meme parseur `ParseHighlightEvents`), donc
// il n y a aucun ecart d horloge a absorber : passer de 0 ms a 1 SECONDE ne gagne que 8 lignes sur
// 74 569. Et toute tolerance non nulle DETRUIT la bijection — a 0 ms, 73 589 lignes de film
// appariees pour 73 589 morts de credit appariees, au chiffre pres ; des 50 ms les deux nombres
// divergent, signature d une ligne de film qui capture plusieurs morts.
//
// L identite est un CONTROLE, pas un critere de jointure : elle doit CONCORDER quand les deux
// cotes la portent, et son absence d un cote (631 victimes, 754 tueurs que le film n a pas
// resolus) ne doit rien empecher.
//
// ─── CE QU ELLE NE PEUT PAS FAIRE PERDRE ───────────────────────────────────────────────────
//
// CHAQUE mort de la base credit produit EXACTEMENT une ligne, enrichie ou non. L invariant « une
// passe ne porte jamais moins de morts que la base credit du match » est donc vrai PAR
// CONSTRUCTION ici — et il est verifie APRES l ecriture (cf. [verifierPlancherReprise]), parce
// qu un invariant tenu par construction est un invariant qu on ne relit plus.

import (
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/observability"
)

func init() {
	Register(Migration{
		Name:     "shared_kill_events_credit_base_v1",
		TargetDB: TargetShared,
		Description: "Inversion de preseance credit<->film : recompose chaque passe = base credit " +
			"(killer_victim_pairs deduplique sur l identite) + enrichissement de la passe de film " +
			"courante + orphelins de film conserves",
		ApplySchema: applyKillEventsCreditBase,
	})
}

// CreditBaseRebuildDecoderRev — la revision du producteur « recomposition credit-base ».
//
// Elle n est portee QUE par les matchs SANS film : ceux qui en ont un gardent la revision du
// DECODEUR, parce que c est elle qui commande le redecodage et sur elle que
// `levelup backfill-killsource` decide qu un match est a jour. Ecraser cette revision ferait
// redecoder tout le corpus pour rien.
const CreditBaseRebuildDecoderRev = "credit-base-2026-08-03"

// creditBasePassPrefix — le prefixe du `decode_pass` de cette reprise. Il sert AUSSI de marque
// d idempotence : un match qui porte deja une passe ainsi prefixee n est pas repris.
const creditBasePassPrefix = "creditbase-"

// Compteurs de sante de la reprise (ADR 0009).
const (
	metricRepriseBaseMatchs   = "killsource_reprise_base_matchs"
	metricRepriseBaseLignes   = "killsource_reprise_base_lignes"
	metricRepriseBaseDiverge  = "killsource_reprise_base_identites_divergentes"
	metricRepriseBaseAmbiguus = "killsource_reprise_base_instants_ambigus"
)

// applyKillEventsCreditBase : controle, reprise, verification. Idempotente.
func applyKillEventsCreditBase(db *sql.DB) error {
	if err := EnsureMatchKillEvents(db); err != nil {
		return fmt.Errorf("reprise credit-base: dependance: %w", err)
	}
	estTable, err := killerVictimPairsIsTable(db)
	if err != nil {
		return err
	}
	if !estTable {
		// `killer_victim_pairs` est devenue une vue (etape 7 du chemin de retrait) : la base
		// credit vit desormais dans la table canonique, il n y a plus rien a reprendre.
		slog.InfoContext(bootCtx(), "reprise credit-base: killer_victim_pairs n est plus une table, "+
			"aucune reprise a faire")
		return nil
	}

	compterAnomaliesReprise(db)
	return reprendreEnBaseCredit(db)
}

// ctesReprise : les CTE partagees par le controle, l INSERT et la verification.
//
// ⚠ LA DEDUPLICATION PORTE SUR L IDENTITE `(time_ms, victime, tueur)`, PAS SUR LA CLEF NI SUR
// TOUTES LES COLONNES, et c est une mesure du 2026-08-03 qui l impose : dedupliquee sur
// `(match_id, time_ms)` seul, la base PERD 7 morts REELLES sur Halo 5 (deux victimes distinctes a
// la meme milliseconde) ; dedupliquee sur toutes les colonnes, elle FABRIQUE 12 088 doublons sur
// Halo Infinite (une meme mort ecrite sous deux orthographes de gamertag). Seule l identite par
// xuids rend les deux cardinaux justes : 133 886 et 268 337.
//
// Le gamertag est choisi par `MAX` quand une identite en porte plusieurs — deterministe, donc
// reproductible d une execution a l autre.
const ctesReprise = `
	WITH base AS (
		SELECT kvp.match_id AS match_id,
		       COALESCE(kvp.time_ms, 0) AS t,
		       COALESCE(kvp.victim_xuid, '') AS vx,
		       MAX(kvp.victim_gamertag) AS vg,
		       COALESCE(kvp.killer_xuid, '') AS kx,
		       MAX(COALESCE(kvp.killer_gamertag, '')) AS kg
		FROM killer_victim_pairs kvp
		WHERE kvp.victim_gamertag IS NOT NULL AND kvp.victim_gamertag <> ''
		GROUP BY 1, 2, 3, 5
	),
	film AS (
		SELECT * FROM match_kill_events_latest WHERE read_path IN (?, ?)
	),
	base_n AS (SELECT match_id, t, COUNT(*) AS n FROM base GROUP BY 1, 2),
	film_n AS (SELECT match_id, time_ms AS t, COUNT(*) AS n FROM film GROUP BY 1, 2),
	appariable AS (
		SELECT f.* FROM film f
		JOIN base_n bn ON bn.match_id = f.match_id AND bn.t = f.time_ms AND bn.n = 1
		JOIN film_n fn ON fn.match_id = f.match_id AND fn.t = f.time_ms AND fn.n = 1
	),
	passe AS (
		SELECT match_id, MAX(decoder_rev) AS rev, BOOL_AND(publishable) AS publiable
		FROM film GROUP BY 1
	),
	aprendre AS (
		SELECT DISTINCT match_id FROM killer_victim_pairs
		WHERE match_id NOT IN (
			SELECT match_id FROM match_kill_events WHERE decode_pass LIKE '` +
	creditBasePassPrefix + `%'
		)
	)`

// argsReprise : les deux voies de film, dans l ordre des `?` de [ctesReprise].
//
// LUES CHEZ LEUR PROPRIETAIRE, jamais recopiees : `migration` ne peut importer ni `persist` (cycle
// a la compilation de ses tests) ni `killsource` (title-specific), mais `domain/killscope` est une
// feuille sans import — c est exactement pour ce cas qu elle existe.
func argsReprise() []any {
	return []any{killscope.ReadPathFilmWalk, killscope.ReadPathFilmScan}
}

// compterAnomaliesReprise : les deux populations qui ne vont pas de soi, comptees AVANT d ecrire.
//
// ─── LES IDENTITES DIVERGENTES ─────────────────────────────────────────────────────────────
//
// Quand les deux cotes portent un xuid, il doit etre le meme (0 divergence sur 73 589 lignes
// appariees, mesure du 2026-08-02). Une divergence signifierait que la clef `(match_id, time_ms)`
// a apparie deux morts differentes — c est-a-dire que la propriete sur laquelle repose toute cette
// reprise est fausse.
//
// ⚠ POURQUOI ELLE N ARRETE PAS LA MIGRATION, LA OU LE FUSIONNEUR EN MEMOIRE REND UNE ERREUR :
// cette migration tourne AU BOOT, sur toutes les bases. Une erreur y bloquerait le demarrage de
// l application entiere sur une donnee qui n a jamais ete observee. Les lignes divergentes ne sont
// donc PAS enrichies (la mort de credit garde son etat credit, aucune arme n est attribuee au
// hasard, aucune mort n est perdue) et l anomalie sort en `ErrorContext` avec son compteur. Ce
// n est pas « ecarter en silence » — c est degrader en le disant. Le fusionneur en memoire
// (`persist.MergeCreditAndFilm`), lui, echoue franchement : il travaille match par match pendant
// un cycle de sync, ou l echec est rattrapable et visible.
//
// ─── LES INSTANTS AMBIGUS ──────────────────────────────────────────────────────────────────
//
// Plusieurs morts de credit ou plusieurs lignes de film au meme instant : l enrichissement est
// REFUSE la (`appariable` exige n = 1 des deux cotes). L unicite de la clef est une propriete
// MESUREE du corpus Infinite, pas une garantie de schema — Halo 5 porte 7 collisions reelles.
//
// Contrat d erreur : ces mesures sont des DIAGNOSTICS. Leur echec est journalise et n arrete pas
// la reprise ; les compteurs restent alors a zero, ce que le log explique.
func compterAnomaliesReprise(db *sql.DB) {
	divergentes := compterUn(db, ctesReprise+`
		SELECT COUNT(*) FROM base b
		JOIN appariable a ON a.match_id = b.match_id AND a.time_ms = b.t
		WHERE (b.vx <> '' AND a.victim_xuid IS NOT NULL AND a.victim_xuid <> ''
		       AND b.vx <> a.victim_xuid)
		   OR (b.kx <> '' AND a.feed_killer_xuid IS NOT NULL AND a.feed_killer_xuid <> ''
		       AND b.kx <> a.feed_killer_xuid)`, "identites divergentes")
	if divergentes > 0 {
		observability.AddInt(metricRepriseBaseDiverge, int64(divergentes))
		slog.ErrorContext(bootCtx(), "reprise credit-base: identites DIVERGENTES entre le credit et "+
			"le film au meme instant — ces morts ne sont PAS enrichies (aucune arme attribuee au "+
			"hasard, aucune mort perdue). La clef (match_id, time_ms) a apparie deux morts "+
			"differentes : l hypothese d unicite de la clef est en defaut sur cette base",
			"lignes", divergentes)
	}

	ambigus := compterUn(db, ctesReprise+`
		SELECT COUNT(*) FROM base_n bn
		JOIN film_n fn ON fn.match_id = bn.match_id AND fn.t = bn.t
		WHERE bn.n <> 1 OR fn.n <> 1`, "instants ambigus")
	if ambigus > 0 {
		observability.AddInt(metricRepriseBaseAmbiguus, int64(ambigus))
		slog.WarnContext(bootCtx(), "reprise credit-base: instants portant plusieurs morts — "+
			"enrichissement refuse sur ces instants", "instants", ambigus)
	}
}

// compterUn : un `COUNT(*)` de diagnostic. Rend 0 et journalise en cas d echec — une sonde ne doit
// pas faire tomber une migration de boot, mais son echec ne doit pas non plus etre invisible.
func compterUn(db *sql.DB, query, quoi string) int {
	var n int
	if err := db.QueryRowContext(bootCtx(), query, argsReprise()...).Scan(&n); err != nil {
		slog.WarnContext(bootCtx(), "reprise credit-base: sonde impossible, la reprise continue "+
			"sans ce diagnostic", "sonde", quoi, "err", err)
		return 0
	}
	return n
}

// insertReprise : LES DEUX BRANCHES DE LA RECOMPOSITION.
//
//	branche 1  chaque mort de la BASE CREDIT, enrichie par `appariable` quand l instant est
//	           univoque. Une mort non enrichie garde exactement l etat que le producteur credit
//	           lui donne : `assist_known = FALSE` (ON NE SAIT PAS), source/parts/divergence NULL.
//	           ⚠ `COALESCE(a.assist_known, FALSE)` est le SEUL defaut applique, et il ne
//	           s applique qu a l ABSENCE de ligne de film. Defauter a TRUE fabriquerait
//	           60 297 « mesures : pas d assistant » jamais observees.
//	branche 2  les ORPHELINS : les lignes de film dont l instant ne porte AUCUNE mort de credit.
//	           819 « un humain tue un BOT », 149 « un BOT tue un humain », 13 humain contre
//	           humain — et sur les 980, ZERO ne tombe sur un instant sans evenement de l API. Le
//	           kill-feed de l API est HUMAIN SEUL : les rejeter serait traiter une absence de
//	           mesure comme une mesure d absence.
//
// Une ligne de film dont l instant PORTE une mort de credit sans etre appariable (instant ambigu)
// n entre par AUCUNE des deux branches : la mort est deja dans la base, l ajouter la compterait
// deux fois.
const insertReprise = ctesReprise + `
	INSERT INTO match_kill_events (
		match_id, decode_pass, decoder_rev, written_at, publishable,
		time_ms, victim_gamertag, victim_xuid,
		feed_killer_gamertag, feed_killer_xuid, feed_present,
		assist_gamertag, assist_xuid, assist_known, assist_index,
		assist_rejected, assist_extra_count,
		source_tag, source_category, killer_damage_pct, assist_damage_pct,
		diverges, read_path, read_origin
	)
	SELECT b.match_id, '` + creditBasePassPrefix + `' || b.match_id, COALESCE(p.rev, ?), CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
	       COALESCE(p.publiable, TRUE),
	       b.t, b.vg, NULLIF(b.vx, ''),
	       NULLIF(b.kg, ''), NULLIF(b.kx, ''), TRUE,
	       a.assist_gamertag, a.assist_xuid, COALESCE(a.assist_known, FALSE), a.assist_index,
	       a.assist_rejected, COALESCE(a.assist_extra_count, 0),
	       a.source_tag, a.source_category, a.killer_damage_pct, a.assist_damage_pct,
	       a.diverges, COALESCE(a.read_path, ?), COALESCE(a.read_origin, ?)
	FROM base b
	JOIN aprendre ap ON ap.match_id = b.match_id
	LEFT JOIN passe p ON p.match_id = b.match_id
	LEFT JOIN appariable a ON a.match_id = b.match_id AND a.time_ms = b.t
		AND (b.vx = '' OR a.victim_xuid IS NULL OR a.victim_xuid = '' OR b.vx = a.victim_xuid)
		AND (b.kx = '' OR a.feed_killer_xuid IS NULL OR a.feed_killer_xuid = ''
		     OR b.kx = a.feed_killer_xuid)
	UNION ALL
	SELECT f.match_id, '` + creditBasePassPrefix + `' || f.match_id, COALESCE(p.rev, ?), CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
	       COALESCE(p.publiable, TRUE),
	       f.time_ms, f.victim_gamertag, f.victim_xuid,
	       f.feed_killer_gamertag, f.feed_killer_xuid, f.feed_present,
	       f.assist_gamertag, f.assist_xuid, f.assist_known, f.assist_index,
	       f.assist_rejected, f.assist_extra_count,
	       f.source_tag, f.source_category, f.killer_damage_pct, f.assist_damage_pct,
	       f.diverges, f.read_path, f.read_origin
	FROM film f
	JOIN aprendre ap ON ap.match_id = f.match_id
	LEFT JOIN passe p ON p.match_id = f.match_id
	LEFT JOIN base_n bn ON bn.match_id = f.match_id AND bn.t = f.time_ms
	WHERE bn.match_id IS NULL`

// reprendreEnBaseCredit joue la recomposition, puis relit l invariant.
func reprendreEnBaseCredit(db *sql.DB) error {
	args := argsReprise()
	args = append(args, CreditBaseRebuildDecoderRev, killscope.ReadPathLiveFeed,
		killscope.OriginCreditOnly)
	args = append(args, argsReprise()...)
	args = append(args, CreditBaseRebuildDecoderRev)

	res, err := db.ExecContext(bootCtx(), insertReprise, args...)
	if err != nil {
		return fmt.Errorf("reprise credit-base: recomposition: %w", err)
	}
	lignes, _ := res.RowsAffected()
	observability.AddInt(metricRepriseBaseLignes, lignes)

	var matchs int
	if err := db.QueryRowContext(bootCtx(),
		`SELECT COUNT(DISTINCT match_id) FROM match_kill_events WHERE decode_pass LIKE ?`,
		creditBasePassPrefix+"%").Scan(&matchs); err != nil {
		slog.WarnContext(bootCtx(), "reprise credit-base: comptage des matchs repris impossible, "+
			"la reprise reste acquise", "err", err)
	}
	observability.AddInt(metricRepriseBaseMatchs, int64(matchs))
	slog.InfoContext(bootCtx(), "reprise credit-base: passes recomposees",
		"lignes", lignes, "matchs", matchs, "decoder_rev_sans_film", CreditBaseRebuildDecoderRev)

	return verifierPlancherReprise(db)
}

// verifierPlancherReprise : L INVARIANT, RELU APRES COUP.
//
// « Une passe ne peut jamais porter moins de morts que la base credit de son match. » Il est vrai
// par construction ici (chaque mort de la base produit exactement une ligne) — et c est
// precisement pour ca qu il se relit : un invariant tenu par construction est un invariant que
// personne ne verifie plus, jusqu au jour ou une clause `WHERE` ajoutee de bonne foi le casse.
//
// C est l invariant dont l ABSENCE a coute la session 3 : la passe de film remplacait la passe
// credit en n en publiant que 74,4 %, sans erreur ni compteur.
func verifierPlancherReprise(db *sql.DB) error {
	var manquants, pire int
	err := db.QueryRowContext(bootCtx(), ctesReprise+`
		, publiees AS (
			SELECT match_id, COUNT(*) AS n FROM match_kill_events_latest GROUP BY 1
		)
		SELECT COUNT(*), COALESCE(MAX(bn.n - COALESCE(pu.n, 0)), 0)
		FROM (SELECT match_id, SUM(n) AS n FROM base_n GROUP BY 1) bn
		LEFT JOIN publiees pu ON pu.match_id = bn.match_id
		WHERE COALESCE(pu.n, 0) < bn.n`, argsReprise()...).Scan(&manquants, &pire)
	if err != nil {
		return fmt.Errorf("reprise credit-base: verification du plancher: %w", err)
	}
	if manquants > 0 {
		return fmt.Errorf("reprise credit-base: %d match(s) publient MOINS de morts que leur base "+
			"credit (jusqu a %d morts manquantes). La vue _latest ne retient qu une passe par "+
			"match : l ecart serait EFFACE de la lecture sans erreur ni compteur", manquants, pire)
	}
	return nil
}
