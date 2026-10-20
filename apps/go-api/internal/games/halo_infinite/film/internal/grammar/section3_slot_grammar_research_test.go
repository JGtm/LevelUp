//go:build research

package grammar

// section3_slot_grammar_research_test.go — LA GRAMMAIRE COMPLETE D'UN ENREGISTREMENT DE SLOT.
//
// ## D'OU VIENT CETTE GRAMMAIRE : DU DESASSEMBLAGE, PAS DES OCTETS
//
// `FUN_1407ecb08` serialise l'en-tete d'un slot (85 bits puis le XUID sur 64 : phase 1), puis
// delegue a `FUN_1407edea8(sub = slot+0x18, writer)`. Le desassemblage de CETTE fonction, lu au
// HTTP Ghidra le 2026-09-12, donne l'ordre d'ecriture et la largeur de CHAQUE champ :
//
//	CALL FUN_1407ecd00                         -> trois listes prefixees (voir plus bas)
//	LEA R8,[RDI+0xc48] ; R9D=0x340             -> 832 bits  = 104 octets bruts
//	LEA R8,[RDI+0xc14] ; R9D=0x10   FUN_1407ece18 -> CHAINE UTF-16 terminee par NUL, <= 16 unites
//	LEA R8,[RDI+0xc38] ; R9D=0x80              -> 128 bits  = 16 octets bruts
//	MOV R8D,[RDI+0xcb0] ; RDX=DAT_143686818    -> 32 bits, etiquete "desired-representation"
//	LEA R8,[RDI+0xcb8] ; R9D=0x40              -> 64 bits
//	LEA R8,[RDI+0xc12]              FUN_1407edcc4 -> 10 bits (ushort)
//	LEA R8,[RDI+0xc36]              FUN_1407edd3c -> 14 bits (ushort)
//	MOV R8B,[RDI+0xc35]             FUN_1407eddb4 -> 6 bits, VALEUR + 1 (char signe : -1 possible)
//	MOVZX R8D,[RDI+0xc10]           inline        -> 8 bits
//	LEA R8,[RDI+0xc34]              FUN_1407ede30 -> 7 bits (octet)
//	MOVZX EDX,[RDI+0xc11] & 1       inline        -> 1 bit
//	LEA R8,[RDI+0xcc0] ; R9D=0x39e0            -> 14 816 bits = 1 852 octets bruts
//	LEA R8,[RDI+0x1400] ; R9D=0x160            -> 352 bits   = 44 octets bruts
//
// `FUN_1407ecd00` (desassemble) : `FUN_1407ecd78(sub+0x000)` ecrit un MASQUE de 2 048 bits
// prefixe de 11 bits (`FUN_1424ccf94` ecrit `rang_du_bit_haut` puis les bits un par un) ; puis
// `FUN_1411b1a24` ecrit N sur 12 BITS (source `sub+0x100`) suivi de N octets (`sub+0x108`) ;
// puis `FUN_1411b198c` ecrit M sur 8 BITS (source `sub+0x908`) suivi de M mots de 32 bits.
//
// ## LA FERMETURE ARITHMETIQUE, QUI DIT QUE LA STRUCTURE EST COMPRISE ET PAS DEVINEE
//
//	0xc14 + 16 unites x 2 o = 0xc34   -> le champ qui suit la chaine commence pile a sa fin
//	0xc48 + 4 + 5 x 0x14    = 0xcb0   -> le bloc de 104 o est « un compte + 5 x 20 o »
//	                                    (lu chez le serialiseur reseau FUN_140969c54)
//	0xcc0 + 1 852           = 0x13fc  -> le bloc de personnalisation
//	slot+0x18 + 0x1430      = 0x1448  -> l'offset du dernier u32, et 0x1448 + 4 = 0x1450,
//	                                    le pas de la table. AUCUN TROU dans la structure.
//
// ## LES CONTROLES, TOUS ECRITS AVANT LA MESURE
//
//	G-CLO  FERMETURE DE LONGUEUR, oracle 100 % interne. La grammaire ci-dessus predit la
//	       longueur TOTALE d'un enregistrement a partir de quatre nombres lus dans le flux
//	       (le compte du masque, N, M, la longueur de la chaine). Cette prediction doit valoir
//	       EXACTEMENT l'ecart mesure jusqu'au debut de l'enregistrement suivant. Une seule
//	       largeur fausse casse la prediction. Critere : 100 % des ecarts mesurables.
//	G-REF  LE GAMERTAG. La chaine UTF-16 decodee doit etre le gamertag du joueur dont le XUID
//	       est dans le MEME enregistrement. Reference externe : le `roster` des documents de
//	       rejeu deja produits (`CHUNK00_REPLAYS`), qui porte xuid + nom + filmIndex.
//	G-NEG  Plancher de faux positifs MESURE : la meme lecture de chaine est tentee aux 32
//	       decalages de bit voisins (-16..+16, hors 0). Combien rendent un texte imprimable de
//	       3 caracteres ou plus ? C'est le bruit, compte et non calcule (methode, regle 4).
//	G-IDX  L'ORDRE DES ENREGISTREMENTS. L'ordre des slots doit-il etre le `filmIndex` (5 bits)
//	       que `resolvePlayerIndices` reconstruit aujourd'hui par le fil des morts ? Compare au
//	       `roster[].filmIndex` du document de rejeu.
//	G-EQP  L'EQUIPE. Les six champs courts (2, 10, 14, 6, 8, 7 bits) sont confrontes au
//	       `tracks[].team` du document de rejeu. Critere ecrit d'avance : un champ est retenu
//	       comme candidat s'il coincide avec l'equipe sur TOUS les slots de TOUS les films.
//
// Garde `CHUNK00_FILMS`. Lecture seule, aucun code de production touche.

