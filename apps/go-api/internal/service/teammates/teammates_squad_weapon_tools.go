// Package teammates — teammates_squad_weapon_tools.go : « Outils de destruction » de
// l'Escouade, chaque frag nommé (décision D8 du plan
// .ai/PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26.md).
//
// # Ce que ce builder est, et pourquoi il n'est pas fragdist
//
// `fragdist.Build` sert la « Répartition des frags » de six appelants : une ventilation PAR
// CLASSE, réconciliée aux compteurs de la feuille de match, où un type de grenade que le
// film et la feuille comptent différemment retombe en une classe « Grenade » sans détail.
// Les « Outils de destruction » posent une autre question — avec quoi chacun a frappé, outil
// par outil — et ne doivent RIEN changer aux six autres. D'où ce builder propre, à côté de
// buildSquadWeaponKills, qui lit les mêmes lignes :
//
//   - une ligne par clé d'arme (film pour un titre qui le décode, table native sinon),
//     grenades comprises, chacune par son type, sans repli « Grenade » ;
//   - la mêlée depuis la feuille de match (le film n'a pas de clé de mêlée) — et, sur un
//     titre aux mécaniques natives (capability), assassinats et capacités spartanes ;
//   - « objet explosif » et « chute, environnement » depuis la CATÉGORIE de la source du
//     film ; les lignes par arme qu'elles recouvrent (bobines, environnement) s'effacent ;
//   - le reliquat feuille − lignes, s'il est positif, en « Non attribué », toujours dernier.
//
// Aucun libellé : des natures (domain.SquadToolKind*) et, pour une arme, le nom du registre.
// Aucun plafond, aucun regroupement « Autres ».
package teammates

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strconv"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/squadagg"
)

// squadToolInputs regroupe les entrées du builder (règle des 5 paramètres).
type squadToolInputs struct {
	// rows : lignes par arme telles que LoadWeaponKills les rend (sentinelles grenade /
	// mêlée de la feuille comprises — ignorées ici, la feuille arrive par sheet).
	rows []port.WeaponKillRow
	// categories : frags par catégorie de source du film ; nil sans film.
	categories     []port.KillSourceCategoryRow
	playersOrdered []string
	gtByXUID       map[string]string
	// sheet : compteurs de la feuille de match par gamertag (total, mêlée, mécaniques).
	sheet        map[string]domain.FragKillTypeCounts
	hasMechanics bool
}

// toolAcc accumule une ligne pendant la construction.
type toolAcc struct {
	line domain.SquadWeaponToolLine
}

// toolLines indexe les lignes par identifiant stable (nature ou clé d'arme).
type toolLines map[string]*toolAcc

func (t toolLines) add(id string, proto domain.SquadWeaponToolLine, gt string, kills int) {
	if kills <= 0 {
		return
	}
	a := t[id]
	if a == nil {
		proto.KillsByPlayer = make(map[string]int)
		a = &toolAcc{line: proto}
		t[id] = a
	}
	a.line.KillsByPlayer[gt] += kills
	a.line.TotalSquad += kills
}

// weaponLineID : une ligne par clé d'arme ; une ligne sans clé (titre natif, arme hors
// registre) se reconnaît à son identifiant numérique.
func weaponLineID(r port.WeaponKillRow) string {
	if r.WeaponKey != "" {
		return "key:" + r.WeaponKey
	}
	return "id:" + strconv.FormatInt(r.WeaponID, 10)
}

// buildSquadWeaponTools assemble les lignes D8. Pur : aucune IO, aucun log.
// nil si aucune ligne.
func buildSquadWeaponTools(in squadToolInputs) *domain.SquadWeaponTools {
	if len(in.playersOrdered) == 0 {
		return nil
	}
	lines := toolLines{}
	claimed := addCategoryLines(lines, in)
	addWeaponLines(lines, in, claimed)
	addSheetLines(lines, in)
	addUnattributedLine(lines, in)
	if len(lines) == 0 {
		return nil
	}
	out := make([]domain.SquadWeaponToolLine, 0, len(lines))
	for _, a := range lines {
		out = append(out, a.line)
	}
	sortToolLines(out)
	return &domain.SquadWeaponTools{Players: in.playersOrdered, Lines: out}
}

