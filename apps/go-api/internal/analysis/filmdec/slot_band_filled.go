package filmdec

// slot_band_filled.go — LA BANDE COMBLEE DES OBJETS DU MONDE.
//
// DEPLACEMENT PUR depuis `projectiles.go` (lot 1 de PLAN_CUISSON_PERF, 2026-09-02) : la
// migration des balayages vers un film deja charge y a ajoute une trentaine de lignes et pousse
// le fichier au-dessus du seuil de 500 (CLAUDE.md n°5). Les deux fonctions ci-dessous sont
// reprises TELLES QUELLES, commentaires compris — aucune ligne de logique n'a change.
//
// ELLES ONT LEUR PLACE ICI ET PAS AILLEURS : `slot_band_observed.go` porte l'AUTRE regle de
// bande et compare deja les deux dans son en-tete. Les lire cote a cote est ce qui evite qu'on
// applique la mauvaise a un archetype (la lecon mesuree du 2026-07-26, rappelee ci-dessous).

import "levelup/go-api/internal/analysis/filmsource"

// worldObjectSlotBand rend les slots utilisables pour un archétype, lus dans les keyframes.
//
// NI L'UN NI L'AUTRE DES DEUX EXTRÊMES N'EST JUSTE — c'est la leçon du 2026-07-26, obtenue en
// les essayant tous les deux :
//
//	COMBLER la plage (min..max), comme pour le bipède : contamine massivement. Le bipède a une
//	plage remplie à 91 %, mais le projectile à 8 %, l'équipement à 23 %, le corps rigide à 10 %
//	-- et les plages de ti=37 [1462,2660] et ti=38 [1280,2641] se recouvrent presque entièrement.
//	Symptôme mesuré : les trois archétypes rendaient des chiffres identiques à 1 % près sur cinq
//	critères indépendants (nombre de vies, durée, immobilité, convergence, vitesse d'approche).
//
//	S'EN TENIR À L'OBSERVÉ : rate l'essentiel. Un projectile vit moins d'une seconde et les
//	keyframes sont espacés de 20 s : seuls 19 slots de projectile y apparaissent, contre 57 vies
//	décodées au lieu de 580.
//
// LA FORME JUSTE est donc : combler la plage de l'archétype, PUIS retirer tout slot vu porter un
// AUTRE archétype. On récupère la couverture sans la contamination, et le retrait est fondé sur
// une observation, pas sur une heuristique.
func worldObjectSlotBand(film *filmsource.Film, typeIndex int) map[uint32]bool {
	seen := map[uint32]bool{}
	others := map[uint32]bool{}
	for _, c := range FilmChunkNumbers(film) {
		data, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				if r.TI == typeIndex {
					seen[uint32(r.Slot)] = true
				} else {
					others[uint32(r.Slot)] = true
				}
			}
		}
	}
	return slotBandExcluding(seen, others)
}

// slotBandExcluding applique la règle ci-dessus à des ensembles DÉJÀ RELEVÉS : combler la plage
// de l'archétype, puis retirer tout slot vu porter un autre archétype.
//
// ELLE EST EXTRAITE PARCE QU'UN SECOND APPELANT LA LIT (`ScanFilmWorldObjectKeyframes`, qui
// relève bande et recensement dans la MÊME marche d'images-clés). Deux copies d'une règle de
// bande divergeraient au premier correctif — et celle-ci a déjà été corrigée une fois.
func slotBandExcluding(seen, others map[uint32]bool) map[uint32]bool {
	band := fillSlotBand(seen)
	for s := range others {
		delete(band, s)
	}
	return band
}
