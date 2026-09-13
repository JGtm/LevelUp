// Package squadformes — L'ASSEMBLAGE DU BLOC « FORMES RETENUES » de la page
// Escouade (artefact 2ec1b8eb, lot D2 du 2026-09-13).
//
// # CE QUE CE PAQUET FAIT, ET CE QU'IL NE FAIT PAS
//
// Il ASSEMBLE, il ne résume pas : une ligne par joueur et par match, les deux
// camps, telle que les vues `_latest` la servent. Les dix-neuf cartes de
// l'artefact emploient quatre dénominateurs différents (mon équipe, le lobby,
// l'escouade seule, les occupations de socle) et six formes ; figer ici un
// agrégat par carte aurait rendu impossible d'en changer un seul sans recuire
// le contrat. Les parts se calculent donc dans les modèles purs du web, à
// l'endroit exact où elles s'affichent.
//
// Ce qu'il tranche, en revanche, et qui n'appartient qu'au serveur :
//
//   - le PÉRIMÈTRE des colonnes d'objectif d'une famille de mode — dérivé de
//     narrative (source unique de la classification par rôle), puis restreint
//     aux colonnes que le scope mesure vraiment (« les colonnes sont pilotées
//     par la donnée ») ;
//   - la FAMILLE d'une arme de socle (lourde / précision / autre), dérivée du
//     registre canonique d'armes, jamais d'une table écrite pour l'occasion ;
//   - l'exclusion des grenades (ce ne sont pas des équipements) et du
//     répulseur (aucun canal ne mesure son usage — il n'a pas de geste ici).
//
// # PAS DE NORMALISATION PAR LA DURÉE
//
// Décision utilisateur du 2026-09-13 : « cadence c'est par match, pas par
// minutes ». Aucune grandeur n'est divisée par un temps de jeu ; la durée du
// match reste publiée parce qu'elle DIT le match, pas parce qu'elle normalise
// quoi que ce soit.
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
}

// FilmPads — le grain match des socles, lu sur `match_usage_films_latest`
// (colonnes que l'agrégat de session ne consommait pas jusqu'ici).
type FilmPads struct {
	MatchID    string
	PadNamed   int
	PadUnnamed int
	// WeaponPads : un élément par SOCLE du match, dans l'ordre du document.
	WeaponPads []WeaponPad
}

