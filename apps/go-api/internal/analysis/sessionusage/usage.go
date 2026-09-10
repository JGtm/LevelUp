// Package sessionusage — AGRÉGAT DE SESSION des usages d'équipement et de socles
// (chantier session-usage S2, .ai/HANDOFF_SESSION_USAGE_BDD_2026-09-04.md §5/S2).
//
// Fonctions PURES (zéro DB, zéro HTTP) : la couche repo (platform/duckdb,
// SessionUsageRepo) lit les vues `match_usage_players_latest` /
// `match_usage_films_latest` (ADR 0026 — JAMAIS les tables brutes) et
// `match_participants` ; le service assemble [Input] ; ce package calcule le bloc
// contractuel domain.SessionUsageBlock.
//
// LES QUATRE DÉCISIONS DE CALCUL, dans l'ordre où elles mordent :
//
//  1. Un match est MESURÉ s'il a une ligne film. Toutes les sommes, parts,
//     étendues et cadences ne portent QUE sur les matchs mesurés ; le couple
//     « matchs mesurés N / matchs M » dit le reste (couverture jamais totale).
//  2. Le camp du joueur vient de match_participants (S1 ne le stocke pas, à
//     dessein — §3 du handoff). Camp inconnu sur un match (FFA, participant
//     absent) : les grandeurs d'équipe de CE match sont hors calcul (part nil),
//     jamais un 0 inventé — et les grandeurs d'équipe de la SESSION se calculent
//     sur le sous-ensemble des matchs à camp connu, numérateurs ET dénominateurs
//     (règle de scope détaillée sur computeMetric ; sous-ensemble vide = nil).
//  3. La parité d'un match est 100/effectif DE CE MATCH ; les parités de session
//     sont 100/effectif MOYEN. Les deux coexistent : le compte « au-dessus de la
//     parité » se juge match par match, la ligne de parité affichée est la
//     moyenne.
//  4. Les cadences par 10 min ne portent que sur les matchs à durée CONNUE
//     (numérateur et dénominateur) : un match mesuré sans échelle de temps reste
//     dans les totaux et les parts, jamais dans une cadence.
package sessionusage

import (
	"sort"
	"strings"

	"levelup/go-api/internal/domain"
)

// Clés des grandeurs fixes du bloc (les déploiements portent la clé dynamique
// MetricDeployedPrefix + famille du manifeste, ex. "deployed_wall").
const (
	MetricGrapplePulls       = "grapple_pulls"
	MetricCamoEpisodes       = "camo_episodes"
	MetricOvershieldEpisodes = "overshield_episodes"
	MetricDroppedObjects     = "dropped_objects"
	MetricPadPickups         = "pad_pickups"
	MetricDeployedPrefix     = "deployed_"
	// MetricEquipmentPrefix — LE BILAN D'ÉQUIPEMENT par famille (étape E3) :
	// "equipment_wall", "equipment_powerup_camo"... Sa valeur est un compte
	// d'OBJETS (utilisé + gardé + lâché), pas un compte de gestes, et c'est la
	// seule famille de clés qui porte domain.SessionUsageOutcomes.
	MetricEquipmentPrefix = "equipment_"
)

// PlayerRow — la ligne (match, joueur) telle que servie par
// match_usage_players_latest. Les grenades n'y figurent PAS : produites en S1,
// exclues du contrat de session (décision utilisateur 2026-09-04).
type PlayerRow struct {
	MatchID            string
	XUID               string
	GrapplePulls       int
	CamoEpisodes       int
	OvershieldEpisodes int
	DroppedObjects     int
	PadPickups         int
	DeployedByFamily   map[string]int
	PadPickupsByFamily map[string]int
	// Les QUATRE ventilations d'issue (colonnes taken/spent/kept/dropped_json,
	// révision de projection `us4`). Toutes dans le vocabulaire des POSES : les
	// quatre se joignent sur UNE clé de famille, sans pont. Une ligne écrite par
	// une passe antérieure les porte vides — ce qui rend la grandeur absente, pas
	// nulle.
	TakenByFamily   map[string]int
	SpentByFamily   map[string]int
	KeptByFamily    map[string]int
	DroppedByFamily map[string]int
}

