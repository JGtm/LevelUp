package filmdec

// section3_ecrivain_research_test.go — LA CARTE DE chunk_00 RELUE CHEZ L'ECRIVAIN.
//
// ## D'OU VIENT CETTE GRAMMAIRE
//
// Elle n'est pas devinee sur les octets : elle est LUE dans `HaloInfinite.exe`. Deux fonctions
// serialisent la totalite de `chunk_00`, et leur desassemblage donne, champ par champ, la source
// et la LARGEUR EN BITS (le quatrieme argument `R9D` de l'ecrivain de bits `FUN_1406d60f4`, qui
// avance le curseur `*(writer+0x2c) += N`) :
//
//	FUN_14299b198(base, writer)   — la tete, dix champs :
//	   base+0x000000  0x20 bits      u32 A            -> film 0x000000
//	   base+0x000004  0x20 bits      u32 B            -> film 0x000004
//	   base+0x000008  0x659000 bits  = 0xCB200 octets -> film 0x000008  LE REGISTRE (832 000 o)
//	   base+0x0CB208  0xF60 bits     = 0x1EC octets   -> film 0x0CB208  LA TABLE PAR TYPE (123 u32)
//	   base+0x0CB3F4  0x100 bits     = 32 octets      -> film 0x0CB3F4  version
//	   base+0x0CB414  0x100 bits     = 32 octets      -> film 0x0CB414  build
//	   base+0x0CB434  0x100 bits     = 32 octets      -> film 0x0CB434  saveur
//	   base+0x0CB454  0x20 bits                       -> film 0x0CB454  u32
//	   base+0x0CB458  0x20 bits                       -> film 0x0CB458  u32
//	   base+0x0CB45C  FUN_1406d49c4 = UN SEUL BIT     -> film 0x0CB45C  booleen
//
//	FUN_14299b278(base, writer) — la suite, onze champs puis le corps :
//	   base+0x0CB460  0x800 bits = 256 o    champ de nom (128 caracteres UTF-16)
//	   base+0x0CB560  0x800 bits = 256 o    second champ de nom
//	   base+0x0CB660  0x20 bits             <- `_time64()` ecrit la : L'HORODATAGE DU MATCH
//	   base+0x0CB664  0x20 · 0x0CB668 0x20 · 0x0CB66C 0x20
//	   base+0x0CB670  0x8000 bits = 4096 o  (x3 : 0x0CB670, 0x0CC670, 0x0CD670)
//	   base+0x0CE670  0x80 bits = 16 o      (x2 : 0x0CE670, 0x0CE680)
//	   base+0x0CE690  FUN_1407ec560         LE CORPS (section 3 dense)
//
// ## LES DEUX CONSEQUENCES QUI CHANGENT LA CARTE DU 2026-08-30
//
//  1. **Le registre commence a l'octet 8, pas a 0.** Les deux u32 de tete sont des champs a part
//     entiere. `parseRegistry` lit depuis 0 : son « kind » du slot i est donc le champ situe 8
//     octets AVANT le nom du slot i, ce qui explique d'un coup les deux observations du depot —
//     « kind = 0 sur 1 066 des 1 067 slots » et « le niveau lu un cran plus loin » de
//     `registryBlockTail`. Decouverte hors perimetre, NON TRAITEE.
//  2. **Apres le booleen d'un bit, tout le reste du fichier est decale d'UN BIT.** C'est pour
//     cela qu'aucune lecture alignee sur l'octet ne rendait jamais rien dans la section 3 : les
//     champs y sont a `offset*8 + 1`, pas a `offset*8`.
//
// ## LES CONTROLES, ECRITS AVANT LA MESURE
//
//	W-POS  La fermeture arithmetique. 0x0CB208 + 123 x 4 = 0x0CB3F4 = l'offset du champ version.
//	       Trois mesures independantes (debut de table, cardinal, offset de version) doivent se
//	       fermer SANS ajustement, sinon la lecture de l'ecrivain est fausse.
//	W-NEG  Le decalage d'un bit. L'horodatage est cherche aux dix-sept decalages 0..16 bits.
//	       Critere ecrit d'avance : UN SEUL decalage doit rendre une valeur plausible comme
//	       temps Unix ; si deux ou zero le font, la mesure ne conclut pas.
//	W-REF  L'horodatage lu doit tomber a moins de 120 secondes du debut du match. La reference
//	       arrive par `CHUNK00_DEBUTS` (film=epoch), remplie depuis `match_registry`. Absente,
//	       l'instrument se contente d'imprimer la date lue.
//
// Garde `CHUNK00_FILMS`. Lecture seule, aucun code de production touche.

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Offsets de la tete, lus dans FUN_14299b198. Ils sont en OCTETS du tampon inflate.
const (
	s3wRegistreDebut = 0x000008 // le registre commence ici, pas a 0
	s3wRegistreBits  = 0x659000 // 6 656 000 bits = 832 000 octets = 50 blocs de 0x4100
	s3wTableDebut    = 0x0CB208 // la table par type
	s3wTableBits     = 0xF60    // 3 936 bits = 492 octets = 123 u32
	s3wVersionOff    = 0x0CB3F4 // trois champs de 32 octets
	s3wBuildOff      = 0x0CB414
	s3wSaveurOff     = 0x0CB434
	s3wU32aOff       = 0x0CB454
	s3wU32bOff       = 0x0CB458
	s3wBoolOff       = 0x0CB45C // UN bit : tout ce qui suit est decale
	s3wNomBits       = 0x800    // 256 octets par champ de nom, deux champs
	s3wCorpsOff      = 0x0CE68C // debut du corps dans le FLUX (FUN_1407ec560) : struct 0x0CE690 - 4
)