// WeaponPad — un socle d'arme et ses occupations.
type WeaponPad struct {
	Weapon      string
	Occupations int
	Named       int
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

// WeaponInfo — ce que le catalogue du titre sait d'une famille d'arme de socle.
// Résolu par le service (catalogue du titre + registre canonique) : ce paquet ne
// lit aucun fichier.
type WeaponInfo struct {
	// Label : nom d'affichage dans la langue de la requête. Vide = famille hors
	// catalogue (le web affiche sa réserve, jamais un nom approchant).
	Label string
	// WeaponKey : clé canonique du registre. Vide = famille hors registre.
	WeaponKey string
	// Class / Role : les deux dimensions du registre canonique, telles quelles.
	Class string
	Role  string
}

// Input — tout ce que l'assemblage demande.
type Input struct {
	PlayerXUID string
	// SquadPlayers : le joueur de la page EN TÊTE, puis les coéquipiers
	// sélectionnés, dans l'ordre d'affichage.
	SquadPlayers []domain.SessionUsageSquadPlayer
	// Metas : les matchs du scope, dans l'ordre d'affichage. C'est cette liste
	// qui fait le scope — un match absent des autres entrées est « non mesuré ».
	Metas []MatchMeta
	// Matches : le même scope, assemblé par sessionusage.BuildMatchInputs (camp,
	// effectifs, lignes joueur du lobby entier).
	Matches []sessionusage.MatchInput
	// Films : la durée mesurée par match (map par match_id).
	Films map[string]sessionusage.FilmRow
	// Pads : le grain match des socles (map par match_id).
	Pads map[string]FilmPads
	// Gamertags : xuid -> gamertag (participants), pour nommer le lobby.
	Gamertags map[string]string
	// Objectives : les lignes d'objectif du scope, les deux camps.
	Objectives []ObjectiveColumnRow
	// Weapons : clé de famille d'arme -> ce que le titre en sait.
	Weapons map[string]WeaponInfo
	// WallFamilyKey : la clé sous laquelle le TITRE nomme le mur de protection
	// dans sa ventilation de poses (`deployed_json`). Passée en paramètre, jamais
	// importée : `internal/analysis` reste title-agnostic (ADR 0012 / ADR 0025,
	// garde-rail archlint). Vide ⇒ aucune pose de mur n'est attribuée — c'est le
	// montage qui est incomplet, pas la soirée qui est sans mur.
	WallFamilyKey string
}

// Build assemble le bloc. Scope vide ⇒ bloc Available avec zéro match : « 0 sur
// 0 » doit pouvoir s'afficher, le vide n'est pas une indisponibilité.
//
// # SEULS LES MATCHS QUI ONT QUELQUE CHOSE À DIRE SONT PUBLIÉS (2026-09-13)
//
// Un match du scope qui ne porte NI film décodé NI feuille d'objectif ne peut
// alimenter aucune des dix-neuf cartes : les parts, les cadences et les socles
// se lisent dans le film, les grandeurs d'objectif dans la feuille de match.
// Publié quand même, il ne servait qu'à être compté — et sur une portée réelle
// de 1 147 matchs, ces lignes vides pesaient les deux tiers du bloc.
//
// CE QUI EST COMPTÉ NE CHANGE PAS, ET C'EST TOUT L'ENJEU : [domain.SquadFormesBlock.MatchesTotal]
// reste la portée ENTIÈRE et [domain.SquadFormesBlock.MatchesMeasured] le nombre
// de matchs à film. Les écrans qui disent « N matchs sans film décodé sont hors
// de cette forme » dérivent ce nombre des deux compteurs, jamais de la longueur
// de la liste — sans quoi l'allègement se lirait comme une perte de portée.
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
	usedWeapons := map[string]bool{}

	for _, meta := range in.Metas {
		m := domain.SquadFormesMatch{
			MatchID:   meta.MatchID,
			StartTime: meta.StartTime,
			ModeLabel: meta.ModeLabel,
			MapLabel:  meta.MapLabel,
		}
		if mi := byID[meta.MatchID]; mi != nil {
			fillFromMatchInput(&m, mi, &in, usedWeapons)
		}
		if f, ok := in.Films[meta.MatchID]; ok && f.DurationMS > 0 {
			m.DurationSeconds = float64(f.DurationMS) / 1000
		}
		if p, ok := in.Pads[meta.MatchID]; ok {
			m.PadNamed, m.PadUnnamed = p.PadNamed, p.PadUnnamed
			for _, wp := range p.WeaponPads {
				usedWeapons[wp.Weapon] = true
				m.WeaponPads = append(m.WeaponPads, domain.SquadFormesWeaponPad{
					Weapon: wp.Weapon, Occupations: wp.Occupations, Named: wp.Named,
				})
			}
		}
		// LE CAMP DE CHAQUE LIGNE D'OBJECTIF vient des PARTICIPANTS du match, la
		// même source que le lobby : la feuille d'objectif, elle, ne porte pas
		// l'appartenance. Sans ce recollement, aucune ligne n'a de camp et TOUT
		// l'objectif se lit comme adverse (défaut mesuré sur données réelles le
		// 2026-09-13 : « 0,0 % · −50,0 pts » sur toutes les familles de mode).
		var teamOf map[string]int
		if mi := byID[meta.MatchID]; mi != nil {
			teamOf = mi.TeamOf
		}
		m.Objective = buildObjective(objByMatch[meta.MatchID], columnsByFamily, teamOf)
		if m.Measured {
			out.MatchesMeasured++
		}
		// UN MATCH QUI NE PORTE NI FILM NI OBJECTIF N'A RIEN À PUBLIER — voir
		// l'en-tête de fonction. Il reste compté (MatchesTotal, et donc le nombre
		// de matchs sans film que les formes affichent en pied), il n'occupe
		// simplement plus une ligne de contrat vide.
		if !m.Measured && m.Objective == nil {
			continue
		}
		out.Matches = append(out.Matches, m)
	}
	out.Weapons = buildWeapons(usedWeapons, in.Weapons)
	return out
}