// FilmRow — la ligne de grain match de match_usage_films_latest (l'existence de
// cette ligne EST la définition de « match mesuré »). Seules les colonnes
// CONSOMMÉES par l'agrégat sont portées (0 code mort) : frame_interval_ms et
// pad_named vivent en table mais n'ont aucun consommateur ici.
type FilmRow struct {
	MatchID        string
	DurationMS     int64
	PadUnnamed     int
	PowerupPickups map[string]int
}

// ParticipantRow — un participant du match (match_participants) : l'appartenance
// de camp pour l'attribution, la présence à la fin pour les effectifs, le
// gamertag pour la résolution des coéquipiers suivis (contexte escouade).
type ParticipantRow struct {
	MatchID             string
	XUID                string
	Gamertag            string
	TeamID              *int
	PresentAtCompletion bool
}

// MatchInput — un match de la session, assemblé par le service.
type MatchInput struct {
	MatchID  string
	Measured bool
	// DurationSeconds : durée jouée du match MESURÉ (film ; repli durée du match
	// côté stats si le film n'a pas d'échelle de temps). 0 = inconnue.
	DurationSeconds float64
	// PlayerTeam : camp du joueur suivi (nil = inconnu — les parts d'équipe de ce
	// match sont hors calcul). TeamOf : xuid -> camp, pour classer chaque ligne.
	PlayerTeam *int
	TeamOf     map[string]int
	// TeamSize / LobbySize : joueurs présents à la fin (bots inclus). 0 = inconnu.
	TeamSize       int
	LobbySize      int
	Players        []PlayerRow
	PadUnnamed     int
	PowerupPickups map[string]int
}

// Input — la session entière (matchs mesurés ou non, dans l'ordre d'affichage).
type Input struct {
	PlayerXUID string
	// SquadXUIDs : coéquipiers suivis (contexte escouade, résolus par
	// ResolveTrackedSquad) — une ligne Squad par grandeur et par xuid, dans cet
	// ordre. Vide = contexte solo.
	SquadXUIDs []string
	Matches    []MatchInput
}

// ComputeUsage projette la session en bloc contractuel. Session sans aucun match
// mesuré : bloc Available avec MatchesMeasured=0 et aucun métrique — jamais nil
// (« matchs mesurés 0/N » doit s'afficher, §5/S2).
func ComputeUsage(in Input) domain.SessionUsageBlock {
	out := domain.SessionUsageBlock{Available: true, MatchesTotal: len(in.Matches)}
	measured := measuredMatches(in.Matches)
	out.MatchesMeasured = len(measured)
	if len(measured) == 0 {
		return out
	}
	// Dénominateurs des cadences : durée des seuls matchs à durée CONNUE (un
	// match mesuré sans échelle de temps est exclu des cadences, numérateur et
	// dénominateur — sinon il gonflerait le taux en silence) ; la cadence
	// d'équipe se restreint en plus aux matchs à camp connu (règle de scope).
	var durAll, durTeam float64
	for _, m := range measured {
		out.PadUnnamedTotal += m.PadUnnamed
		if m.DurationSeconds > 0 {
			durAll += m.DurationSeconds
			if m.PlayerTeam != nil {
				durTeam += m.DurationSeconds
			}
		}
	}
	out.MeasuredDurationSeconds = durAll
	out.TeamSizeAvg, out.TeamParityPct = averageAndParity(measured, func(m MatchInput) int { return m.TeamSize })
	out.LobbySizeAvg, out.LobbyParityPct = averageAndParity(measured, func(m MatchInput) int { return m.LobbySize })

	for _, key := range metricKeys(in.PlayerXUID, measured) {
		m := computeMetric(in.PlayerXUID, key, measured, durAll, durTeam)
		appendSquadLines(&m, measured, in.SquadXUIDs, durAll)
		attachOutcomes(&m, in.PlayerXUID, measured)
		out.Metrics = append(out.Metrics, m)
	}
	out.PadFamilies = computePadFamilies(in.PlayerXUID, measured)
	out.PowerupPickups = computePowerups(measured, durAll)
	return out
}