// s3wBitApresBool rend la position en BITS du premier champ qui suit le booleen d'un bit.
func s3wBitApresBool(decalage int) int { return s3wBoolOff*8 + decalage }

// TestSection3CarteEcrivain confronte la grammaire de l'ecrivain aux octets des films.
func TestSection3CarteEcrivain(t *testing.T) {
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		t.Logf("=== %s === %d octets inflates (0x%x)", filepath.Base(dir), len(d), len(d))
		s3wTete(t, d)
		s3wFermeture(t)
		s3wChampsNom(t, d)
		s3wBlocs(t, d)
	}
}

// s3wTete imprime les champs de tete aux offsets que donne l'ecrivain.
func s3wTete(t *testing.T, d []byte) {
	t.Helper()
	t.Logf("  u32 A @0x000000 = %d (0x%08x) ; u32 B @0x000004 = %d",
		binary.LittleEndian.Uint32(d), binary.LittleEndian.Uint32(d), binary.LittleEndian.Uint32(d[4:]))
	t.Logf("  premier nom du registre @0x%06x = %q (l'ecrivain place le registre a l'octet 8)",
		s3wRegistreDebut, s3wChaine(d, s3wRegistreDebut, 64))
	t.Logf("  registre : 0x%06x bits = %d octets, donc 0x%06x..0x%06x",
		s3wRegistreBits, s3wRegistreBits/8, s3wRegistreDebut, s3wRegistreDebut+s3wRegistreBits/8)
	t.Logf("  table par type @0x%06x : %d u32 ; trois premieres valeurs %d %d %d",
		s3wTableDebut, s3wTableBits/32,
		binary.LittleEndian.Uint32(d[s3wTableDebut:]),
		binary.LittleEndian.Uint32(d[s3wTableDebut+4:]),
		binary.LittleEndian.Uint32(d[s3wTableDebut+8:]))
	t.Logf("  version %q · build %q · saveur %q",
		s3wChaine(d, s3wVersionOff, 32), s3wChaine(d, s3wBuildOff, 32), s3wChaine(d, s3wSaveurOff, 32))
	t.Logf("  u32 @0x%06x = 0x%08x (identifiant de build) ; u32 @0x%06x = 0x%08x (changelist)",
		s3wU32aOff, binary.LittleEndian.Uint32(d[s3wU32aOff:]),
		s3wU32bOff, binary.LittleEndian.Uint32(d[s3wU32bOff:]))
	t.Logf("  booleen d'un bit @0x%06x : bit de poids fort = %d", s3wBoolOff, d[s3wBoolOff]>>7)
}

