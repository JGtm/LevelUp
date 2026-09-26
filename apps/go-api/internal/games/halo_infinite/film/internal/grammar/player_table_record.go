package grammar

// player_table_record.go — UN ENREGISTREMENT DE SLOT, CHAMP PAR CHAMP (lot 1.5.2 et 1.5.3).
//
// La grammaire, ses fermetures et la provenance de chaque largeur vivent dans l'en-tete de
// `player_table.go` ; ce fichier ne porte que la LECTURE — le decodeur d'un enregistrement, le
// predicat de vacance, et la longueur que la grammaire predit. Il est sorti de `player_table.go`
// au moment ou celui-ci a depasse 500 lignes (CLAUDE.md, seuil 5) : deplacement PUR, aucune
// ligne de lecture n'a change.
//
// Les largeurs constantes restent declarees dans `player_table.go`, a cote de la carte qui les
// justifie : les separer de leur provenance serait exactement la dette que les commentaires de
// ce chantier existent pour empecher.

import (
	"levelup/go-api/internal/games/halo_infinite/film/types"
	"unicode/utf16"
)

// slotVacantBits rend la longueur, en bits, d'un enregistrement de slot ENTIEREMENT A ZERO sur
// un build dont le bloc de personnalisation mesure `persoBits`.
func slotVacantBits(persoBits int) int { return slotVacantHorsPerso + persoBits }

// slotVacant dit si un enregistrement de slot VACANT commence au bit `p`. Le predicat est
// GRAMMATICAL et sans seuil : chaque champ est lu a sa place et doit valoir zero.
//
// Les deux champs laisses libres sont ceux dont la mesure montre qu'ils ne sont PAS nuls dans un
// enregistrement vacant : `sub+0xcb8` (64 bits) et le champ de 6 bits `sub+0xc35`, qui vaut brut
// 1 (donc la valeur signee 0) la ou un slot occupe porte brut 0 (donc -1). C'est le seul champ
// qui distingue un slot vacant d'un bloc de zeros.
func slotVacant(d []byte, p, finBit int) bool {
	if p < 0 || p+slotVacantHorsPerso > finBit {
		return false
	}
	r := &slotReader{br: LecteurSur(d), fin: finBit, ok: true}
	r.br.SetBitPos(p)
	nul := r.bits(3) == 0 && r.bits(32) == 0 && r.bits(2) == 0 && r.bits(48) == 0 &&
		r.bits(slotXUIDBits) == 0 && r.bits(slotMaskPrefixBits+1) == 0 &&
		r.bits(slotListNBits) == 0 && r.bits(slotListMBits) == 0
	if !nul || !r.ok {
		return false
	}
	for reste := slotBloc104Bits; reste > 0; reste -= 64 {
		if r.bits(64) != 0 {
			return false
		}
	}
	return r.bits(16) == 0 && r.bits(slotBloc16Bits/2) == 0 && r.bits(slotBloc16Bits/2) == 0 &&
		r.bits(slotReprBits) == 0 && r.ok
}

// gamertagImprimable : trois caracteres ASCII imprimables au moins, rien d'autre. C'est le
// filtre qui distingue un vrai enregistrement d'une position PARASITE du balayage.
func gamertagImprimable(s string) bool {
	if len(s) < slotNomMinImprimable {
		return false
	}
	for _, r := range s {
		if r < 0x20 || r > 0x7e {
			return false
		}
	}
	return true
}

// slotEnr : un enregistrement decode, plus les quatre nombres LUS DANS LE FLUX dont la longueur
// predite depend.
type slotEnr struct {
	slot                     types.PlayerSlot
	masque, n, m, uniteesNom int
}

// longueurPredite rend la longueur TOTALE que la grammaire predit pour cet enregistrement. Le
// controle de fermeture est qu'elle vaille exactement l'ecart jusqu'au suivant.
func longueurPredite(e slotEnr, persoBits int) int {
	return slotFixeHorsPerso + persoBits + e.masque + e.n*8 + e.m*32 + e.uniteesNom*16
}

// slotReader : un lecteur de bits BORNE. `ok` tombe a faux des qu'une lecture depasserait la
// borne, et ne remonte jamais — un enregistrement lu au-dela du tampon est refuse en entier,
// jamais rendu a moitie.
type slotReader struct {
	br  *Lecteur
	fin int
	ok  bool
}

// bits lit n bits si la borne le permet.
func (r *slotReader) bits(n uint) uint64 {
	if !r.ok || r.br.BitPos()+int(n) > r.fin {
		r.ok = false
		return 0
	}
	return r.br.ReadBits(n)
}

// saute avance de n bits si la borne le permet.
func (r *slotReader) saute(n int) {
	if !r.ok || n < 0 || r.br.BitPos()+n > r.fin {
		r.ok = false
		return
	}
	r.br.Skip(n)
}