// BuildMatchInputs — un MatchInput par match du scope, DANS L'ORDRE DONNÉ, mesuré
// s'il porte une ligne film. SOURCE UNIQUE de l'assemblage : les deux blocs
// (session et période) partent de là, et leurs appelants n'ont plus à regrouper
// les lignes joueur ni à recopier le contexte de camp.
//
// Le résultat est SANS échelle de temps ni compteurs de grain match (durée, socles
// anonymes, occupations de bonus) : ils ne servent qu'aux cadences et à la note de
// pied de la page Sessions, qui les pose elle-même par-dessus — le bloc de période
// n'a ni cadence ni note de pied.
func BuildMatchInputs(
	matchIDs []string, films map[string]FilmRow, players []PlayerRow, tc TeamContext,
) []MatchInput {
	playersByMatch := make(map[string][]PlayerRow, len(films))
	for _, p := range players {
		playersByMatch[p.MatchID] = append(playersByMatch[p.MatchID], p)
	}
	out := make([]MatchInput, 0, len(matchIDs))
	for _, id := range matchIDs {
		_, measured := films[id]
		m := MatchInput{
			MatchID:   id,
			Measured:  measured,
			TeamOf:    tc.TeamOf[id],
			TeamSize:  tc.TeamSize[id],
			LobbySize: tc.LobbySize[id],
			Players:   playersByMatch[id],
		}
		if team, ok := tc.PlayerTeam[id]; ok {
			t := team
			m.PlayerTeam = &t
		}
		out = append(out, m)
	}
	return out
}

// measuredMatches — les seuls matchs qui portent une ligne film. TOUT le calcul
// des deux blocs (session et période) part de là : un match sans film n'est pas
// mesuré, et le couple « matchs mesurés N / matchs M » dit le reste.
func measuredMatches(matches []MatchInput) []MatchInput {
	out := make([]MatchInput, 0, len(matches))
	for _, m := range matches {
		if m.Measured {
			out = append(out, m)
		}
	}
	return out
}

// averageAndParity : effectif moyen sur les matchs où il est connu (>0), et la
// parité 100/moyenne. (0, nil) quand aucun match ne le porte.
func averageAndParity(measured []MatchInput, size func(MatchInput) int) (float64, *float64) {
	sum, n := 0, 0
	for _, m := range measured {
		if s := size(m); s > 0 {
			sum += s
			n++
		}
	}
	if n == 0 {
		return 0, nil
	}
	avg := float64(sum) / float64(n)
	p := 100 / avg
	return avg, &p
}