// addCategoryLines pose les lignes « objet explosif » et « chute, environnement » et rend
// les frags qu'elles prennent aux lignes par arme, par (gamertag, clé de registre).
func addCategoryLines(lines toolLines, in squadToolInputs) map[string]int {
	claimed := map[string]int{}
	for _, c := range in.categories {
		gt, ok := in.gtByXUID[c.XUID]
		if !ok || c.Kills <= 0 {
			continue
		}
		var class string
		switch c.Category {
		case domain.KillSourceCategoryExplosiveObject:
			// Couleur du résidu : la maquette C3EW le veut ainsi (un bidon sans clé est
			// « Non attribué » dans la Répartition des frags).
			class = domain.FragClassUnattributed
		case domain.KillSourceCategoryEnvironment:
			class = domain.FragClassEnvironmental
		default:
			continue
		}
		lines.add("kind:"+c.Category, domain.SquadWeaponToolLine{Kind: c.Category, Class: class}, gt, c.Kills)
		if c.WeaponKey != "" {
			claimed[gt+"|"+c.WeaponKey] += c.Kills
		}
	}
	return claimed
}

// addWeaponLines pose une ligne par clé d'arme. Les mécaniques natives attribuées à l'arme
// tenue (MechanicKills) en sont retirées : la feuille de match les sert. Une ligne sans nom
// n'est pas posée — ses frags rejoignent le reliquat « Non attribué ».
func addWeaponLines(lines toolLines, in squadToolInputs, claimed map[string]int) {
	for _, r := range in.rows {
		if r.IsGrenadeMelee {
			continue
		}
		gt, ok := in.gtByXUID[r.XUID]
		if !ok || r.Label == "" {
			continue
		}
		kills := r.Kills - r.MechanicKills
		if r.WeaponKey != "" {
			ck := gt + "|" + r.WeaponKey
			take := min(claimed[ck], max(kills, 0))
			claimed[ck] -= take
			kills -= take
		}
		lines.add(weaponLineID(r), domain.SquadWeaponToolLine{
			Kind: domain.SquadToolKindWeapon, WeaponKey: r.WeaponKey,
			Label: r.Label, LabelEN: r.LabelEN, Class: r.Class,
		}, gt, kills)
	}
}

// addSheetLines pose la mêlée (et, capability native, assassinats et capacités
// spartanes) depuis la feuille de match.
func addSheetLines(lines toolLines, in squadToolInputs) {
	for _, gt := range in.playersOrdered {
		c := in.sheet[gt]
		lines.add("kind:"+domain.SquadToolKindMelee,
			domain.SquadWeaponToolLine{Kind: domain.SquadToolKindMelee, Class: domain.FragClassMelee}, gt, c.Melee)
		if !in.hasMechanics {
			continue
		}
		lines.add("kind:"+domain.SquadToolKindAssassination,
			domain.SquadWeaponToolLine{Kind: domain.SquadToolKindAssassination, Class: domain.FragClassMelee}, gt, c.Assassination)
		lines.add("kind:"+domain.SquadToolKindGroundPound,
			domain.SquadWeaponToolLine{Kind: domain.SquadToolKindGroundPound, Class: domain.FragClassSpartanAbility}, gt, c.GroundPound)
		lines.add("kind:"+domain.SquadToolKindShoulderBash,
			domain.SquadWeaponToolLine{Kind: domain.SquadToolKindShoulderBash, Class: domain.FragClassSpartanAbility}, gt, c.ShoulderBash)
	}
}

// addUnattributedLine pose le reliquat feuille − lignes de chaque joueur, s'il est positif.
func addUnattributedLine(lines toolLines, in squadToolInputs) {
	for _, gt := range in.playersOrdered {
		named := 0
		for _, a := range lines {
			named += a.line.KillsByPlayer[gt]
		}
		lines.add("kind:"+domain.SquadToolKindUnattributed, domain.SquadWeaponToolLine{
			Kind: domain.SquadToolKindUnattributed, Class: domain.FragClassUnattributed,
		}, gt, in.sheet[gt].Total-named)
	}
}