// s3wFermeture execute le controle W-POS : les trois mesures doivent se fermer sans ajustement.
func s3wFermeture(t *testing.T) {
	t.Helper()
	fin := s3wRegistreDebut + s3wRegistreBits/8
	if fin != s3wTableDebut {
		t.Errorf("W-POS CASSE : registre 0x%06x + %d o = 0x%06x, or la table est a 0x%06x",
			s3wRegistreDebut, s3wRegistreBits/8, fin, s3wTableDebut)
	}
	apres := s3wTableDebut + s3wTableBits/8
	if apres != s3wVersionOff {
		t.Errorf("W-POS CASSE : table 0x%06x + %d u32 = 0x%06x, or version est a 0x%06x",
			s3wTableDebut, s3wTableBits/32, apres, s3wVersionOff)
	}
	t.Logf("  W-POS : 0x%06x + %d o (registre) = 0x%06x = debut de table ; "+
		"0x%06x + %d x 4 o = 0x%06x = offset version. Fermeture SANS ajustement.",
		s3wRegistreDebut, s3wRegistreBits/8, fin, s3wTableDebut, s3wTableBits/32, apres)
}

// s3wChampsNom imprime l'occupation des deux champs de nom de 256 octets et localise
// l'horodatage du match par le balayage de decalage (controle W-NEG).
func s3wChampsNom(t *testing.T, d []byte) {
	t.Helper()
	for k := 0; k < 2; k++ {
		deb := s3wBoolOff + k*s3wNomBits/8
		nz := 0
		for i := deb; i < deb+s3wNomBits/8 && i < len(d); i++ {
			if d[i] != 0 {
				nz++
			}
		}
		t.Logf("  champ de nom %d (0x%06x, 256 o, decale d'un bit) : %d octets non nuls",
			k, deb, nz)
	}
	base := s3wBitApresBool(0) + 2*s3wNomBits
	var bons []int
	for dec := 0; dec <= 16; dec++ {
		v := s3wU32Flux(d, base+dec)
		marque := ""
		if s3wPlausible(v) {
			marque = " <= PLAUSIBLE comme temps Unix : " + s3wUTC(v)
			bons = append(bons, dec)
		}
		t.Logf("    decalage %2d bits : u32 = %d%s", dec, v, marque)
	}
	t.Logf("  W-NEG : %d decalage(s) sur 17 rendent une valeur plausible -> %v", len(bons), bons)
	if len(bons) != 1 {
		t.Logf("  W-NEG NON CONCLUANT sur ce film (le critere exigeait exactement un decalage)")
		return
	}
	if bons[0] != 1 {
		t.Errorf("le decalage plausible est %d, l'ecrivain en predit 1 (booleen d'un bit)", bons[0])
	}
	for k := 1; k < 4; k++ {
		t.Logf("    u32 suivant %d (decalage 1) = %d", k, s3wU32Flux(d, base+1+k*32))
	}
}

// s3wBlocs imprime l'occupation des trois blocs de 4 096 octets et des deux blocs de 16,
// puis les premiers octets du corps.
func s3wBlocs(t *testing.T, d []byte) {
	t.Helper()
	deb := s3wBoolOff + 2*s3wNomBits/8 + 16 // apres les deux noms et les quatre u32
	for k := 0; k < 3; k++ {
		o := deb + k*0x1000
		nz := 0
		for i := o; i < o+0x1000 && i < len(d); i++ {
			if d[i] != 0 {
				nz++
			}
		}
		t.Logf("  bloc de 0x1000 octets @0x%06x : %d octets non nuls", o, nz)
	}
	t.Logf("  deux blocs de 16 octets @0x%06x : % x | % x",
		deb+3*0x1000, d[deb+3*0x1000:deb+3*0x1000+16], d[deb+3*0x1000+16:deb+3*0x1000+32])
	t.Logf("  corps @0x%06x (predit par l'ecrivain, decale d'un bit) : % x",
		s3wCorpsOff, d[s3wCorpsOff:s3wCorpsOff+16])
	t.Logf("  dernier octet non nul du tampon : 0x%06x", dernierNonNul(d))
}

