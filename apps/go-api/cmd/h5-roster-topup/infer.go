// infer.go — logique PURE de reconstruction du roster H5 (aucun accès DB).
//
// Le cœur est inferDroppedTeam : 2-coloration du graphe de kills. En Halo, un
// kill oppose (quasi toujours) deux camps ADVERSES. Donc l'équipe d'un joueur
// droppé D se déduit de l'équipe MAJORITAIRE de ses antagonistes connus : si la
// plupart des joueurs que D a tués / qui ont tué D sont de l'équipe T, alors D
// est dans l'équipe ≠ T (parmi les deux camps présents).
//
// Isolé du main pour être testable sans DB (cf. infer_test.go).
package main

// teamUnknown sentinelle pour un joueur dont l'équipe n'est pas (encore) connue.
const teamUnknown = -1

// knownParticipant : un joueur PERSISTÉ dans match_participants (identité résolue).
type knownParticipant struct {
	XUID    string
	TeamID  int
	Outcome int
}

// droppedPlayer : un joueur présent au kill-feed mais ABSENT de match_participants.
// Kills/Deaths sont les agrégats reconstruits depuis killer_victim_pairs.
type droppedPlayer struct {
	XUID     string
	Gamertag string
	Kills    int
	Deaths   int
	// InferredTeam / InferredOutcome remplis par inferDroppedRoster (teamUnknown
	// si indéterminable).
	InferredTeam    int
	InferredOutcome int
}

// killEdge : une interaction directe killer→victim (1 par kill ; agrégée si la
// source compte plusieurs kills sur la même paire).
type killEdge struct {
	KillerXUID string
	VictimXUID string
	Weight     int // nb de kills (SUM kill_count) — robuste à l'agrégat par-paire
}

// inferDroppedTeam infère l'équipe d'un joueur droppé par vote majoritaire sur
// les équipes CONNUES de ses antagonistes (kill-graph 2-coloration).
//
//   - teamByXUID : équipe connue (résolue) pour chaque joueur déjà identifié.
//     Les joueurs droppés ne doivent PAS y figurer (équipe inconnue).
//   - teamA, teamB : les deux team_id présents dans le match.
//   - edges : toutes les arêtes de kill du match.
//
// Retourne (teamID, true) si une majorité d'antagonistes à équipe connue penche
// d'un côté ; (teamUnknown, false) si D n'a AUCUNE interaction avec un joueur
// d'équipe connue (que des interactions entre droppés) → indéterminable.
//
// En cas d'égalité parfaite des votes des deux camps adverses (rare, kill-feed
// bruité), on renvoie indéterminable plutôt que de deviner.
func inferDroppedTeam(d string, teamByXUID map[string]int, teamA, teamB int, edges []killEdge) (int, bool) {
	// Pour chaque antagoniste (peu importe le sens du kill), on accumule le poids
	// par équipe connue de l'antagoniste. L'équipe de D = l'AUTRE camp que celui
	// qui domine ses antagonistes.
	oppWeight := map[int]int{} // team connu de l'antagoniste -> poids cumulé
	for _, e := range edges {
		var other string
		switch d {
		case e.KillerXUID:
			other = e.VictimXUID
		case e.VictimXUID:
			other = e.KillerXUID
		default:
			continue // arête ne concernant pas D
		}
		t, ok := teamByXUID[other]
		if !ok {
			continue // antagoniste à équipe inconnue (autre droppé) → ignoré
		}
		oppWeight[t] += e.Weight
	}

	wA := oppWeight[teamA]
	wB := oppWeight[teamB]
	if wA == 0 && wB == 0 {
		return teamUnknown, false // aucun antagoniste à équipe connue
	}
	switch {
	case wA > wB:
		// Antagonistes majoritairement teamA → D est dans teamB.
		return teamB, true
	case wB > wA:
		return teamA, true
	default:
		// Égalité parfaite → on n'invente pas.
		return teamUnknown, false
	}
}

