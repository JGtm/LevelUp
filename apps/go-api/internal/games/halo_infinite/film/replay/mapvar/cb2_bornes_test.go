package mapvar

// cb2_bornes_test.go — LE LECTEUR BOND EST BORNE (lot J2.8, constat RB1-4, 2026-09-26).
//
// Un `.mvar` vient d un telechargement : un compte de conteneur, de tableau ou de chaine lu dans
// le flux ne doit JAMAIS dimensionner une allocation au-dela de ce que les octets restants
// peuvent porter. Le defaut : `make([]Value, 0, count)` sur un varint brut — un varint de 2^63
// devient un compte negatif (panique), un compte de 2^40 demande des dizaines de teraoctets.

import (
	"encoding/binary"
	"errors"
	"testing"
)

// compteImpossible : un compte qu aucun .mvar ne peut porter (2^40 elements).
const compteImpossible = uint64(1) << 40

// docAvecChamp3 ecrit une racine Bond dont le champ 3 est de type `typ` et commence par
// `entete` : la longueur de la racine couvre exactement ces octets.
func docAvecChamp3(typ byte, entete []byte) []byte {
	corps := append([]byte{3<<5 | typ}, entete...)
	return append(binary.AppendUvarint(nil, uint64(len(corps))), corps...)
}

// documentsAuCompte rend, pour un compte `c`, un document par forme de compte du format.
func documentsAuCompte(c uint64) map[string][]byte {
	v := binary.AppendUvarint(nil, c)
	return map[string][]byte{
		"liste":        docAvecChamp3(btList, append([]byte{btStruct}, v...)),
		"ensemble":     docAvecChamp3(btSet, append([]byte{btUint32}, v...)),
		"map":          docAvecChamp3(btMap, append([]byte{btUint32, btStruct}, v...)),
		"chaine":       docAvecChamp3(btString, v),
		"chaine large": docAvecChamp3(btWString, v),
	}
}

// parseSansPanique rend l erreur de Parse, ou la panique qu il a levee.
func parseSansPanique(doc []byte) (panique any, err error) {
	defer func() { panique = recover() }()
	_, err = Parse(doc)
	return nil, err
}

// TestParse_CompteDeConteneurImpossibleRendUneErreur : un compte impossible rend une erreur
// typee, jamais une panique.
//
// L ORDRE DES VALEURS EST VOULU : sur un lecteur non borne, 2^63 (negatif en `int`) et 2^62
// levent une panique RATTRAPABLE que ce test nomme ; 2^40, lui, demanderait au runtime des
// dizaines de teraoctets avant toute panique.
func TestParse_CompteDeConteneurImpossibleRendUneErreur(t *testing.T) {
	for _, c := range []uint64{1 << 63, 1 << 62, compteImpossible} {
		for forme, doc := range documentsAuCompte(c) {
			p, err := parseSansPanique(doc)
			if p != nil {
				t.Fatalf("%s au compte %d : panique %v", forme, c, p)
			}
			if !errors.Is(err, ErrCompteHorsBornes) {
				t.Fatalf("%s au compte %d : erreur %v, attendu ErrCompteHorsBornes", forme, c, err)
			}
		}
	}
}

// TestParse_CompteQuiTientEstLu : la borne ne refuse pas un compte que les octets portent.
func TestParse_CompteQuiTientEstLu(t *testing.T) {
	// Une liste de huit uint32 (compte en varint, forme longue) puis la fin de la racine.
	liste := []byte{btUint32, 8, 1, 2, 3, 4, 5, 6, 7, 8, btStop}
	v, err := DecodeRoot(docAvecChamp3(btList, liste))
	if err != nil {
		t.Fatalf("liste de huit elements : %v", err)
	}
	if f, ok := v.Field(3); !ok || len(f.Items) != 8 {
		t.Fatalf("liste de huit elements : %+v", f)
	}
}
