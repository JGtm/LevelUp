package grammar

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// delta_biped_walk.go — LE MARCHEUR DE RECORDS DELTA BIPEDE, ET IL EST UNIQUE.
//
// # CE QU'IL REMPLACE, ET POURQUOI
//
// Neuf balayages portaient, chacun pour son compte, la MEME triple boucle : les chunks de
// replication, les paquets delta de chaque chunk, puis le curseur de bits qui ancre les records
// bipedes un a un. Le squelette etait identique CARACTERE POUR CARACTERE d'un fichier a l'autre
// (verifie par `diff` entre `camo_state.go` et `grapple_state.go` a l'audit du 2026-09-05) ;
// seuls le crochet installe et le corps de visite differaient.
//
// LE COUT DE CETTE DISPERSION A DEJA ETE PAYE UNE FOIS, et le depot le chiffre :
// `film_context.go:32-42` raconte que la regle de decoupage d'i0 n'etait ecrite qu'au site des
// positions, que les canaux delta la re-detectaient pour leur compte, et que « sur Live Fire,
// 27 enregistrements sur 267 400 (mesure du 2026-09-03) qui n'appartiennent pas a l'arene
// jouee » passaient la porte. Le CONTEXTE a ete centralise au lot 3 ; la BOUCLE, elle, restait
// en neuf exemplaires — donc la prochaine regle de porte (bande de slots, filtre de region,
// borne de deroulage) se serait payee neuf fois, avec neuf preuves a refaire.
//
// # DEUX ETAGES, PARCE QU'IL Y A DEUX BESOINS
//
//   - `walkDeltaBipedPayload` marche UN payload deja en main. C'est l'etage que `ScanBipedRecords`
//     utilise : il est PUR, sans film ni chunk, et c'est le coeur testable du decodeur.
//   - `walkDeltaBipedRecords` ajoute les deux boucles externes (chunks du contexte de film, puis
//     paquets delta) et delegue au premier. C'est l'etage des huit balayages de canal.
//
// # CE QUE LE MARCHEUR NE DECIDE PAS
//
// Il n'interprete AUCUN composant : il ancre, il publie, il avance. L'avance est toujours
// `i0 + lay.TotalBits()` — un record reconnu n'est jamais re-balaye en chevauchement — et un
// echec d'ancrage avance d'UN bit. C'est la grammaire d'origine, a la ligne pres.
//
// Garde-rail : `delta_biped_walk_guard_test.go` interdit qu'un dixieme site recalcule le seuil
// `bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt` ou rappelle `matchBipedHeader` hors d'ici.

// deltaBipedRecord est UN record bipede ancre par le marcheur : ou il commence, a qui il
// appartient, et ce que son masque annonce.
type deltaBipedRecord struct {
	// Payload est le payload du paquet porteur, et Total sa longueur en BITS.
	Payload []byte
	Total   int
	// I0 est le bit du composant i0 (la position), c'est-a-dire la fin de l'en-tete.
	I0 int
	// Slot est le slot du bipede, et Mask la liste des index de composants que le masque
	// annonce (le premier est i0 lui-meme).
	Slot uint32
	Mask []int
	// Gen est la GENERATION du handle (lot J5.2) : la vie du corps est la paire (Slot, Gen), cf.
	// [deltaBipedRecord.Vie]. Le slot seul designe tous les corps successifs du slot.
	Gen uint32
	// Chunk et Packet situent le record dans le film. Ils sont a ZERO quand la marche porte
	// sur un payload seul (`walkDeltaBipedPayload`), qui ne sait rien du film.
	Chunk  int
	Packet FilmPacket
}

// deltaBipedMinRecord est la largeur MINIMALE d'un record bipede : l'en-tete, le masque le plus
// court, et le composant i0. C'est la borne de la boucle de curseur — au-dela, il ne reste plus
// assez de bits pour qu'un record tienne.
//
// C'EST LE SEUL SITE QUI COMPOSE CE SEUIL. Il l'etait en neuf exemplaires jusqu'au 2026-09-05
// (lot E, item E.4).
func deltaBipedMinRecord(i0Bits int) int {
	return bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + i0Bits
}

// walkDeltaBipedPayload ancre tous les records bipedes d'UN payload de paquet delta et appelle
// `visit` pour chacun, dans l'ordre du flux.
//
// `gens` est le FILTRE DE GENERATION (lot J5.2, DT-8), passe tel quel a `matchBipedHeader` : les
// huit balayages de canal recoivent les generations VIVANTES du film datees a l instant du paquet
// ([FilmContext.GenerationsVivantesA], lot R2-bis), `ScanBipedRecords` celles de ses options
// (datees par paquet dans `scanBipedChunks`, lot R2). nil : tous les slots dans le repli nomme
// (generation 1).
func walkDeltaBipedPayload(
	pay []byte, slots SlotBand, lay profile.I0Layout, gens *GenerationsVivantes, visit func(deltaBipedRecord),
) {
	total := len(pay) * 8
	i0Bits := lay.TotalBits()
	minRecord := deltaBipedMinRecord(i0Bits)
	for p := 0; p+minRecord <= total; {
		i0, slot, idx, ok := matchBipedHeader(pay, p, total, slots, gens, lay)
		if !ok {
			p++
			continue
		}
		gen := LireHandleDelta(pay, i0-bipedHeaderBits-bipedIndexBits*len(idx)).Gen
		visit(deltaBipedRecord{Payload: pay, Total: total, I0: i0, Slot: slot, Gen: gen, Mask: idx})
		p = i0 + i0Bits // pas de re-scan chevauchant
	}
}

// walkDeltaBipedRecords parcourt les chunks de replication du contexte, leurs paquets DELTA, et
// ancre les records bipedes de chacun. `visit` recoit le record ENRICHI de son chunk et de son
// paquet, ce que la marche d'un payload seul ne peut pas fournir.
//
// Un chunk que le film ne porte pas est SAUTE sans erreur : `ChunkAt` rend un predicat, pas une
// erreur, et `film_chunks.go:96-100` documente que la bande de slots demande deliberement le
// chunk d'apres le dernier.
func walkDeltaBipedRecords(
	fc *FilmContext, chunks []int, slots SlotBand, lay profile.I0Layout, visit func(deltaBipedRecord),
) {
	// LES GENERATIONS VIVANTES DU FILM, et non plus `true` en dur (constat GB-1) : un corps de generation
	// >= 2 est lu par les huit canaux comme par les positions. DATEES A L INSTANT DU PAQUET (lot
	// R2-bis) : un en-tete dont le corps n est pas encore cree n est la replication d aucun corps
	// (generations_vivantes.go).
	for _, c := range chunks {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			gens := fc.GenerationsVivantesA(pk.TimestampUS)
			walkDeltaBipedPayload(pk.Payload(data), slots, lay, gens, func(r deltaBipedRecord) {
				r.Chunk, r.Packet = c, pk
				visit(r)
			})
		}
	}
}

// Vie rend la cle de vie du corps qui porte ce record : (slot, generation du handle).
func (r deltaBipedRecord) Vie() types.LifeKey { return types.LifeKey{Slot: r.Slot, Gen: r.Gen} }
