// Package squadagg — weapon_tools.go : « OUTILS DE DESTRUCTION », chaque frag nommé (décision D8 du
// plan .ai/PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26.md), commun à l'Escouade et à la page Sessions.
//
// # Ce que ce builder est, et pourquoi il n'est pas fragdist
//
// `fragdist.Build` sert la « Répartition des frags » : une ventilation PAR CLASSE, réconciliée aux
// compteurs de la feuille de match, où un type de grenade que le film et la feuille comptent
// différemment retombe en une classe « Grenade » sans détail. Les « Outils de destruction » posent une
// autre question — avec quoi chacun a frappé, outil par outil — et ne changent RIEN à la répartition :
//
//   - une ligne par clé d'arme (film pour un titre qui le décode, table native sinon) ;
//     les grenades par type quand le FILM les type, sinon une ligne « Grenade » au total
//     de la feuille de match (titre sans film : jamais dans le reliquat) ;
//   - la mêlée depuis la feuille de match (le film n'a pas de clé de mêlée) — et, sur un
//     titre aux mécaniques natives (capability), assassinats et capacités spartanes ;
//   - « objet explosif » et « chute, environnement » depuis la CATÉGORIE de la source du
//     film ; les lignes par arme qu'elles recouvrent (bobines, environnement) s'effacent ;
//   - le reliquat feuille − lignes, s'il est positif, en « Non attribué », toujours dernier.
//
// Aucun libellé : des natures (domain.SquadToolKind*) et, pour une arme, le nom du registre. Aucun
// plafond, aucun regroupement « Autres ». Pur : aucune IO, aucun journal (les lectures et leurs traces
// restent aux pages).
package squadagg

import (
	"sort"
	"strconv"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// WeaponToolInputs regroupe les entrées du builder (règle des 5 paramètres).
type WeaponToolInputs struct {
	// Rows : lignes par arme telles que LoadWeaponKills les rend (sentinelles grenade / mêlée de la
	// feuille comprises — ignorées ici, la feuille arrive par Sheet).
	Rows []port.WeaponKillRow
	// Categories : frags par catégorie de source du film ; nil sans film.
	Categories []port.KillSourceCategoryRow
	// PlayersOrdered : gamertags, joueur de la page d'abord. GtByXUID : xuid -> gamertag.
	PlayersOrdered []string
	GtByXUID       map[string]string
	// Sheet : compteurs de la feuille de match par gamertag (total, mêlée, mécaniques).
	Sheet        map[string]domain.FragKillTypeCounts
	HasMechanics bool
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

// BuildWeaponTools assemble les lignes D8. nil si aucun joueur ou aucune ligne.
func BuildWeaponTools(in WeaponToolInputs) *domain.SquadWeaponTools {
	if len(in.PlayersOrdered) == 0 {
		return nil
	}
	lines := toolLines{}
	claimed := addCategoryLines(lines, in)
	filmGrenades := addWeaponLines(lines, in, claimed)
	addSheetLines(lines, in, filmGrenades)
	addUnattributedLine(lines, in)
	if len(lines) == 0 {
		return nil
	}
	out := make([]domain.SquadWeaponToolLine, 0, len(lines))
	for _, a := range lines {
		out = append(out, a.line)
	}
	sortToolLines(out)
	return &domain.SquadWeaponTools{Players: in.PlayersOrdered, Lines: out}
}

// addCategoryLines pose les lignes « objet explosif » et « chute, environnement » et rend
// les frags qu'elles prennent aux lignes par arme, par (gamertag, clé de registre).
func addCategoryLines(lines toolLines, in WeaponToolInputs) map[string]int {
	claimed := map[string]int{}
	for _, c := range in.Categories {
		gt, ok := in.GtByXUID[c.XUID]
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
//
// GRENADES : le détail par type ne vient QUE du film (FromDamageSource). Une ligne de
// grenade d'une autre provenance (table native d'un titre sans film) n'est pas posée : les
// grenades de ce joueur passent par la ligne « Grenade » de la feuille (addSheetLines),
// jamais par le reliquat. Rend, par gamertag, les frags de grenade typés par le film.
func addWeaponLines(lines toolLines, in WeaponToolInputs, claimed map[string]int) map[string]int {
	filmGrenades := map[string]int{}
	for _, r := range in.Rows {
		if r.IsGrenadeMelee {
			continue
		}
		gt, ok := in.GtByXUID[r.XUID]
		if !ok || r.Label == "" {
			continue
		}
		if r.Class == domain.FragClassGrenade {
			if !r.FromDamageSource {
				continue
			}
			filmGrenades[gt] += max(r.Kills, 0)
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
	return filmGrenades
}

// addSheetLines pose la mêlée, la ligne « Grenade » d'un joueur sans détail typé du film
// (total de la feuille) et, capability native, assassinats et capacités spartanes — tout
// depuis la feuille de match.
func addSheetLines(lines toolLines, in WeaponToolInputs, filmGrenades map[string]int) {
	for _, gt := range in.PlayersOrdered {
		c := in.Sheet[gt]
		lines.add("kind:"+domain.SquadToolKindMelee,
			domain.SquadWeaponToolLine{Kind: domain.SquadToolKindMelee, Class: domain.FragClassMelee}, gt, c.Melee)
		if filmGrenades[gt] == 0 {
			lines.add("kind:"+domain.SquadToolKindGrenade,
				domain.SquadWeaponToolLine{Kind: domain.SquadToolKindGrenade, Class: domain.FragClassGrenade}, gt, c.Grenade)
		}
		if !in.HasMechanics {
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
func addUnattributedLine(lines toolLines, in WeaponToolInputs) {
	for _, gt := range in.PlayersOrdered {
		named := 0
		for _, a := range lines {
			named += a.line.KillsByPlayer[gt]
		}
		lines.add("kind:"+domain.SquadToolKindUnattributed, domain.SquadWeaponToolLine{
			Kind: domain.SquadToolKindUnattributed, Class: domain.FragClassUnattributed,
		}, gt, in.Sheet[gt].Total-named)
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

// PlayersAboveSheet compte les joueurs dont la somme des lignes dépasse le total de leur
// feuille de match (anomalie de données tracée par l'appelant).
func PlayersAboveSheet(tools *domain.SquadWeaponTools, sheet map[string]domain.FragKillTypeCounts) int {
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
