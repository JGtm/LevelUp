package filmdec

// r12_palette_research_test.go — LA PALETTE DE CAPACITES, SANS ARTEFACT DE REJEU.
//
// Extrait de `r12_socle_research_test.go` quand celui-ci a franchi le seuil de 500 lignes.
// Une seule responsabilite : NOMMER un rang i48 sur un film dont la carte n'est pas au
// catalogue de bornes, donc dont l'artefact de rejeu n'existe pas et ne porte aucun
// `abilityLabels`.

// --- LA PALETTE, RECOPIEE DU MANIFESTE DU TITRE -------------------------------------------
//
// Source : `config/titles/halo_infinite/mappings/replay_labels.toml`, sections
// `[[ability_palettes]]`. RECOPIE ASSUMEE : un `*_research_test.go` ne charge pas la config
// du titre (elle passe par le registre de titres, qui ouvre des chemins de production). La
// recopie est BORNEE aux deux familles etablies et le nom EN est la cle, jamais un numero —
// le propulseur vaut 5 en famille A et 21 en famille B, un litteral en dur rendrait un
// instrument muet sur l'autre famille (le piege qui avait rendu le translocateur invisible).

import "fmt"

type r12Palette struct {
	id      string
	markers []int
	ranks   map[int]string
}

var r12Palettes = []r12Palette{
	{
		id:      "famille_a",
		markers: []int{1, 2, 4, 5, 6, 8, 9, 10, 11, 12, 23},
		ranks: map[int]string{
			1: "Threat Sensor", 2: "Drop Wall", 4: "Grappleshot", 5: "Thruster",
			6: "Repulsor", 8: "Active Camouflage", 9: "Overshield",
			11: "Quantum Translocator", 12: "Threat Seeker", 23: "Repair Field",
		},
	},
	{
		id:      "famille_b",
		markers: []int{19, 20, 21, 22},
		ranks:   map[int]string{20: "Grappleshot", 21: "Thruster"},
	},
}

// r12ClassifyPalette applique la regle de `replay.classifyAbilityPalette` : au moins 10
// lectures, et 90 % d'entre elles portant les marqueurs d'UNE palette. Une signature ambigue
// ne nomme RIEN — nommer au juge mettrait « grappin » sur un propulseur.
//
// UNE SEULE DIFFERENCE AVEC LA REGLE DE PRODUCTION, ECRITE ICI ET NON GOMMEE : les lectures
// `AbilitySetNoRank` (la PORTE OUVERTE — « ce joueur ne porte rien ») sont retirees du
// DENOMINATEUR. Elles ne portent aucun rang, donc elles ne peuvent voter pour aucune palette,
// et les compter contre la purete penalise un film d'autant plus qu'il est bien lu. Mesure
// sur `215e7022` : 31 lectures a rang sur 35, toutes marqueurs de la famille A — la regle de
// production rendrait 31/35 = 88,6 % et ne classerait PAS, la regle d'ici rend 31/31 = 100 %.
// Aucune autre difference : memes marqueurs, meme seuil, meme refus de nommer en cas
// d'ambiguite.
func r12ClassifyPalette(ranks []r12RankRead) *r12Palette {
	total := 0
	counts := make([]int, len(r12Palettes))
	for _, r := range ranks {
		if r.Rank == AbilitySetNoRank {
			continue
		}
		total++
		for i, p := range r12Palettes {
			if r12Contains(p.markers, r.Rank) {
				counts[i]++
				break
			}
		}
	}
	if total < 10 {
		return nil
	}
	for i := range r12Palettes {
		if float64(counts[i])/float64(total) >= 0.90 {
			return &r12Palettes[i]
		}
	}
	return nil
}

func r12Contains(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

// r12RankOf rend le rang qui porte l'equipement nomme (nom EN de la palette), ou -1.
func (p *r12Palette) r12RankOf(en string) int {
	if p == nil {
		return -1
	}
	for r, n := range p.ranks {
		if n == en {
			return r
		}
	}
	return -1
}

// r12LabelOf nomme un rang, ou rend son numero quand la palette ne le nomme pas.
func (p *r12Palette) r12LabelOf(rank int) string {
	if rank == AbilitySetNoRank {
		return "porte ouverte : rien de porte"
	}
	if p != nil {
		if n, ok := p.ranks[rank]; ok {
			return n
		}
	}
	return fmt.Sprintf("rang %d (non nomme)", rank)
}

// r12PalID rend l'identifiant de la palette retenue, ou une mention explicite quand aucune ne
// l'a ete — un journal muet se lirait comme « pas de capacite lue ».
func r12PalID(p *r12Palette) string {
	if p == nil {
		return "non classee"
	}
	return p.id
}
