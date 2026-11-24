package killcollector

// postsync_requetes.go — les requetes de l arriere de la passe : la page de travail et la jauge,
// sur un seul predicat.

import "levelup/go-api/internal/platform/duckdb/sqlfrag"

// matchsSansPasseAJour : les matchs du registre dont la passe COURANTE ne porte pas la
// revision de decodeur courante sur une voie de FILM, du plus vieux au plus recent.
//
// ⚠ LECTURE PAR LA VUE `_latest` (ADR 0026) : une lecture brute servirait des passes
// perimees et ferait sauter des matchs qui ont besoin d etre redecodes.
//
// Le filtre `read_path <> credit-backfill` est ce qui rend l etape UTILE : un match couvert
// par le producteur credit-seul reste candidat, parce que c est precisement le credit qui ne
// sait rien de l assistant. Sans ce filtre, l etape se croirait a jour sur toute la base et
// l attribution ne repartirait jamais.
//
// `start_time_utc` par le COALESCE canonique (regle 8) — un `start_time` brut trierait faux.
//
// C est la taille d une PAGE de lecture, pas la borne de TRAVAIL (c est `perCycle`, avec le budget de
// temps) : un cycle lit autant de pages qu il en faut pour remplir sa liste de travail, dans la limite
// de [PostSyncBacklogPagesMax]. La TAILLE du backlog reste comptee separement, sans borne — un
// compteur tronque mentirait sur le retard.
const PostSyncBacklogHorizon = 64

// conditionBacklog : le predicat commun a la liste et a la jauge. UNE seule copie, parce que
// deux copies divergent et la jauge se met alors a decrire un autre ensemble que le travail.
//
// QUATRE CONDITIONS, QUATRE RAISONS — les retirer casse l etape en silence :
//
//	match_kill_events_latest        la VUE, jamais la table (ADR 0026) : une passe perimee
//	                                ferait sauter un match qui a besoin d etre redecode.
//	read_path <> credit-backfill    c est CE filtre qui rend l etape utile. Toute la base est
//	                                couverte par le producteur credit-seul, qui porte pourtant
//	                                la revision courante : sans lui, l etape se croirait a jour
//	                                partout et l assistant ne se resoudrait plus jamais.
//	backfill_completed & no_film    LE MARQUEUR TERMINAL, ET IL EXISTAIT DEJA. L etape 1.55
//	                                (weapon kills) tourne AVANT celle-ci sur le meme film et
//	                                pose `MBitWeaponKillsNoFilm` quand 343 ne le sert plus.
//	                                Sans cette exclusion, un film expire ne produit aucune
//	                                ligne, donc le match reste au backlog A VIE : mesure du
//	                                2026-08-29, 581 des 999 candidats sont dans ce cas, tous
//	                                anterieurs a 2026. Ils occupaient toute la liste de travail
//	                                et les 415 matchs de 2026 n etaient JAMAIS atteints.
//	killsource_sans_killfeed_rev    le film COMPLET a ete decode sous la revision courante sans
//	                                aucun kill : il n ecrit aucune ligne, donc sans cette
//	                                condition il reste candidat a vie. Une REVISION et non un
//	                                bit : une nouvelle revision le rend de nouveau candidat.
//	                                Mesure du 2026-10-02 : huit films de ~55 s redecodes a chaque
//	                                cycle occupaient les huit places, backlog fige a 7 351.
const conditionBacklog = `
		FROM match_registry r
		WHERE COALESCE(r.backfill_completed, 0) & ? = 0
		  AND r.killsource_sans_killfeed_rev IS DISTINCT FROM ?
		  AND NOT EXISTS (
			SELECT 1 FROM match_kill_events_latest e
			WHERE e.match_id = r.match_id
			  AND e.decoder_rev = ?
			  AND e.read_path <> ?
		)`

// requeteBacklog : les identifiants a traiter, DU PLUS RECENT AU PLUS VIEUX.
//
// L ORDRE A ETE INVERSE LE 2026-08-29, ET LE RAISONNEMENT D ORIGINE ETAIT FAUX. Il disait
// « les films expirent, sauvons d abord les plus vieux ». Mais un film deja expire ne se sauve
// pas : il est perdu, et le depot le SAIT (bit no-film ci-dessus). Trier du plus vieux au plus
// recent revenait donc a offrir toute la liste de travail a des matchs irrecuperables. Les
// recents sont a la fois les seuls recuperables et ceux que l utilisateur regarde.
var requeteBacklog = `SELECT r.match_id` + conditionBacklog + `
		ORDER BY ` + sqlfrag.SQLStartTimeCanonical("r") + ` DESC, r.match_id
		LIMIT ? OFFSET ?`

// requeteBacklogTaille : la jauge, SANS borne. Elle repond a « combien reste-t-il », pas a
// « qu est-ce que je traite maintenant ».
var requeteBacklogTaille = `SELECT COUNT(*)` + conditionBacklog
