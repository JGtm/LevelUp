// Package duckdb — kill_measured.go : LA JOINTURE « MORT MESURÉE », ET IL N'Y EN A QU'UNE.
//
// # CE QU'ELLE EST
//
// Une mort mesurée est une mort dont on connaît À LA FOIS l'arme (`source_tag` du kill-feed
// reconstruit) et la position des DEUX joueurs (table de positions). Elle porte donc une
// distance et un dénivelé, et c'est la brique de tout ce qui parle de portée : le POC
// « distance par arme » d'un match (KillDistanceRepo) comme l'agrégat multi-matchs de la
// Synthèse (WeaponRangeRepo).
//
// # POURQUOI CE FICHIER EXISTE (règle n°6 du dépôt)
//
// La même jointure `match_kill_events_latest × <positions>_latest` — mêmes gardes, mêmes
// NULL-checks, même unanimité — était écrite une fois dans le POC G.3 ; le lot 3 en aurait
// fait une TROISIÈME et une QUATRIÈME copie (côté tueur, côté victime). À la troisième copie
// on centralise ET on pose le garde-rail : `kill_measured_guard_test.go` interdit désormais
// la jointure ailleurs qu'ici.
//
// # LES DEUX GARDES, ET POURQUOI ELLES NE SONT PAS NÉGOCIABLES
//
//	publishable        — cette lecture est PAR KILL (on nomme une arme ET une distance).
//	                     Une passe non publiable est juste en AGRÉGAT et fausse
//	                     individuellement : elle est écartée, contrairement à ce que
//	                     KillSourceClassRepo tolère pour ses comptages.
//	unanimité          — `HAVING count(DISTINCT e.source_tag) = 1` : un double kill au même
//	                     (tueur, instant) qui ne s'accorde pas sur l'arme ne publie RIEN.
//	                     Accrocher une position à la mauvaise arme serait indétectable à
//	                     l'écran. Même doctrine que Q21b (queries_match.go).
//
// # LE GROUPEMENT SUIT LA CLÉ DE LA TABLE DE POSITIONS, PAS L'INVERSE
//
// `kill_positions` (et sa sœur `kill_openings`) est clé par (match_id, killer_xuid, time_ms) :
// un double kill au même instant n'y a qu'UNE ligne. Le GROUP BY reprend donc cette clé —
// ce n'est pas un choix d'agrégation, c'est la forme de la donnée. Conséquence assumée,
// héritée du POC et laissée telle quelle : un double kill à la MÊME arme compte pour une
// mesure, pas deux.
//
// La clause `WHERE` s'applique AVANT le groupement. Une lecture côté victime
// (`e.victim_xuid = ?`) ne voit donc que la ligne de CETTE victime, et l'unanimité s'y juge
// sur elle seule — c'est plus fin que la lecture côté tueur, et c'est correct : la mort
// dont on parle est identifiée, il n'y a plus d'ambiguïté à lever.
//
// # AUCUNE DISTANCE N'EST STOCKÉE (doctrine G.0)
//
// La base porte des COORDONNÉES ; la distance et le dénivelé se calculent ICI, à la lecture.
// On stocke une mesure, pas une résolution améliorable.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
)

// measuredPositionsTable : LA table de positions sur laquelle la jointure se fait.
//
// Deux instants, deux tables, une seule jointure — c'est la raison d'être du paramètre
// (item 3.12 du plan) : la distance À LA MORT et la distance À L'ENTAME se lisent avec
// exactement les mêmes gardes, et une seconde requête recopiée finirait par diverger sur
// l'une des deux.
type measuredPositionsTable string

const (
	// positionsAtKill — les positions à l'instant du coup fatal.
	positionsAtKill measuredPositionsTable = "kill_positions_latest"
	// positionsAtOpening — les positions un temps-pour-tuer AVANT le coup fatal (proxy
	// d'entame, `replay.OpeningLeadMS`). La table porte le time_ms DU KILL, pas l'instant
	// décalé : c'est ce qui permet à cette jointure d'être la même.
	positionsAtOpening measuredPositionsTable = "kill_openings_latest"
)

// killMeasured : UNE mort mesurée, avant toute traduction de sens (l'arme est encore un
// `source_tag`, le côté n'est pas encore choisi).
type killMeasured struct {
	// matchID, killerXUID, timeMS forment LA CLÉ DU FRAG — celle par laquelle un appelant
	// apparie la mesure à la mort (et l'entame au coup fatal, cf. lot 4).
	matchID    string
	killerXUID string
	timeMS     int64
	// victimXUID peut être vide : un bot n'a pas de xuid, et la mort reste mesurable.
	victimXUID string
	// sourceTag est la source du dégât, à traduire par un port.KillSourceClassifier.
	sourceTag uint32
	// distanceM est la distance 3D tueur <-> victime, en mètres.
	distanceM float64
	// deltaZ est le dénivelé BRUT `killer_z - victim_z`, en mètres — LA GRANDEUR PHYSIQUE,
	// SANS POINT DE VUE. Le repo ne l'inverse JAMAIS, pas même pour une lecture côté
	// victime : c'est `analysis.WeaponRangeAggregate` qui la ramène au point de vue du côté
	// demandé, et deux inversions s'annuleraient (convention tranchée au lot 2).
	deltaZ float64
}