// decodeSlot decode un enregistrement de slot a partir de son PREMIER bit.
func decodeSlot(d []byte, debut, finBit, persoBits int) (slotEnr, bool) {
	var e slotEnr
	if debut < 0 {
		return e, false
	}
	r := &slotReader{br: LecteurSur(d), fin: finBit, ok: true}
	r.br.SetBitPos(debut)
	if r.bits(3) != slotBooleensTete {
		return e, false
	}
	e.slot.Bit = debut
	e.slot.Shorts.Tete = uint32(r.bits(32))
	e.slot.Shorts.Deux = uint32(r.bits(2))
	e.slot.SessionToken = r.bits(48)
	e.slot.XUID = r.bits(slotXUIDBits)
	if !decodeSlotListes(r, &e) || !decodeSlotQueue(r, &e, persoBits) {
		return e, false
	}
	e.slot.TotalBits = r.br.BitPos() - debut
	return e, r.ok
}

// decodeSlotListes lit le masque de presence, les deux listes prefixees, le bloc de 104 octets
// et le gamertag.
func decodeSlotListes(r *slotReader, e *slotEnr) bool {
	e.masque = int(r.bits(slotMaskPrefixBits)) + 1
	if !r.ok || e.masque > slotMaskMaxBits {
		return false
	}
	r.saute(e.masque)
	e.n = int(r.bits(slotListNBits))
	if !r.ok || e.n > slotListNMax {
		return false
	}
	r.saute(e.n * 8)
	e.m = int(r.bits(slotListMBits))
	if !r.ok || e.m > slotListMMax {
		return false
	}
	r.saute(e.m * 32)
	r.saute(slotBloc104Bits)
	e.uniteesNom, e.slot.Gamertag = lireGamertag(r)
	return r.ok
}

// lireGamertag lit la chaine de `sub+0xc14` : des unites de 16 bits MSB d'abord, l'ecriture
// s'arretant APRES l'unite nulle — ou a la 16e unite, sans terminateur, quand la chaine est
// pleine. Le nombre d'unites CONSOMMEES est rendu : c'est lui qui entre dans la longueur predite.
func lireGamertag(r *slotReader) (int, string) {
	u := make([]uint16, 0, slotGamertagMaxUnits)
	for k := 0; k < slotGamertagMaxUnits; k++ {
		v := uint16(r.bits(16))
		if !r.ok {
			return 0, ""
		}
		if v == 0 {
			return len(u) + 1, string(utf16.Decode(u))
		}
		u = append(u, v)
	}
	return slotGamertagMaxUnits, string(utf16.Decode(u))
}

// decodeSlotQueue lit tout ce qui suit le gamertag : le CORPS commun, puis le `u32` de queue
// propre a la table de `chunk_00` (`slot+0x1448`).
func decodeSlotQueue(r *slotReader, e *slotEnr, persoBits int) bool {
	if !decodeSlotCorps(r, e, persoBits) {
		return false
	}
	r.saute(slotQueueU32Bits)
	return r.ok
}

// decodeSlotCorps lit le CORPS d'un enregistrement de joueur : le bloc de 16 octets, les deux
// champs larges, les six champs courts, le bloc de personnalisation et le bloc de 44 octets.
//
// IL EST PARTAGE AVEC LE PAQUET DE TYPE 8 (lot 5.17.1). Le corps de `sub+0x000` a `sub+0x142c`
// est le MEME chez les deux ecrivains — `FUN_1407edea8` pour la table de `chunk_00`,
// `FUN_1407eeba4` pour une entree de paquet de type 8 — et les deux se lisent donc par
// `decodeSlotListes` puis cette fonction. Seuls les EN-TETES et la queue diffèrent : 85 bits
// puis le XUID en MSB pour `chunk_00`, `R(1)[R(5)]` puis le XUID en ordre d'octets du flux pour
// le type 8, et le `u32` de `slot+0x1448` n'existe que dans `chunk_00`. Cf. `roster_type8.go`.
func decodeSlotCorps(r *slotReader, e *slotEnr, persoBits int) bool {
	r.saute(slotBloc16Bits)
	e.slot.Shorts.Repr = uint32(r.bits(slotReprBits))
	e.slot.Shorts.Q64 = r.bits(slotQ64Bits)
	e.slot.Shorts.F10 = uint32(r.bits(10))
	e.slot.Shorts.F14 = uint32(r.bits(14))
	e.slot.Shorts.F6 = int(r.bits(6)) - 1
	e.slot.Shorts.F8 = uint32(r.bits(8))
	e.slot.Shorts.F7 = uint32(r.bits(7))
	e.slot.Shorts.F1 = uint32(r.bits(1))
	r.saute(persoBits)
	r.saute(slotBloc44Bits)
	return r.ok
}
