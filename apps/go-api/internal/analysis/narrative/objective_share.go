package narrative

// objective_share.go — LA PART DU JOUEUR DANS L'OBJECTIF DE SON ÉQUIPE, pour UN match.
//
// Entrée de la note de performance des matchs à objectif : le poids de la métrique
// d'objectif croît avec ce que le joueur a fait de l'objectif de son camp, et sa valeur
// est l'indice par famille de mode du match (r_f, objective_participation.go).
//
// Tout vient des tables de ce package : familles, poids d'action
// (ObjectiveFamilyActionWeights), colonnes de durée (ObjectiveFamilyHoldColumns) et
// repères P80. Aucune liste de colonnes parallèle : une ligne porte les valeurs brutes de
// `match_objective_stats_latest` par nom de colonne, et c'est la famille du MATCH qui
// décide lesquelles comptent.
//
// AUCUN I/O. Un titre sans `match_objective_stats` (Halo 5) ne fournit aucune ligne : les
// fonctions rendent ok == false, jamais une erreur ni une valeur inventée.

// ObjectiveColTimesSelectedAsVIP est la colonne « sélections comme VIP » : dénominateur de
// kills_as_vip et de time_as_vip_seconds (famille VIP). Elle ne figure dans aucune table
// de poids (colonne écartée des actions), mais une ligne doit la porter.
const ObjectiveColTimesSelectedAsVIP = "times_selected_as_vip"

// ObjectiveTeamRow est la ligne d'un joueur dans un match à objectif.
type ObjectiveTeamRow struct {
	XUID string
	// TeamID : camp du joueur dans le match ; NoObjectiveTeam si inconnu.
	TeamID int
	// Family : famille de la ligne (discriminant de ligne, KOTH/Strongholds compris) ;
	// "" = ligne sans aucun bloc d'objectif.
	Family ObjectiveFamily
	// Values : valeurs brutes de la ligne par nom de colonne de match_objective_stats
	// (colonnes d'action, de durée et ObjectiveColTimesSelectedAsVIP ; une colonne NULL
	// ou absente vaut 0).
	Values map[string]float64
	// TimePlayedSeconds : durée jouée par le joueur dans le match (terme de tenue).
	TimePlayedSeconds float64
}

// NoObjectiveTeam marque un camp inconnu (un camp réel est >= 0).
const NoObjectiveTeam = -1

// ObjectiveTeamShare est la part d'un joueur dans l'objectif de son camp pour un match.
type ObjectiveTeamShare struct {
	// Family : famille du MATCH (celle dont les poids ont servi).
	Family ObjectiveFamily
	// Points : actions pondérées du joueur (ObjectiveActionPoints).
	Points float64
	// TeamPoints : actions pondérées de tout son camp, lui compris.
	TeamPoints float64
	// Share = Points / TeamPoints dans [0, 1] ; 0 quand le camp n'a aucune action.
	Share float64
	// Index : indice r_f du joueur pour ce match, écrêté à [0, ObjectiveIndexClip] (la valeur
	// des radars et des profils).
	Index float64
	// RawIndex : le même indice SANS écrêtage (>= 0) : la valeur qui sert à classer les matchs
	// entre eux.
	RawIndex float64
	// SupportPoints / TeamSupportPoints / SupportShare : mêmes grandeurs sur les seules actions de
	// SOUTIEN (ObjectiveSupportPoints : frags à poids réduit). SupportShare vaut 0 quand le camp n'a
	// aucune action de soutien.
	SupportPoints, TeamSupportPoints, SupportShare float64
}

// MatchObjectiveFamily rend la famille d'un MATCH depuis les familles de ses lignes : la
// plus fréquente, KOTH l'emportant sur Strongholds dès qu'une ligne y compte des ticks (le
// discriminant de ligne classe en Strongholds un joueur de KOTH sans tick). "" si aucune
// ligne ne porte de famille. Égalité : l'ordre de AllObjectiveFamilies.
func MatchObjectiveFamily(rows []ObjectiveTeamRow) ObjectiveFamily {
	count := map[ObjectiveFamily]int{}
	for i := range rows {
		if rows[i].Family != "" {
			count[rows[i].Family]++
		}
	}
	if count[FamilyZonesKOTH] > 0 {
		return FamilyZonesKOTH
	}
	var best ObjectiveFamily
	for _, fam := range AllObjectiveFamilies() {
		if count[fam] > count[best] {
			best = fam
		}
	}
	return best
}

// objectiveStatsOfRow ramène une ligne au vocabulaire d'UNE famille : un agrégat d'un match
// que ObjectiveActionPoints et objectiveFamilySubScore savent lire.
func objectiveStatsOfRow(fam ObjectiveFamily, r *ObjectiveTeamRow) ObjectiveFamilyStats {
	weights := ObjectiveFamilyActionWeights[fam]
	st := ObjectiveFamilyStats{
		Matches:           1,
		TimePlayedSeconds: r.TimePlayedSeconds,
		ColumnSums:        make(map[string]float64, len(weights)),
	}
	for col := range weights {
		st.ColumnSums[col] = r.Values[col]
	}
	for _, col := range ObjectiveFamilyHoldColumns[fam] {
		st.HoldSeconds += r.Values[col]
	}
	if fam == FamilyVIP {
		st.TimesSelectedAsVIP = r.Values[ObjectiveColTimesSelectedAsVIP]
	}
	return st
}

// ComputeObjectiveTeamShare calcule la part de `xuid` dans l'objectif de son camp pour un
// match, depuis les lignes de TOUS les joueurs du match (les deux camps : seules celles du
// camp de `xuid` entrent dans TeamPoints).
//
// ok == false — la métrique est alors ABSENTE, jamais zéro — quand : aucune ligne, pas de
// ligne pour `xuid`, famille du match inconnue (titre sans match_objective_stats,
// match sans bloc d'objectif) ou camp de `xuid` inconnu. Un camp sans aucune action
// (TeamPoints == 0) n'est PAS une absence : la part vaut 0, et l'indice du joueur
// (tenue comprise) reste mesuré.
func ComputeObjectiveTeamShare(rows []ObjectiveTeamRow, xuid string) (ObjectiveTeamShare, bool) {
	fam := MatchObjectiveFamily(rows)
	if fam == "" {
		return ObjectiveTeamShare{}, false
	}
	var me *ObjectiveTeamRow
	for i := range rows {
		if rows[i].XUID == xuid {
			me = &rows[i]
			break
		}
	}
	if me == nil || me.TeamID == NoObjectiveTeam {
		return ObjectiveTeamShare{}, false
	}
	var teamPoints, teamSupport float64
	for i := range rows {
		r := &rows[i]
		if r.TeamID != me.TeamID {
			continue
		}
		st := objectiveStatsOfRow(fam, r)
		teamPoints += ObjectiveActionPoints(fam, st)
		teamSupport += ObjectiveSupportPoints(fam, st)
	}
	mine := objectiveStatsOfRow(fam, me)
	out := ObjectiveTeamShare{
		Family:            fam,
		Points:            ObjectiveActionPoints(fam, mine),
		TeamPoints:        teamPoints,
		Index:             min(objectiveFamilyRawScore(fam, mine), ObjectiveIndexClip),
		RawIndex:          objectiveFamilyRawScore(fam, mine),
		SupportPoints:     ObjectiveSupportPoints(fam, mine),
		TeamSupportPoints: teamSupport,
	}
	if teamPoints > 0 {
		out.Share = min(1, out.Points/teamPoints)
	}
	if teamSupport > 0 {
		out.SupportShare = min(1, out.SupportPoints/teamSupport)
	}
	return out, true
}