// measuredKillsQuery compose la requête des morts mesurées pour une table de positions et
// une clause `WHERE` données. La clause est du SQL du dépôt (jamais une entrée utilisateur) ;
// ses valeurs voyagent en paramètres liés, pas dans le texte.
func measuredKillsQuery(table measuredPositionsTable, where string) string {
	return fmt.Sprintf(measuredKillsSQLTemplate, string(table), where)
}

// measuredKillsSQLTemplate : %s = la table de positions, %s = la clause de portée.
//
// Les six NULL-checks ne sont pas décoratifs : une ligne de positions PARTIELLE (un seul
// côté localisé) existe réellement en base, et une distance calculée sur un côté manquant
// serait un nombre plausible et faux. Elle est écartée, jamais approchée.
const measuredKillsSQLTemplate = `
SELECT
    e.match_id,
    e.feed_killer_xuid,
    e.time_ms,
    min(e.victim_xuid) AS victim_xuid,
    min(e.source_tag) AS source_tag,
    min(kp.killer_x) AS killer_x, min(kp.killer_y) AS killer_y, min(kp.killer_z) AS killer_z,
    min(kp.victim_x) AS victim_x, min(kp.victim_y) AS victim_y, min(kp.victim_z) AS victim_z
FROM match_kill_events_latest e
JOIN %s kp
    ON kp.match_id = e.match_id
   AND kp.killer_xuid = e.feed_killer_xuid
   AND kp.time_ms = e.time_ms
WHERE %s
  AND e.publishable
  AND e.source_tag IS NOT NULL
  AND e.feed_killer_xuid IS NOT NULL
  AND kp.killer_x IS NOT NULL AND kp.killer_y IS NOT NULL AND kp.killer_z IS NOT NULL
  AND kp.victim_x IS NOT NULL AND kp.victim_y IS NOT NULL AND kp.victim_z IS NOT NULL
GROUP BY e.match_id, e.feed_killer_xuid, e.time_ms
HAVING count(DISTINCT e.source_tag) = 1`

// queryMeasuredKills exécute une requête composée par [measuredKillsQuery] et rend les
// morts mesurées. `scope` n'a d'autre rôle que de nommer l'appelant dans les journaux —
// un « scan failed » sans son lecteur est illisible en production.
//
// LE SQL NE CONNAÎT QUE DES NOMBRES : distance et dénivelé se calculent en Go, comme dans
// tous les lecteurs de cette famille (aucune traduction de sens n'a lieu en SQL).
func queryMeasuredKills(
	ctx context.Context, db *sql.DB, query string, args []any, scope string,
) ([]killMeasured, error) {
	dbRows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer dbRows.Close()

	out := make([]killMeasured, 0)
	for dbRows.Next() {
		m, err := scanMeasuredKill(dbRows)
		if err != nil {
			// Une ligne illisible est une anomalie de schéma, pas un cas nominal : on la
			// signale AVANT de dégrader, on ne l'avale jamais en silence.
			slog.ErrorContext(ctx, "kill_measured: scan failed, row skipped", "scope", scope, "err", err)
			continue
		}
		out = append(out, m)
	}
	if err := dbRows.Err(); err != nil {
		slog.ErrorContext(ctx, "kill_measured: rows iteration failed", "scope", scope, "err", err)
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}

// scanMeasuredKill lit UNE ligne et calcule ses deux grandeurs dérivées.
func scanMeasuredKill(dbRows *sql.Rows) (killMeasured, error) {
	var (
		m                         killMeasured
		victim                    sql.NullString
		killerX, killerY, killerZ float64
		victimX, victimY, victimZ float64
	)
	if err := dbRows.Scan(&m.matchID, &m.killerXUID, &m.timeMS, &victim, &m.sourceTag,
		&killerX, &killerY, &killerZ, &victimX, &victimY, &victimZ); err != nil {
		return killMeasured{}, err
	}
	m.victimXUID = victim.String
	m.distanceM = hypot3D(killerX, killerY, killerZ, victimX, victimY, victimZ)
	m.deltaZ = killerZ - victimZ
	return m, nil
}

// hypot3D : distance euclidienne entre deux points de l'espace monde (mètres). UNIQUE
// formule de distance du paquet.
func hypot3D(x1, y1, z1, x2, y2, z2 float64) float64 {
	dx, dy, dz := x1-x2, y1-y2, z1-z2
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}