import (
	"path/filepath"
	"strconv"
	"testing"
	"unicode/utf16"
)

// s3sEnrs decode tous les enregistrements de la grappe terminale d'un film.
func s3sEnrs(d []byte) []*s3sEnr {
	fin := (dernierNonNul(d) + 1) * 8
	g := s3rGrappe(s3rBalayage(d, fin))
	out := make([]*s3sEnr, 0, len(g))
	for _, h := range g {
		if e := s3sDecode(d, h.bit-s3rEnteteBits, fin); e != nil {
			out = append(out, e)
		}
	}
	return out
}

// s3sChaine rend la table des slots LUE PAR LA GRAMMAIRE : on part du premier enregistrement
// que le balayage rend et dont le nom est imprimable, puis on AVANCE de la longueur PREDITE.
//
// C'est le lecteur canonique, et il est strictement meilleur que le balayage de la phase 1 : le
// balayage rend 1 a 2 positions parasites par film (un motif d'en-tete fortuit a l'interieur
// d'un vrai enregistrement), que le pas predit enjambe sans les voir. Borne : 32, celle que
// l'ecrivain impose (0x28A00 / 0x1450).
func s3sChaine(d []byte) []*s3sEnr {
	fin := (dernierNonNul(d) + 1) * 8
	brut := s3sEnrs(d)
	deb := -1
	for i, e := range brut {
		if s3sImprimable(e.gamertag) {
			deb = e.debut
			_ = i
			break
		}
	}
	if deb < 0 {
		return nil
	}
	var out []*s3sEnr
	for p := deb; len(out) < 32; {
		e := s3sDecode(d, p, fin)
		if e == nil || !s3sImprimable(e.gamertag) {
			break
		}
		out = append(out, e)
		p += s3sPredite(e)
	}
	return out
}

// TestSection3SlotGrammaire execute G-CLO : la longueur predite par la grammaire doit valoir
// EXACTEMENT l'ecart mesure jusqu'a l'enregistrement suivant.
func TestSection3SlotGrammaire(t *testing.T) {
	bons, vus, parasites := 0, 0, 0
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		es := s3sEnrs(d)
		t.Logf("=== %s === build %q ; %d enregistrement(s) decode(s)",
			filepath.Base(dir), s3wChaine(d, s3wBuildOff, 32), len(es))
		for i, e := range es {
			pred := s3sPredite(e)
			// Le balayage rend, sur certains films, une position PARASITE : un motif d'en-tete
			// fortuit a l'interieur d'un vrai enregistrement. Elle se reconnait sans rien
			// supposer de la grammaire — son champ de nom ne rend pas de texte imprimable —
			// et elle fausse les DEUX ecarts qui l'encadrent. On la compte a part plutot que
			// de la faire passer pour un echec de fermeture ; le lecteur canonique
			// (`s3sChaine`) l'enjambe par construction.
			if !s3sImprimable(e.gamertag) {
				parasites++
				t.Logf("  pos %2d bit %9d PARASITE (champ de nom non imprimable %q)",
					i, e.debut, e.gamertag)
				continue
			}
			mes := 0
			if i+1 < len(es) && s3sImprimable(es[i+1].gamertag) {
				mes = es[i+1].debut - e.debut
				vus++
				if mes == pred {
					bons++
				}
			}
			t.Logf("  slot %2d bit %9d xuid %19d gt %-16q masque %4d/%4d N %4d M %3d "+
				"predite %6d mesuree %6d %s", i, e.debut, e.xuid, e.gamertag,
				e.popMasque, e.compteMasque, e.n, e.m, pred, mes, s3sVerdict(pred, mes))
		}
	}
	t.Logf("=== BILAN G-CLO === %d/%d ecarts mesurables egaux a la longueur PREDITE par la "+
		"grammaire (aucun ajustement) ; %d position(s) parasite(s) du balayage ecartee(s)",
		bons, vus, parasites)
	if vus > 0 && bons != vus {
		t.Logf("G-CLO INCOMPLET : la grammaire ne ferme pas sur %d enregistrement(s)", vus-bons)
	}
}

