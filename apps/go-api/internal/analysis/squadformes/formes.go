// Package squadformes — L'ASSEMBLAGE DU BLOC « FORMES RETENUES », réduit à ce que lisent les
// cartes d'objectif (Escouade › Contributions, Séries temporelles › Usages ; plan
// PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05, décision D7).
//
// # CE QUE CE PAQUET FAIT, ET CE QU'IL NE FAIT PAS
//
// Il ASSEMBLE, il ne résume pas : une feuille d'objectif par match, les deux camps, telle que
// les vues `_latest` la servent. Les parts se calculent dans les modèles purs du web, à l'endroit
// exact où elles s'affichent.
//
// Ce qu'il tranche, en revanche, et qui n'appartient qu'au serveur : le PÉRIMÈTRE des colonnes
// d'objectif d'une famille de mode — dérivé de narrative (source unique de la classification par
// rôle), puis restreint aux colonnes que le scope mesure vraiment (« les colonnes sont pilotées
// par la donnée ») —, et le CAMP de chaque ligne, recollé depuis les participants.
//
// Pur : aucune ouverture de base, aucune horloge, aucune chaîne de langue.
package squadformes

import (
	"sort"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
)

// MatchMeta — l'identité d'affichage d'un match du scope, résolue par le service
// appelant (libellés déjà passés par les adapters du titre).
type MatchMeta struct {
	MatchID   string
	StartTime string
	ModeLabel string
	MapLabel  string
	// ObjectiveExcluded : mode que le titre écarte des parts de rôle de l'escouade (drapeau
	// neutre, D6 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26). Décidé par l'appelant :
	// ce paquet n'importe aucun paquet de titre.
	ObjectiveExcluded bool
}

// ObjectiveColumnRow — une ligne (match, joueur) de `match_objective_stats_latest`
// projetée sur les colonnes de sa famille. Les deux camps y sont.
type ObjectiveColumnRow struct {
	MatchID string
	XUID    string
	Family  narrative.ObjectiveFamily
	Values  map[string]float64
	// FlagJuggleWindowSeconds : la fenêtre sous laquelle les prises nettes de
	// cette ligne ont été calculées. Zéro = la ligne n'en porte pas.
	FlagJuggleWindowSeconds float64
}

// WeaponInfo — ce que le catalogue du titre sait d'une famille d'arme de socle, lu par l'Emprise
// (`squademprise.Input.Weapons`). Résolu par le service (catalogue du titre + registre canonique) :
// ce paquet ne lit aucun fichier.
type WeaponInfo struct {
	// Label : nom d'affichage dans la langue de la requête. Vide = famille hors
	// catalogue (le web affiche sa réserve, jamais un nom approchant).
	Label string
	// WeaponKey : clé canonique du registre. Vide = famille hors registre.
	WeaponKey string
}

// Input — tout ce que l'assemblage demande.
type Input struct {
	PlayerXUID string
	// SquadPlayers : le joueur de la page EN TÊTE, puis les coéquipiers
	// sélectionnés, dans l'ordre d'affichage.
	SquadPlayers []domain.SessionUsageSquadPlayer
	// Metas : les matchs du scope, dans l'ordre d'affichage. C'est cette liste
	// qui fait le scope.
	Metas []MatchMeta
	// Matches : le même scope, assemblé par sessionusage.BuildMatchInputs (film décodé, camp du
	// joueur, camp de chaque participant).
	Matches []sessionusage.MatchInput
	// Objectives : les lignes d'objectif du scope, les deux camps.
	Objectives []ObjectiveColumnRow
}