// inferDroppedRoster infère l'équipe + l'outcome de chaque joueur droppé, puis
// vérifie l'équilibre final des effectifs.
//
//   - knowns : participants déjà persistés (équipe + outcome connus).
//   - dropped : joueurs à reconstruire (mutés en place : InferredTeam/Outcome).
//   - edges : arêtes de kill du match.
//
// Retourne :
//   - reconstructible : les droppés dont l'équipe a pu être inférée ET qui, une
//     fois ajoutés, laissent un roster valide : total <= 8 ET |A-B| <= 1.
//   - residualReason != "" si le match doit partir en re-fetch (résidu) :
//     équipe indéterminable pour au moins un droppé, OU roster reconstruit
//     > 8 / déséquilibré.
//
// Cap roster (anti BUG 2) : on n'accepte la reconstruction QUE si
// (resA + resB) <= 8 ET |resA - resB| <= 1 (resX = connusX + droppésX). Cela
// RÉSERVE l'offline aux matchs Arena propres (≤ 4v4). Les BTB (8v8 = 16) et les
// matchs à SUBSTITUTIONS (un joueur quitte, un autre prend le slot → kill-feed
// distinct > effectif concurrent → rosters > 8, ex. 5v6 = 11) partent en résidu
// pour un re-fetch carnage précis (phase ultérieure). On ne devine NI le mode NI
// un cap par mode : la règle total<=8 + |A-B|<=1 suffit.
//
// Note : un match est traité en TOUT-ou-RIEN. Si UN seul droppé reste
// indéterminable, on ne reconstruit aucun joueur de ce match (résidu) : insérer
// un roster partiel pourrait laisser le moteur LUSR sur un déséquilibre, ce
// qu'on cherche justement à éliminer.
func inferDroppedRoster(knowns []knownParticipant, dropped []droppedPlayer, teamA, teamB int, edges []killEdge) (reconstructible []droppedPlayer, residualReason string) {
	teamByXUID := make(map[string]int, len(knowns))
	outcomeByTeam := map[int]int{}
	sizeByTeam := map[int]int{teamA: 0, teamB: 0}
	for _, k := range knowns {
		teamByXUID[k.XUID] = k.TeamID
		outcomeByTeam[k.TeamID] = k.Outcome
		sizeByTeam[k.TeamID]++
	}

	out := make([]droppedPlayer, 0, len(dropped))
	for i := range dropped {
		d := dropped[i]
		team, ok := inferDroppedTeam(d.XUID, teamByXUID, teamA, teamB, edges)
		if !ok {
			return nil, "equipe indeterminable -> re-fetch (droppe sans antagoniste a equipe connue ou vote a egalite)"
		}
		d.InferredTeam = team
		// L'outcome de D = celui de son équipe (depuis un connu de cette équipe).
		oc, hasOC := outcomeByTeam[team]
		if !hasOC {
			// Camp sans aucun connu (théoriquement impossible vu teamA/teamB issus
			// des connus) → outcome inconnu → résidu plutôt qu'un outcome erroné.
			return nil, "outcome introuvable pour l'equipe inferee"
		}
		d.InferredOutcome = oc
		sizeByTeam[team]++
		out = append(out, d)
	}

	// Vérif d'équilibre + cap roster (anti BUG 2). resA/resB = effectif final
	// reconstruit par camp (connus + droppés inférés). On n'accepte QUE les
	// rosters Arena propres : total <= 8 ET |A-B| <= 1. Tout ce qui dépasse
	// (BTB 8v8, matchs à substitutions → kill-feed distinct gonfle l'effectif)
	// part en résidu pour un re-fetch carnage précis.
	resA := sizeByTeam[teamA]
	resB := sizeByTeam[teamB]
	diff := resA - resB
	if diff < 0 {
		diff = -diff
	}
	if resA+resB > 8 || diff > 1 {
		return nil, "roster reconstruit > 8 ou desequilibre -> re-fetch"
	}
	return out, ""
}