// TestSection3HorodatageContreRegistre execute le controle W-REF : l'horodatage lu dans le film
// doit tomber a moins de 120 s du debut du match. La reference arrive par CHUNK00_DEBUTS, au
// format `film=epoch;film=epoch` (rempli depuis match_registry.start_time_utc).
func TestSection3HorodatageContreRegistre(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	refs := s3wRefs(os.Getenv("CHUNK00_DEBUTS"))
	if len(refs) == 0 {
		t.Skip("CHUNK00_DEBUTS absent : pas de reference de debut de match, controle W-REF saute")
	}
	base := s3wBitApresBool(1) + 2*s3wNomBits
	for _, dir := range dirs {
		id := filepath.Base(dir)
		ref, ok := refs[id]
		if !ok {
			t.Logf("=== %s === pas de reference dans CHUNK00_DEBUTS", id)
			continue
		}
		_, d := readChunk00(t, dir)
		v := int64(s3wU32Flux(d, base))
		ecart := v - ref
		t.Logf("=== %s === film %s (%d) ; registre %s (%d) ; ecart %+d s",
			id, s3wUTC(uint32(v)), v, s3wUTC(uint32(ref)), ref, ecart)
		if ecart < -120 || ecart > 120 {
			t.Errorf("W-REF CASSE sur %s : ecart de %d s (seuil 120 s ecrit avant la mesure)", id, ecart)
		}
	}
}

// s3wRefs decoupe `film=epoch;film=epoch`.
func s3wRefs(v string) map[string]int64 {
	out := map[string]int64{}
	for _, p := range strings.Split(v, ";") {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) != 2 {
			continue
		}
		if n, err := strconv.ParseInt(strings.TrimSpace(kv[1]), 10, 64); err == nil {
			out[strings.TrimSpace(kv[0])] = n
		}
	}
	return out
}

// s3wU32Flux lit 32 bits MSB-first a `bit` et les relit comme un u32 petit-boutiste : l'ecrivain
// pousse les octets de la source DANS L'ORDRE DES ADRESSES, MSB d'abord, donc les quatre octets
// sortis du flux sont ceux de la memoire et se relisent en LE.
func s3wU32Flux(d []byte, bit int) uint32 {
	var acc uint32
	for k := 0; k < 32; k++ {
		b := bit + k
		if b>>3 >= len(d) {
			return 0
		}
		acc = acc<<1 | uint32((d[b>>3]>>(7-uint(b&7)))&1)
	}
	var tmp [4]byte
	binary.BigEndian.PutUint32(tmp[:], acc)
	return binary.LittleEndian.Uint32(tmp[:])
}

// s3wPlausible : fenetre 2020-09..2030-01, bornes ecrites avant la mesure.
func s3wPlausible(v uint32) bool { return v > 1600000000 && v < 1900000000 }

// s3wUTC formate un temps Unix.
func s3wUTC(v uint32) string {
	return time.Unix(int64(v), 0).UTC().Format("2006-01-02 15:04:05Z")
}

// s3wChaine lit une chaine ASCII terminee par NUL dans un champ de largeur fixe.
func s3wChaine(d []byte, off, max int) string {
	if off < 0 || off >= len(d) {
		return ""
	}
	end := off + max
	if end > len(d) {
		end = len(d)
	}
	s := string(d[off:end])
	if i := strings.IndexByte(s, 0); i >= 0 {
		s = s[:i]
	}
	return s
}