// Build assemble le bloc. Scope vide ⇒ bloc Available avec zéro match : « 0 sur
// 0 » doit pouvoir s'afficher, le vide n'est pas une indisponibilité.
//
// SEULS LES MATCHS À OBJECTIF SONT PUBLIÉS : un match sans feuille d'objectif n'alimente aucune
// carte. CE QUI EST COMPTÉ NE CHANGE PAS : [domain.SquadFormesBlock.MatchesTotal] reste la portée
// ENTIÈRE et [domain.SquadFormesBlock.MatchesMeasured] le nombre de matchs à film — les écrans
// dérivent leurs pieds des deux compteurs, jamais de la longueur de la liste.
func Build(in Input) domain.SquadFormesBlock {
	out := domain.SquadFormesBlock{
		Available:    true,
		MatchesTotal: len(in.Metas),
		MainXUID:     in.PlayerXUID,
		Squad:        in.SquadPlayers,
	}
	byID := make(map[string]*sessionusage.MatchInput, len(in.Matches))
	for i := range in.Matches {
		byID[in.Matches[i].MatchID] = &in.Matches[i]
	}
	objByMatch, columnsByFamily := projectObjectives(in.Objectives)

	for _, meta := range in.Metas {
		mi := byID[meta.MatchID]
		if mi != nil && mi.Measured {
			out.MatchesMeasured++
		}
		// LE CAMP DE CHAQUE LIGNE D'OBJECTIF vient des PARTICIPANTS du match : la feuille
		// d'objectif ne porte pas l'appartenance. Sans ce recollement, aucune ligne n'a de camp et
		// TOUT l'objectif se lit comme adverse (« 0,0 % » sur toutes les familles de mode).
		var teamOf map[string]int
		if mi != nil {
			teamOf = mi.TeamOf
		}
		objective := buildObjective(objByMatch[meta.MatchID], columnsByFamily, teamOf)
		if objective == nil {
			continue
		}
		objective.ExcludedFromBalance = meta.ObjectiveExcluded
		m := domain.SquadFormesMatch{
			MatchID:   meta.MatchID,
			StartTime: meta.StartTime,
			ModeLabel: meta.ModeLabel,
			MapLabel:  meta.MapLabel,
			Objective: objective,
		}
		if mi != nil && mi.PlayerTeam != nil {
			t := *mi.PlayerTeam
			m.PlayerTeam = &t
		}
		out.Matches = append(out.Matches, m)
	}
	return out
}

// buildObjective rend le bloc objectif d'un match : sa famille, les colonnes de
// cette famille (les mêmes pour tous ses matchs, sinon deux grilles du même mode
// n'auraient pas les mêmes colonnes), les valeurs des deux camps ET LE CAMP DE
// CHACUN — recollé depuis les participants, la feuille d'objectif ne le portant
// pas. Une ligne sans camp connu reste publiée SANS camp : elle comptera dans le
// lobby et dans aucun des deux côtés, ce qui est la vérité.
func buildObjective(
	rows []ObjectiveColumnRow, columnsByFamily map[narrative.ObjectiveFamily][]domain.SquadFormesObjectiveColumn,
	teamOf map[string]int,
) *domain.SquadFormesObjective {
	if len(rows) == 0 {
		return nil
	}
	fam := rows[0].Family
	cols := columnsByFamily[fam]
	if len(cols) == 0 {
		return nil
	}
	out := &domain.SquadFormesObjective{Family: string(fam), Columns: cols}
	for _, r := range rows {
		// La fenêtre est la MÊME pour toutes les lignes d'un match (une passe,
		// une règle) : la première non nulle la donne.
		if out.FlagJuggleWindowSeconds == 0 {
			out.FlagJuggleWindowSeconds = r.FlagJuggleWindowSeconds
		}
		p := domain.SquadFormesObjectivePlayer{XUID: r.XUID, Values: map[string]float64{}}
		if team, ok := teamOf[r.XUID]; ok {
			t := team
			p.TeamID = &t
		}
		for _, c := range cols {
			v, mesure := r.Values[c.Key]
			// UNE GRANDEUR OPTIONNELLE ABSENTE N'EST PAS UN ZÉRO : sa clé reste
			// hors de `values`, et le web rend « non mesuré ». Les colonnes de
			// `match_objective_stats`, elles, gardent leur 0 — il y est une
			// mesure (cf. SquadFormesObjectiveColumn.Optional).
			if c.Optional && !mesure {
				continue
			}
			p.Values[c.Key] = v
		}
		out.Players = append(out.Players, p)
	}
	sort.Slice(out.Players, func(a, b int) bool { return out.Players[a].XUID < out.Players[b].XUID })
	return out
}