// metricKeys — les cinq grandeurs fixes, puis les familles déployées observées,
// puis les familles du BILAN D'ÉQUIPEMENT observées (triées dans chaque groupe :
// l'ordre de sortie est un contrat de stabilité, pas une itération de map).
//
// `deployed_<famille>` reste au critère du LOBBY (n'importe quel joueur du match) :
// c'est une grandeur comparative — « combien de murs déployés dans ce lobby » — qui
// a un sens même à zéro pour le joueur suivi.
//
// `equipment_<famille>` (le BILAN) entre au critère du SUJET SEUL, depuis le lot
// 6.4 point 4 : [subjectBilanFamilies], LA MÊME fonction que lit [overviewFamilies]
// (usage_overview.go) pour les pages Synthèse et Escouade. AVANT ce correctif, ce
// critère était le LOBBY ici et le SUJET là-bas — deux pages qui affichent la MÊME
// barre subjet-only (`attachOutcomes` / `computeOutcomes`, toutes deux réduites au
// joueur suivi) pouvaient ouvrir une ligne différente pour le même scope : la page
// Sessions montrait une barre entièrement VIDE (0 utilisé / 0 gardé / 0 lâché) dès
// qu'un COÉQUIPIER avait touché la famille, exactement le « reproche sans objet »
// que le commentaire d'[overviewFamilies] refuse déjà côté Synthèse/Escouade. Une
// famille du bilan entre dès qu'une de ses TROIS issues (utilisé/gardé/lâché) est
// non nulle POUR LE SUJET, et pas seulement sur ses poses : le capteur du parc
// compte 4 objets utilisés pour 36 pris (mesure E0.4) — le lire sur ses seules
// poses effacerait précisément l'histoire que ce bloc raconte.
func metricKeys(playerXUID string, measured []MatchInput) []string {
	deployed := map[string]bool{}
	for _, m := range measured {
		for _, p := range m.Players {
			for fam := range p.DeployedByFamily {
				deployed[fam] = true
			}
		}
	}
	bilan := subjectBilanFamilies(playerXUID, measured)
	keys := make([]string, 0, 5+len(deployed)+len(bilan))
	keys = append(keys,
		MetricPadPickups, MetricCamoEpisodes, MetricOvershieldEpisodes,
		MetricGrapplePulls, MetricDroppedObjects,
	)
	keys = append(keys, prefixedSorted(MetricDeployedPrefix, deployed)...)
	keys = append(keys, prefixedSorted(MetricEquipmentPrefix, bilan)...)
	return keys
}

// prefixedSorted — les clés d'un ensemble, triées et préfixées.
func prefixedSorted(prefix string, set map[string]bool) []string {
	fams := make([]string, 0, len(set))
	for fam := range set {
		fams = append(fams, fam)
	}
	sort.Strings(fams)
	for i, fam := range fams {
		fams[i] = prefix + fam
	}
	return fams
}

// metricValue — la valeur d'une grandeur pour une ligne joueur.
func metricValue(key string, p *PlayerRow) int {
	switch key {
	case MetricGrapplePulls:
		return p.GrapplePulls
	case MetricCamoEpisodes:
		return p.CamoEpisodes
	case MetricOvershieldEpisodes:
		return p.OvershieldEpisodes
	case MetricDroppedObjects:
		return p.DroppedObjects
	case MetricPadPickups:
		return p.PadPickups
	}
	if fam, ok := strings.CutPrefix(key, MetricEquipmentPrefix); ok {
		// LA VALEUR EST LA SOMME DES TROIS ISSUES, jamais les prises : c'est ce qui
		// garantit qu'une barre empilée sur `outcomes` remplit EXACTEMENT sa
		// longueur. Même règle que la colonne fusionnée de la vue match
		// (`equipmentGroup`, étape E2) — les prises restent servies à côté, comme
		// dénominateur d'honnêteté.
		return equipmentOutcomeOf(p, fam).total()
	}
	if fam, ok := strings.CutPrefix(key, MetricDeployedPrefix); ok {
		return p.DeployedByFamily[fam]
	}
	return 0
}