func s3sVerdict(pred, mes int) string {
	switch {
	case mes == 0:
		return "(dernier : pas d'ecart mesurable)"
	case mes == pred:
		return "FERME"
	default:
		return "ECART " + strconv.Itoa(mes-pred)
	}
}

// TestSection3SlotGamertag execute G-REF et G-NEG : le nom decode est-il celui du XUID du meme
// enregistrement, et combien de decalages de bit voisins rendent un texte imprimable ?
func TestSection3SlotGamertag(t *testing.T) {
	justes, vus, bruit, essais := 0, 0, 0, 0
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		ref := s3sOracle(dir)
		es := s3sEnrs(d)
		t.Logf("=== %s === %d enregistrement(s) ; oracle externe : %d joueur(s)",
			filepath.Base(dir), len(es), len(ref))
		for i, e := range es {
			r, ok := ref[e.xuid]
			marque := "pas dans l'oracle"
			if ok {
				vus++
				if r.nom == e.gamertag {
					justes++
					marque = "JUSTE"
				} else {
					marque = "FAUX (oracle " + r.nom + ")"
				}
			}
			n, tot := s3sPlancherNom(d, e)
			bruit += n
			essais += tot
			t.Logf("  slot %2d xuid %19d nom lu %-16q %s ; G-NEG %d/%d decalage(s) voisin(s) "+
				"rendent un texte imprimable", i, e.xuid, e.gamertag, marque, n, tot)
		}
	}
	t.Logf("=== BILAN G-REF === %d/%d noms decodes egaux au gamertag de l'oracle externe", justes, vus)
	t.Logf("=== BILAN G-NEG === %d touches sur %d decalages de bit voisins essayes", bruit, essais)
}

// s3sPlancherNom relit la chaine aux 32 decalages voisins et compte ceux qui rendent un texte
// imprimable de trois caracteres ou plus. Plancher de bruit MESURE.
func s3sPlancherNom(d []byte, e *s3sEnr) (touches, essais int) {
	base := e.debut + s3rEnteteBits + 64 + s3sPrefixeMasque + e.compteMasque +
		s3sLargeurN + e.n*8 + s3sLargeurM + e.m*32 + s3sBloc104
	for dec := -16; dec <= 16; dec++ {
		if dec == 0 {
			continue
		}
		essais++
		var u []uint16
		p := base + dec
		for range s3sGtMax {
			v := uint16(s3rBit(d, p, 16))
			p += 16
			if v == 0 {
				break
			}
			u = append(u, v)
		}
		if s3sImprimable(string(utf16.Decode(u))) {
			touches++
		}
	}
	return touches, essais
}

// s3sTexte16 relit un bloc BRUT comme une suite d'unites UTF-16 terminee par NUL. Le parametre
// `pf` choisit l'ordre des octets DANS LE BLOC : petit-boutiste (l'image memoire d'une chaine
// UTF-16LE recopiee telle quelle par `FUN_1406d60f4`) ou gros-boutiste (ce que produit
// `FUN_1407ece18`, qui ecrit chaque unite comme un scalaire de 16 bits MSB d'abord).
//
// Cette distinction n'est pas un detail de commodite : c'est un CONTROLE. Les deux champs de
// texte de l'enregistrement passent par des ecrivains differents, donc un seul des deux ordres
// doit rendre un texte lisible sur chaque champ. Si les deux rendaient du texte, la lecture
// serait fortuite.
func s3sTexte16(b []byte, pf bool) string {
	var u []uint16
	for k := 0; k+1 < len(b); k += 2 {
		v := uint16(b[k])<<8 | uint16(b[k+1])
		if pf {
			v = uint16(b[k+1])<<8 | uint16(b[k])
		}
		if v == 0 {
			break
		}
		u = append(u, v)
	}
	return string(utf16.Decode(u))
}
