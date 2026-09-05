package filmdec

import "levelup/go-api/internal/analysis/filmsource"

// slot_band_observed.go — L'AUTRE REGLE DE BANDE : les slots REELLEMENT OBSERVES, sans
// comblement.
//
// # DEUX REGLES, DEUX MONDES, ET LE CHOIX SE FAIT SUR LA VIE DE L'OBJET
//
// L'ancrage d'un record delta d'objet du monde repose sur une BANDE DE SLOTS relevee aux
// images-cles : un record n'est reconnu que si son slot y appartient. Deux regles de bande
// existent, et elles ne repondent pas a la meme question.
//
//	COMBLEE (`worldObjectSlotBand`)   toute la plage [min, max] des slots vus, trous compris,
//	                                  moins ceux vus porter un autre archetype. Elle existe pour
//	                                  les objets NOMBREUX ET EPHEMERES — projectiles,
//	                                  equipements, armes au sol : un projectile vit moins d'une
//	                                  seconde et les images-cles sont espacees de 20 s, donc un
//	                                  slot peut servir entre deux images-cles sans jamais y
//	                                  apparaitre. Sans comblement on decode 57 vies au lieu de
//	                                  580 (mesure du 2026-07-26).
//	OBSERVEE (ce fichier)             les seuls slots vus, meme exclusion. Pour un objet RARE ET
//	                                  DURABLE — objectifs geres (ti=11), proprietes d'objet gere
//	                                  (ti=13) — le comblement N'A RIEN A RATTRAPER : l'objet vit
//	                                  toute la partie et apparait a chaque image-cle.
//
// # CE QUE LE COMBLEMENT COUTE QUAND IL N'APPORTE RIEN (mesure du 2026-09-01, 13 films)
//
// Elargir la fenetre d'ancrage sans rien rattraper ne fait qu'accepter de FAUX records — le
// chainage des records delta (leur fin tombe-t-elle sur un en-tete de record valide ?) le dit,
// contre un plancher de 3 % sur des positions arbitraires :
//
//	ti=11   5,6 % sur 1 616 281 marches (comblee)  ->  29,3 % sur 19 666 (observee)
//	ti=13   6,3 % sur   278 670 marches (comblee)  ->  43,7 % sur 27 409 (observee)
//
// Et la ou le comblement n'ajoute AUCUN slot, rien ne bouge : les deux Strongholds du corpus
// gardent leurs 26 slots et leur taux au dixieme pres. Le comblement ne divise donc le chainage
// que la ou il invente des slots. Instrument : `TestObjectifTi11DeltaControleTi13`.

// observedSlotBand rend les slots d'un archetype REELLEMENT OBSERVES aux images-cles, SANS
// combler les trous — cf. l'en-tete pour le depart entre les deux regles.
func observedSlotBand(film *filmsource.Film, typeIndex int) map[uint32]bool {
	seen, others := map[uint32]bool{}, map[uint32]bool{}
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
					continue
				}
				others[uint32(r.Slot)] = true
			}
		}
	}
	// UN SLOT VU PORTER AUTRE CHOSE NE PEUT PAS PORTER CET ARCHETYPE : meme exclusion que
	// `slotBandExcluding`, c'est le comblement qui saute, pas la prudence.
	for s := range others {
		delete(seen, s)
	}
	return seen
}