// sortToolLines : total décroissant, « Non attribué » en dernier ; à égalité, la nature
// puis le nom (sortie déterministe).
func sortToolLines(out []domain.SquadWeaponToolLine) {
	sort.Slice(out, func(i, j int) bool {
		ui := out[i].Kind == domain.SquadToolKindUnattributed
		uj := out[j].Kind == domain.SquadToolKindUnattributed
		if ui != uj {
			return uj
		}
		if out[i].TotalSquad != out[j].TotalSquad {
			return out[i].TotalSquad > out[j].TotalSquad
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		if out[i].Label != out[j].Label {
			return out[i].Label < out[j].Label
		}
		return out[i].WeaponKey < out[j].WeaponKey
	})
}

// playerFragCounts : compteurs de la feuille de match d'un joueur sur les matchs partagés,
// mécaniques natives comprises quand le chargeur les a rendues.
func playerFragCounts(
	pts []domain.SquadPerformanceSeriesPoint, mech port.KillMechanicsRow, hasMech bool,
) domain.FragKillTypeCounts {
	counts := aggregateFragCounts(pts)
	if hasMech {
		counts.Assassination = mech.Assassinations
		counts.GroundPound = mech.GroundPound
		counts.ShoulderBash = mech.ShoulderBash
	}
	return counts
}

// loadSquadToolCategories charge les frags par catégorie de source du film. nil (sans
// erreur) si le chargeur ne sait pas les lire ou si le titre n'a pas de film : les deux
// lignes manquent alors, leurs frags restent dans le reliquat. Toute autre erreur est
// journalisée avant la dégradation.
func (s *TeammatesService) loadSquadToolCategories(
	ctx context.Context, sharedMatches, xuids []string,
) []port.KillSourceCategoryRow {
	loader, ok := s.squadLoader.(squadagg.SquadKillSourceCategoryLoader)
	if !ok {
		return nil
	}
	rows, err := loader.LoadKillSourceCategories(ctx, s.titleSlug, port.WeaponKillFilters{
		MatchIDs: sharedMatches,
		XUIDs:    xuids,
	})
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			slog.DebugContext(ctx, "teammates_weapon_tools_categories_unsupported", "title", s.titleSlug)
			return nil
		}
		slog.WarnContext(ctx, "teammates_weapon_tools_categories_load_failed",
			"title", s.titleSlug, "matches", len(sharedMatches), "err", err)
		return nil
	}
	return rows
}

// playersAboveSheet compte les joueurs dont la somme des lignes dépasse le total de leur
// feuille de match (anomalie de données tracée par l'appelant).
func playersAboveSheet(tools *domain.SquadWeaponTools, sheet map[string]domain.FragKillTypeCounts) int {
	n := 0
	for _, gt := range tools.Players {
		sum := 0
		for _, l := range tools.Lines {
			sum += l.KillsByPlayer[gt]
		}
		if sum > sheet[gt].Total {
			n++
		}
	}
	return n
}

// squadSheet : la feuille de match de chaque joueur sur les matchs partagés.
func squadSheet(
	players []string,
	perf map[string][]domain.SquadPerformanceSeriesPoint,
	mechByGT map[string]port.KillMechanicsRow,
) map[string]domain.FragKillTypeCounts {
	sheet := make(map[string]domain.FragKillTypeCounts, len(players))
	for _, gt := range players {
		m, ok := mechByGT[gt]
		sheet[gt] = playerFragCounts(perf[gt], m, ok)
	}
	return sheet
}

// buildSquadToolsSection charge les catégories de source du film, assemble les lignes
// D8 et trace le résultat. Un joueur dont les lignes dépassent sa feuille de match (le
// film compte un frag que la feuille ignore) n'a pas de « Non attribué » : l'écart est
// tracé, pas avalé.
func (s *TeammatesService) buildSquadToolsSection(
	ctx context.Context,
	sc squadScope,
	rows []port.WeaponKillRow,
	sheet map[string]domain.FragKillTypeCounts,
	hasMechanics bool,
) *domain.SquadWeaponTools {
	tools := buildSquadWeaponTools(squadToolInputs{
		rows:           rows,
		categories:     s.loadSquadToolCategories(ctx, sc.sharedMatches, sc.xuids),
		playersOrdered: sc.playersOrdered,
		gtByXUID:       sc.gtByXUID,
		sheet:          sheet,
		hasMechanics:   hasMechanics,
	})
	lineCount, overSheet := 0, 0
	if tools != nil {
		lineCount = len(tools.Lines)
		overSheet = playersAboveSheet(tools, sheet)
	}
	slog.DebugContext(ctx, "teammates_weapon_tools_built", "title", s.titleSlug,
		"lines", lineCount, "players_above_sheet", overSheet)
	return tools
}
