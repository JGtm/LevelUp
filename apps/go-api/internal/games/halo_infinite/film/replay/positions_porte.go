package replay

// positions_porte.go — LA PORTE DES POSITIONS DE BIPEDE : la grammaire de la vie, puis l emprise
// jouee (lot M1 des retours du rejeu, 2026-09-23 ; annexe
// `.ai/V7.5/retours_rejeu_2026-09-23/RAPPORT_positions_limbe.md`).
//
// # LE DEFAUT, MESURE
//
// Les balayages ancres bit a bit publient des « records » dont la position n en est pas une : la
// meme suite de bits revient au meme decalage sur des cartes aux quantifications differentes. Au
// parc du 2026-09-23 : 33 documents sur 111 portent de telles positions ; cote joueurs, 9 premiers
// points de vie, 22 vies d un seul point hors carte, une excursion et 3 derniers points. Le client
// les interpolait sur des silences de plusieurs secondes : Madina97294 « volait » depuis 440 m sous
// la carte jusqu au point d apparition (81c02726).
//
// # CE QUE LE FILM DIT, ET QUE CETTE PORTE APPLIQUE : AUCUNE POSITION DE CORPS AVANT SA CREATION
//
// Le moteur CREE l entite (record `ti=35`), puis replique ses positions dans un paquet ulterieur
// (lot E2 : delai creation -> premiere position valide, mediane 10 a 30 ms). Mesure sur les faits
// de trois films (2026-09-23) : les seules positions anterieures a la creation de leur corps sont
// les deux points aberrants de Madina ; au parc, les 20 vies entierement anterieures a la premiere
// creation de leur slot font TOUTES 1 ou 2 points. D ou, sur un slot dont le PREMIER record de
// creation lu est celui du premier corps (generation 1) :
//
//	R-B1  une vie ouverte par ce record commence a ce record : ses points anterieurs sont ecartes
//	      (`coverage.tracks.avantCreation`) ;
//	R-B2  aucune vie avant ce record : une vie entierement anterieure est ecartee
//	      (`coverage.tracks.viesAvantPremiereCreation`, ses points dans `avantCreation`).
//
// LA GENERATION EST LA GARDE, ET ELLE N EST PAS UN DETAIL. Un slot recycle porte plusieurs corps
// (`084a804d` : `gen=1` en tete de film, `gen=2` apres la 11e minute). Si le PREMIER record lu d un
// slot n est pas `gen=1`, un corps anterieur a existe dont la creation n a pas ete lue (film
// commence apres elle, record manque) : ses positions sont vraies, et la regle se DESARME — elle ne
// coupe jamais une vie a un record qu elle ne sait pas etre le premier. Une creation manquee
// desarme aussi la regle (slot sans record) : les deux sens d erreur sont prudents.
//
// # L ORIGINE NE BOUGE PAS
//
// La porte tourne APRES `ouvrir` : la frame 0, `frameCount` et `originMs` restent lus sur les
// paquets de position du film, filtres ou non. Deplacer l origine decalerait TOUS les calques du
// document pour un point ecarte ; et l origine est une horloge (premier paquet de position moins
// premier paquet du film), controlee par le fil des morts, pas une position publiee.
//
// # PUIS L EMPRISE (repli nomme `repli_position_hors_emprise_ecartee`)
//
// Ce qui reste hors de l emprise jouee (cf. emprise_jouee.go) SANS continuite physique avec une
// position dans l emprise est ecarte et compte (`coverage.tracks.horsEmprise`) : une chute reelle
// hors de la carte reste publiee. L emprise se mesure APRES la regle de creation, sur les positions
// retenues, et elle sert ensuite aux vehicules (positions_porte_vehicules.go) : une seule emprise
// par document.