// fillFromMatchInput pose le camp, les effectifs et le lobby d'un match mesuré.
//
// UN MATCH NON MESURÉ N'A PAS DE LOBBY, et c'est la règle : ses cases se rendent
// en hachure « pas de film décodé », jamais en zéros qui se liraient comme une
// soirée sans gestes.
func fillFromMatchInput(
	m *domain.SquadFormesMatch, mi *sessionusage.MatchInput, in *Input, usedWeapons map[string]bool,
) {
	m.Measured = mi.Measured
	m.TeamSize, m.LobbySize = mi.TeamSize, mi.LobbySize
	if mi.PlayerTeam != nil {
		t := *mi.PlayerTeam
		m.PlayerTeam = &t
	}
	if !mi.Measured {
		return
	}
	for i := range mi.Players {
		p := &mi.Players[i]
		row := domain.SquadFormesLobbyPlayer{
			XUID:       p.XUID,
			Gamertag:   in.Gamertags[p.XUID],
			Camo:       p.CamoEpisodes,
			Overshield: p.OvershieldEpisodes,
			Wall:       p.DeployedByFamily[in.WallFamilyKey],
			Grapple:    p.GrapplePulls,
			Dropped:    p.DroppedObjects,
			PadPickups: p.PadPickups,
		}
		if team, ok := mi.TeamOf[p.XUID]; ok {
			t := team
			row.TeamID = &t
		}
		if len(p.PadPickupsByFamily) > 0 {
			row.PadsByWeapon = make(map[string]int, len(p.PadPickupsByFamily))
			for fam, v := range p.PadPickupsByFamily {
				row.PadsByWeapon[fam] = v
				usedWeapons[fam] = true
			}
		}
		m.Lobby = append(m.Lobby, row)
	}
	sort.Slice(m.Lobby, func(a, b int) bool { return m.Lobby[a].XUID < m.Lobby[b].XUID })
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

// buildWeapons nomme et range les armes de socle rencontrées, triées par clé
// (contrat stable ; l'ordre d'affichage — par volume de prises — appartient au
// lecteur, qui seul connaît le dénominateur qu'il affiche).
func buildWeapons(used map[string]bool, catalog map[string]WeaponInfo) []domain.SquadFormesWeapon {
	keys := make([]string, 0, len(used))
	for k := range used {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]domain.SquadFormesWeapon, 0, len(keys))
	for _, k := range keys {
		info := catalog[k]
		out = append(out, domain.SquadFormesWeapon{
			Key: k, Label: info.Label, WeaponKey: info.WeaponKey, Class: WeaponClassOf(info),
		})
	}
	return out
}

// Les valeurs du registre canonique d'armes lues ici (internal/games/weapons) :
// `class` est l'axe de MANIPULATION, `role` la fonction de combat.
const (
	registryClassHeavy    = "heavy"
	registryRolePrecision = "precision"
	registryRoleSniper    = "sniper"
)

// WeaponClassOf range une arme de socle dans l'une des trois familles produit du
// bloc « contrôle des armes spéciales ».
//
// LA TABLE EST DÉRIVÉE, PAS DÉCLARÉE. L'artefact posait la réserve « la table
// famille d'arme → catégorie est un savoir du titre, à déclarer en donnée » :
// ce savoir EXISTE déjà, et en un seul endroit — le registre canonique d'armes,
// qui porte `class` (manipulation) et `role` (fonction) pour les deux titres. Le
// redéclarer dans un TOML aurait fait une SECONDE source d'identité d'arme,
// exactement ce que le registre a supprimé (V72-06). L'ordre des tests compte :
// le fusil de précision S7 est classé `heavy` par le registre, et c'est bien une
// arme lourde de socle.
func WeaponClassOf(info WeaponInfo) string {
	switch {
	case info.Class == registryClassHeavy:
		return domain.SquadFormesWeaponHeavy
	case info.Role == registryRolePrecision || info.Role == registryRoleSniper:
		return domain.SquadFormesWeaponPrecision
	default:
		return domain.SquadFormesWeaponOther
	}
}
