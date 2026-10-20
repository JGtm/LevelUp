// Package ops — seed_demo_replay_vehicles.go : la famille de rejeu « véhicules » de la démo,
// le match 4 contre 4 où l'on monte le plus de types de véhicules (décision D6 du plan de
// release v8.0.0, 2026-10-17).
//
// CONTRAIREMENT AUX FAMILLES DE MODE, elle ne se reconnaît PAS au nom de la variante : un
// match de n'importe quel mode géré y prétend dès que ses données le disent. La règle, lue en
// base partagée, déterministe pour un même état du dépôt :
//
//   - 8 joueurs au coup d'envoi, 4 contre 4 : les participants `present_at_beginning` (bots
//     compris, ils tiennent un slot), répartis en exactement deux `team_id` de 4. Un match
//     dont AUCUN participant n'a le drapeau renseigné (non rattrapé) retombe sur tous ses
//     participants, comme `concurrentTeamSize` (sync/skill) — un remplaçant tardif ne
//     transforme pas un 4v4 en 5v4 ;
//   - au moins [demoVehicleMinTypes] FAMILLES de véhicules distinctes montées, lues dans la
//     dernière passe entière de `match_vehicle_takes_latest` (ADR 0026 : la vue seulement),
//     lignes `take` d'une passe mesurée. La famille « inconnue » (châssis hors table) n'est pas
//     un type et ne compte pas ;
//   - départage : le plus de montées (somme de `takes`), puis le plus récent, puis le plus
//     petit match_id ([betterVehicleDemoReplay]). Pas de priorité à l'artefact à jour :
//     la cuisson de la démo se fait en local (D7).
//
// Les exclusions des autres familles s'appliquent (mode non géré, joueur source présent, film
// au cache : [loadDemoReplayCandidates]). Le match retenu peut être aussi le meilleur de sa
// famille de mode : les deux entrées coexistent au manifeste (le seed range le film une fois).
package ops

import (
	"fmt"

	"levelup/go-api/internal/persist"
)

// demoModeVehicles : clé de manifeste de la famille « véhicules » (identifiant, jamais un
// libellé). STABLE : une famille déjà figée sous cette clé garde son match (émission additive).
const demoModeVehicles = "vehicles"

// Seuils de la règle, nommés.
const (
	// demoVehicleTeams et demoVehicleTeamSize : un 4 contre 4 compte deux camps de quatre
	// joueurs au coup d'envoi, soit 8 joueurs.
	demoVehicleTeams    = 2
	demoVehicleTeamSize = 4
	// demoVehicleMinTypes : familles de véhicules distinctes montées au minimum.
	demoVehicleMinTypes = 2
)

// demoVehicleFamilyUnknown : la famille « châssis hors table » de `match_vehicle_takes`. Valeur
// de donnée écrite par la projection des prises (constante VehicleFamilyUnknown du paquet
// film/replay) ; reprise ici parce que la façade du décodeur ne l'exporte pas et que le
// ratchet de surface (archlint/film_facade_surface_test.go) interdit d'y ajouter un symbole
// hors de film/.
const demoVehicleFamilyUnknown = "unknown"

// qDemoReplayVehicleCTEs : les trois clauses WITH de la règle, devant la requête des
// candidats. Aucun paramètre SQL ; les quatre %d (camps, camps, joueurs par camp, joueurs par
// camp) et les deux %s (nature de ligne, famille inconnue) sont posés par
// [vehicleDemoReplayClauses]. `starters` pose le drapeau de présence au coup d'envoi avec le
// repli par match, et ne regarde que les matchs qui portent des prises : « 4 contre 4 » n'est
// donc établi que pour eux (c'est tout ce que la famille consulte, et le calcul reste borné).
const qDemoReplayVehicleCTEs = `
WITH vehicles AS (
	SELECT match_id, CAST(COUNT(DISTINCT family) AS BIGINT) AS types, CAST(SUM(takes) AS BIGINT) AS takes
	FROM match_vehicle_takes_latest
	WHERE row_kind = '%s' AND measured AND family <> '%s'
	GROUP BY match_id
),
starters AS (
	SELECT match_id, team_id FROM (
		SELECT p.match_id, p.team_id,
		       CASE WHEN COUNT(p.present_at_beginning) OVER (PARTITION BY p.match_id) > 0
		            THEN COALESCE(p.present_at_beginning, FALSE) ELSE TRUE END AS starter
		FROM match_participants p
		WHERE p.match_id IN (SELECT match_id FROM vehicles)
	) WHERE starter
),
four_v_four AS (
	SELECT match_id FROM (
		SELECT match_id, team_id, COUNT(*) AS n FROM starters GROUP BY match_id, team_id
	) GROUP BY match_id
	HAVING COUNT(*) = %d AND COUNT(team_id) = %d AND MIN(n) = %d AND MAX(n) = %d
)`

// qDemoReplayVehicleSelect : les colonnes de la règle ajoutées au SELECT des candidats, et les
// jointures qui les portent (alias de la table du registre : r).
const qDemoReplayVehicleSelect = `,
       f.match_id IS NOT NULL AS four_v_four, COALESCE(v.types, 0) AS vehicle_types,
       COALESCE(v.takes, 0) AS vehicle_takes`

const qDemoReplayVehicleJoins = `
LEFT JOIN four_v_four f ON f.match_id = r.match_id
LEFT JOIN vehicles v ON v.match_id = r.match_id`

// vehicleDemoReplayClauses rend les clauses WITH de la règle, effectifs posés.
func vehicleDemoReplayClauses() string {
	return fmt.Sprintf(qDemoReplayVehicleCTEs, persist.VehicleRowKindTake, demoVehicleFamilyUnknown,
		demoVehicleTeams, demoVehicleTeams, demoVehicleTeamSize, demoVehicleTeamSize)
}

// isVehicleDemoReplay : le candidat remplit-il la règle « 4 contre 4, au moins deux types de
// véhicules montés » ?
func (c demoReplayCandidate) isVehicleDemoReplay() bool {
	return c.fourVsFour && c.vehicleTypes >= demoVehicleMinTypes
}

// betterVehicleDemoReplay : a passe-t-il devant b pour la famille « véhicules » ? Plus de
// montées, puis plus récent, puis plus petit match_id.
func betterVehicleDemoReplay(a, b demoReplayCandidate) bool {
	if a.vehicleTakes != b.vehicleTakes {
		return a.vehicleTakes > b.vehicleTakes
	}
	if !a.start.Equal(b.start) {
		return a.start.After(b.start)
	}
	return a.matchID < b.matchID
}