// computeMetric agrège UNE grandeur sur les matchs mesurés : triplet de sommes,
// parts croisées, cadences, étendue et comptes au-dessus de la parité du match.
//
// RÈGLE DE SCOPE (ronde de correction S2, appliquée à l'identique dans
// computePadFamilies et objectiveRoleMetrics) : les grandeurs de session
// RELATIVES À L'ÉQUIPE (team_total, player_share_of_team, team_share_of_lobby,
// team_per_10min, matches_above_team_parity) se calculent sur le SOUS-ENSEMBLE
// des matchs mesurés à camp CONNU — numérateurs ET dénominateurs. Croiser les
// scopes (joueur/lobby sommés sur tous les matchs contre une équipe sommée sur
// les seuls matchs à camp connu) ferait dépasser 100 % à player_share_of_team
// dès qu'une session mêle équipe et FFA, et diluerait team_share_of_lobby.
// Sous-ensemble vide : toutes ces valeurs restent nil — jamais un 0 inventé
// pour dire « inconnu ». Les grandeurs joueur/lobby, elles, portent sur TOUS
// les matchs mesurés. Cadences : matchs à durée connue seulement (durAll /
// durTeam, calculés par l'appelant), numérateur et dénominateur.
func computeMetric(playerXUID, key string, measured []MatchInput, durAll, durTeam float64) domain.SessionUsageMetric {
	out := domain.SessionUsageMetric{Key: key}
	var teamSum, playerTeamScope, lobbyTeamScope float64 // scope camp connu
	var playerDur, teamDur, lobbyDur float64             // scope durée connue
	teamKnown, aboveTeamParity := false, 0
	for i := range measured {
		m := &measured[i]
		p, t, l := matchSums(playerXUID, key, m)
		out.PlayerTotal += float64(p)
		out.LobbyTotal += float64(l)
		if m.DurationSeconds > 0 {
			playerDur += float64(p)
			lobbyDur += float64(l)
		}
		point := domain.SessionUsageMatchPoint{MatchID: m.MatchID}
		point.PlayerShareOfLobbyPct = sharePct(float64(p), float64(l))
		if m.PlayerTeam != nil {
			teamKnown = true
			teamSum += float64(t)
			playerTeamScope += float64(p)
			lobbyTeamScope += float64(l)
			if m.DurationSeconds > 0 {
				teamDur += float64(t)
			}
			point.PlayerShareOfTeamPct = sharePct(float64(p), float64(t))
			point.TeamShareOfLobbyPct = sharePct(float64(t), float64(l))
		}
		if s := point.PlayerShareOfTeamPct; s != nil {
			if m.TeamSize > 0 && *s > 100/float64(m.TeamSize) {
				aboveTeamParity++
			}
		}
		if s := point.PlayerShareOfLobbyPct; s != nil && m.LobbySize > 0 && *s > 100/float64(m.LobbySize) {
			out.MatchesAboveLobbyParity++
		}
		out.PerMatch = append(out.PerMatch, point)
	}
	// Grandeurs d'équipe SEULEMENT si le sous-ensemble à camp connu est non
	// vide : sur une session entièrement FFA, un total, une part ou une cadence
	// « d'équipe » à 0 serait une mesure inventée.
	if teamKnown {
		out.TeamTotal = &teamSum
		out.TeamShareOfLobbyPct = sharePct(teamSum, lobbyTeamScope)
		out.PlayerShareOfTeamPct = sharePct(playerTeamScope, teamSum)
		out.TeamPer10Min = per10Min(teamDur, durTeam)
		out.MatchesAboveTeamParity = &aboveTeamParity
	}
	out.PlayerShareOfLobbyPct = sharePct(out.PlayerTotal, out.LobbyTotal)
	out.PlayerPer10Min = per10Min(playerDur, durAll)
	out.LobbyPer10Min = per10Min(lobbyDur, durAll)
	return out
}

// matchSums — (joueur, camp, lobby) d'une grandeur sur UN match. Le camp
// n'additionne que si le camp du joueur est connu (le lobby, toujours).
func matchSums(playerXUID, key string, m *MatchInput) (player, team, lobby int) {
	for i := range m.Players {
		p := &m.Players[i]
		v := metricValue(key, p)
		lobby += v
		if p.XUID == playerXUID {
			player += v
		}
		if m.PlayerTeam != nil {
			if teamID, ok := m.TeamOf[p.XUID]; ok && teamID == *m.PlayerTeam {
				team += v
			}
		}
	}
	return player, team, lobby
}

// sharePct — 100*num/den, nil si le dénominateur est nul (0/0 n'est pas 0 %).
func sharePct(num, den float64) *float64 {
	if den <= 0 {
		return nil
	}
	v := 100 * num / den
	return &v
}

// per10Min — cadence par dix minutes de jeu mesuré ; nil sans durée.
func per10Min(total, durationSeconds float64) *float64 {
	if durationSeconds <= 0 {
		return nil
	}
	v := total * 600 / durationSeconds
	return &v
}