import (
	"log/slog"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// premiereGenerationDuCorps est la generation que porte le record de creation du PREMIER corps d un
// slot (lot E2 : `gen=1` sur les cinq films mesures ; `gen=2` au premier recyclage, `084a804d`).
//
// C EST UNE CONSTANTE MESUREE, ET SA DERIVE EST PUBLIEE : un build qui numeroterait autrement le
// premier corps desarmerait R-B1 et R-B2 sur tous ses slots, sans rien casser d autre.
// `coverage.tracks.slotsArmes` / `slotsDesarmes` le montrent au document. CRITERE DE SURVEILLANCE :
// au parc du 2026-09-24, 1 slot desarme pour 11 407 armes (111 documents, chronique v69) ; un
// document ou `slotsArmes` tombe a 0 alors que `slotsDesarmes` ne l est pas signe la derive, et
// la constante se re-mesure (lot E2, `identity_registry_creation.go`).
const premiereGenerationDuCorps = 1

// couverturePorte est ce que la porte des positions a ecarte, en positions BRUTES du film (avant
// la decimation sur la grille de frames : un point ecarte ici n atteint aucune frame).
type couverturePorte struct {
	AvantCreation             int
	ViesAvantPremiereCreation int
	HorsEmprise               int
	// SlotsArmes / SlotsDesarmes : slots dont le premier record lu est, ou n est pas, celui du
	// premier corps — la regle de creation ne s applique qu aux premiers. Publies (cf.
	// `premiereGenerationDuCorps`).
	SlotsArmes    int
	SlotsDesarmes int
}

// premiereCreation est, pour un slot, l instant de son premier record de creation et sa generation.
type premiereCreation struct {
	tUS uint64
	gen uint32
}

// passerLaPorteDesPositions applique la grammaire de la vie puis l emprise aux positions TRIEES.
// Rend les positions retenues (nouvelle tranche, l entree n est pas modifiee), l emprise mesuree
// et ce qui a ete ecarte.
func passerLaPorteDesPositions(sorted []grammar.BipedPosition, creations []grammar.BipedCreation,
	fb *fallback.Compteur,
) ([]grammar.BipedPosition, empriseJouee, couverturePorte) {
	var cov couverturePorte
	premieres := premieresCreations(creations)
	for _, c := range premieres {
		if c.gen == premiereGenerationDuCorps {
			cov.SlotsArmes++
		} else {
			cov.SlotsDesarmes++
		}
	}
	retenues := ecarterAvantCreation(sorted, premieres, &cov)
	emprise := empriseDesAxes(axesDesPositions(retenues))
	rejets := rejetsParSlot(retenues, emprise)
	out := make([]grammar.BipedPosition, 0, len(retenues))
	for i, p := range retenues {
		if rejets[i] {
			cov.HorsEmprise++
			continue
		}
		out = append(out, p)
	}
	fb.DeclencheN(fallback.NomPositionHorsEmpriseEcartee, cov.HorsEmprise)
	return out, emprise, cov
}

// rejetsParSlot applique la regle d emprise (hors de l emprise ET isolee, cf. emprise_jouee.go)
// slot par slot, et rend les rejets aux indices de `pos` — trie par instant, tous slots meles.
func rejetsParSlot(pos []grammar.BipedPosition, e empriseJouee) []bool {
	rejets := make([]bool, len(pos))
	if !e.armee {
		return rejets
	}
	indices := map[uint32][]int{}
	for i, p := range pos {
		indices[p.Slot] = append(indices[p.Slot], i)
	}
	for _, idx := range indices {
		duSlot := make([]grammar.BipedPosition, len(idx))
		for k, i := range idx {
			duSlot[k] = pos[i]
		}
		for k, r := range e.rejetsIsoles(duSlot) {
			rejets[idx[k]] = r
		}
	}
	return rejets
}

// premieresCreations rend, par slot, le premier record de creation lu (le plus precoce ; a
// egalite, la plus petite generation — l ordre des records d un chunk n est pas garanti stable).
func premieresCreations(creations []grammar.BipedCreation) map[uint32]premiereCreation {
	out := map[uint32]premiereCreation{}
	for _, c := range creations {
		prev, vu := out[c.Slot]
		if vu && (prev.tUS < c.TimestampUS || (prev.tUS == c.TimestampUS && prev.gen <= c.Generation)) {
			continue
		}
		out[c.Slot] = premiereCreation{tUS: c.TimestampUS, gen: c.Generation}
	}
	return out
}

// ecarterAvantCreation retire les positions anterieures au premier record de creation de leur
// slot, sur les seuls slots ARMES (premier record = premier corps), et compte les vies retirees
// entieres : le bloc ecarte d un slot est une vie a part quand plus de `lifeGapUS` le separe de la
// premiere position retenue (le record de creation tombe dans ce trou et y ecrit une apparition,
// cf. lives_decoupe.go), ou quand le slot n a aucune position retenue.
func ecarterAvantCreation(sorted []grammar.BipedPosition, premieres map[uint32]premiereCreation,
	cov *couverturePorte,
) []grammar.BipedPosition {
	derniereEcartee := map[uint32]uint64{}
	premiereRetenue := map[uint32]uint64{}
	out := make([]grammar.BipedPosition, 0, len(sorted))
	for _, p := range sorted {
		c, ok := premieres[p.Slot]
		if ok && c.gen == premiereGenerationDuCorps && p.TimestampUS < c.tUS {
			cov.AvantCreation++
			derniereEcartee[p.Slot] = p.TimestampUS
			continue
		}
		if _, vu := premiereRetenue[p.Slot]; !vu {
			premiereRetenue[p.Slot] = p.TimestampUS
		}
		out = append(out, p)
	}
	for slot, t := range derniereEcartee {
		suiv, vu := premiereRetenue[slot]
		if !vu || int64(suiv)-int64(t) > lifeGapUS {
			cov.ViesAvantPremiereCreation++
		}
	}
	return out
}

// axesDesPositions rend les trois axes des positions MONDE, TRIES — la forme qu attend
// `empriseDesAxes`.
func axesDesPositions(pos []grammar.BipedPosition) (xs, ys, zs []float32) {
	xs, ys, zs = make([]float32, 0, len(pos)), make([]float32, 0, len(pos)), make([]float32, 0, len(pos))
	for _, p := range pos {
		if !p.HasWorld {
			continue
		}
		xs, ys, zs = append(xs, p.X), append(ys, p.Y), append(zs, p.Z)
	}
	sort.Slice(xs, func(i, j int) bool { return xs[i] < xs[j] })
	sort.Slice(ys, func(i, j int) bool { return ys[i] < ys[j] })
	sort.Slice(zs, func(i, j int) bool { return zs[i] < zs[j] })
	return xs, ys, zs
}

// poserSur publie ce que la porte a ecarte dans la couverture des traces, et le journalise.
func (c couverturePorte) poserSur(tc *TrackCoverage, matchID string) {
	tc.AvantCreation, tc.ViesAvantPremiereCreation, tc.HorsEmprise =
		c.AvantCreation, c.ViesAvantPremiereCreation, c.HorsEmprise
	tc.SlotsArmes, tc.SlotsDesarmes = c.SlotsArmes, c.SlotsDesarmes
	if c.AvantCreation+c.HorsEmprise+c.SlotsDesarmes == 0 {
		return
	}
	// JOURNALISE, JAMAIS AVALE (regle n° 3 du depot) : ces positions ne sont pas publiees.
	slog.Info("rejeu : porte des positions de bipede",
		"match_id", matchID, "avantCreation", c.AvantCreation,
		"viesAvantPremiereCreation", c.ViesAvantPremiereCreation, "horsEmprise", c.HorsEmprise,
		"slotsArmes", c.SlotsArmes, "slotsDesarmes", c.SlotsDesarmes)
}