// projectObjectives regroupe les lignes par match et décide, PAR FAMILLE, les
// colonnes publiées.
//
// LES COLONNES SONT PILOTÉES PAR LA DONNÉE (règle de l'artefact) : une colonne
// que personne n'a alimentée sur le scope n'est pas une colonne à zéro, c'est
// une colonne qui n'existe pas — la publier ferait cinq grandeurs vides autour
// de celle qui compte. L'ordre est celui des rôles (prendre, défendre, tenir),
// puis celui de narrative dans le rôle : un contrat stable, jamais l'ordre
// d'arrivée d'une map.
func projectObjectives(rows []ObjectiveColumnRow) (
	map[string][]ObjectiveColumnRow, map[narrative.ObjectiveFamily][]domain.SquadFormesObjectiveColumn,
) {
	byMatch := map[string][]ObjectiveColumnRow{}
	nonZero := map[narrative.ObjectiveFamily]map[string]bool{}
	for _, r := range rows {
		byMatch[r.MatchID] = append(byMatch[r.MatchID], r)
		seen := nonZero[r.Family]
		if seen == nil {
			seen = map[string]bool{}
			nonZero[r.Family] = seen
		}
		for col, v := range r.Values {
			if v != 0 {
				seen[col] = true
			}
		}
	}
	cols := map[narrative.ObjectiveFamily][]domain.SquadFormesObjectiveColumn{}
	for fam, seen := range nonZero {
		var list []domain.SquadFormesObjectiveColumn
		for _, role := range narrative.AllObjectiveRoles() {
			for _, col := range familyColumnsOfRole(fam, role) {
				if !seen[col] {
					continue
				}
				_, optional := narrative.ObjectiveExtraGrandeurFamily(col)
				list = append(list, domain.SquadFormesObjectiveColumn{
					Key: col, Role: string(role),
					Duration: role == narrative.ObjectiveRoleHold,
					Optional: optional,
				})
			}
		}
		cols[fam] = list
	}
	return byMatch, cols
}

// familyColumnsOfRole — les grandeurs d'un rôle QUE CETTE FAMILLE possède :
// l'intersection de la classification par rôle et du vocabulaire de la famille,
// les deux tables uniques de narrative. Aucune liste locale, aucune curation.
//
// LES GRANDEURS HORS COLONNE (prises nettes de drapeau) s'y ajoutent par LEUR
// PROPRE famille déclarée : elles ne sont dans aucune table de poids — c'est
// exactement ce qui les empêche d'entrer dans un SUM sur
// `match_objective_stats` — donc le vocabulaire ci-dessus ne les contient pas.
func familyColumnsOfRole(fam narrative.ObjectiveFamily, role narrative.ObjectiveRole) []string {
	vocab := map[string]bool{}
	for col := range narrative.ObjectiveFamilyActionWeights[fam] {
		vocab[col] = true
	}
	for _, col := range narrative.ObjectiveFamilyHoldColumns[fam] {
		vocab[col] = true
	}
	var out []string
	for _, col := range narrative.ObjectiveRoleColumns(role) {
		if vocab[col] {
			out = append(out, col)
		}
	}
	for _, g := range narrative.ObjectiveRoleExtraGrandeurs(role) {
		if f, ok := narrative.ObjectiveExtraGrandeurFamily(g); ok && f == fam {
			out = append(out, g)
		}
	}
	return out
}
