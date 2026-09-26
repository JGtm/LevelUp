package replay

// zone_states_lettres.go — LA LETTRE D'UNE ZONE (A, B, C), et les trois portes qui la taisent.
//
// DEPLACEMENT PUR depuis `zone_states_owner.go` (lot 5.6) : ce fichier avait franchi les 500
// lignes en recevant le cablage du canal POUSSEUR, et le ratchet de taille l'a refuse. La
// lettre est le morceau le plus autonome du volet — elle ne lit que la carte des slots de
// jauge et le catalogue —, donc c'est elle qui sort. Aucune ligne n'a change.

import "sort"

// zoneLetterMax est le nombre de lettres que le HUD du jeu affiche sur une carte a bases
// simultanees : A, B, C. Au-dela, le fallback se tait — un « D » serait une invention, et les
// cartes mesurees n'en portent de toute facon que trois (8 cartes, phase 0.2).
const zoneLetterMax = 3

// zoneLetterRanks rend le rang de lettre de chaque zone : les zones RANGEES PAR NUMERO DE SLOT
// `ti=13` CROISSANT, chacune prenant son rang (0 = A, 1 = B, 2 = C).
//
// POURQUOI CET ORDRE-LA, ET CE QU'IL VAUT (mesure du 2026-08-24, plan
// `.ai/V7.5/replay2d/PLAN_LETTRES_BASES_FALLBACK.md`, phase 0.2). La lettre affichee en jeu
// n'existe dans AUCUNE donnee decodee : ni le catalogue de formes, ni la variante, ni — verdict
// de la RE Ghidra du meme jour — le binaire, ou elle vient d'un script de mode. Ce qui se mesure,
// c'est que l'ordre des slots de jauge est REPRODUCTIBLE : sur 8 cartes et 17 films de Bastion,
// les films d'une meme carte rendent 8/8 la MEME permutation zone -> rang. Les slots d'une carte
// forment des blocs reguliers de pas 5 (un par zone : proprietaire, canal neutre, jauge) — l'ordre
// des jauges est donc l'ordre d'allocation du moteur au chargement, pas une coincidence de vote.
// Un ordre stable suffit a dire A, B ou C ; que ce soient LES lettres du jeu appartient au releve
// Theater de l'utilisateur, et ce paquet ne le pretend nulle part.
//
// TROIS PORTES FERMEES, chacune parce que l'ouvrir publierait du faux :
//
//	la BIJECTION   sans une zone appariee PAR zone du catalogue, une zone muette decale les
//	               lettres de toutes les suivantes. Le cas existe (un film du corpus n'a aucune
//	               capture attribuee) et il est invisible a l'oeil : rien plutot qu'un decalage.
//	l'ALPHABET     au-dela de trois zones, la lettre suivante n'existe pas dans le HUD.
//	la COLLINE     un mode a colline n'a qu'une zone active a la fois et aucune lettre. La porte
//	               est fermee ici EN PLUS du chemin (le volet colline ne passe pas par cette
//	               fonction) : une ceinture, parce qu'un mode a colline QUI porterait aussi des
//	               captures nommees retomberait sinon dans ce chemin sans que rien ne le dise.
func zoneLetterRanks(gauge map[int]uint32, catalog int, hill bool) map[int]int {
	if hill || catalog <= 0 || catalog > zoneLetterMax || len(gauge) != catalog {
		return nil
	}
	refs := make([]int, 0, len(gauge))
	for ref := range gauge {
		refs = append(refs, ref)
	}
	// ORDRE TOTAL : le slot de jauge, PUIS la reference de zone.
	//
	// LE SLOT SEUL N'EST PAS UNE CLE (correction du 2026-09-02, item 0.4bis etendu de
	// PLAN_CUISSON_PERF). `pairGaugeSlots` garantit AU PLUS UNE jauge PAR ZONE — jamais au plus
	// une zone par jauge : rien n'empeche un meme slot d'etre l'argmax de deux zones (a la
	// difference d'`electZoneOwners`, qui tient un `held` par canal). Deux zones ex aequo sur le
	// slot laissaient donc l'ordre d'iteration de la MAP `gauge`, tire au sort a chaque
	// execution, decider quelle zone s'appelle A et laquelle s'appelle B. La reference de zone
	// ferme l'egalite avec une donnee de l'element, jamais un rang d'iteration.
	sort.Slice(refs, func(i, j int) bool {
		if gauge[refs[i]] != gauge[refs[j]] {
			return gauge[refs[i]] < gauge[refs[j]]
		}
		return refs[i] < refs[j]
	})
	out := make(map[int]int, len(refs))
	for i, ref := range refs {
		out[ref] = i
	}
	return out
}

// zoneSpanCtx porte ce qu'un intervalle doit connaitre (regle des 5 parametres).
type zoneSpanCtx struct {
	frames int
	// teams est l'ensemble des camps du roster. Vide : seuls 0 et 1 — les deux valeurs
	// MESUREES du canal — sont acceptes comme camps.
	teams map[uint64]bool
}

// zoneTeamSet rend les camps du roster, en valeurs de canal.
func zoneTeamSet(teams map[string]int) map[uint64]bool {
	out := map[uint64]bool{}
	for _, t := range teams {
		if t >= 0 {
			out[uint64(t)] = true
		}
	}
	return out
}
