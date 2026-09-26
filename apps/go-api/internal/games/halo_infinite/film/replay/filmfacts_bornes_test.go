package replay

// filmfacts_bornes_test.go — LE DECODEUR DU FICHIER DE FAITS EST BORNE (lot J2.7, constat RA1-5,
// 2026-09-26).
//
// UN FICHIER DE FAITS VIENT DU DISQUE : perime, tronque ou corrompu, il doit rendre une ERREUR,
// jamais faire tomber le processus. Le defaut mesure : un compte d elements lu dans le flux
// dimensionnait `make([]T, 0, n)` sans borne : un varint de 2^63 devient un compte NEGATIF
// (panique « makeslice: cap out of range »), un compte de 2^40 demande une allocation de dizaines
// de teraoctets.
//
// AUCUN OCTET DE FILM : les temoins sont fabriques par l encodeur de production, a partir de
// l entree de catalogue du film de reference.

import (
	"encoding/binary"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// compteImpossible : un compte qu aucun fichier de faits ne peut porter (2^40 elements).
const compteImpossible = uint64(1) << 40

// faitsVidesDeLaCarte rend des faits SANS AUCUN ELEMENT dont la cle de cuisson est celle que
// l entree de catalogue accepte : le blob qui en sort est fait, apres son en-tete, de comptes et
// de compteurs a zero — c est ce qui permet de viser chaque compte sans connaitre le format.
func faitsVidesDeLaCarte(entry profile.MapQuantEntry) *FilmFacts {
	g := &FilmFacts{Film: "temoin", MapModule: entry.Module}
	if impose := grammar.NewFilmContextForMap(nil, &entry, nil).ImposedLayout(); impose != nil {
		g.AxisW = impose.AxisW
	} else {
		g.AxisW, g.LayoutDetected = entry.AxisWidths, true
	}
	return g
}

// debutDesComptes rend l offset du PREMIER compte du blob qui suit la section des positions :
// celui des creations de bipede.
func debutDesComptes(g *FilmFacts) int {
	w := &gwriter{b: []byte(filmFactsMagic)}
	encodeEntete(w, g)
	encodePositionSection(w, nil)
	return len(w.b)
}

// fichierAutourDuBlob ecrit un fichier de faits dont la section des entrees porte `blob` tel
// quel. L en-tete est celui que l encodeur de production ecrit pour `g`.
func fichierAutourDuBlob(t *testing.T, g *FilmFacts, blob []byte) []byte {
	t.Helper()
	valide, err := EncodeFilmFactsFile(&FilmFactsFile{Facts: *g,
		EmpreinteDeCle: EmpreinteDeCle(goldenEntryPourTest(t))})
	if err != nil {
		t.Fatalf("encodage du fichier temoin : %v", err)
	}
	entete, err := DecodeFilmFactsEntete(valide)
	if err != nil {
		t.Fatalf("en-tete du fichier temoin : %v", err)
	}
	w := &gwriter{b: append([]byte(nil), valide[:entete.corps]...)}
	entrees := &gwriter{}
	entrees.u(uint64(len(blob)))
	entrees.b = append(entrees.b, blob...)
	encodeGardesDeMode(entrees, g.FilmInputs)
	encodeEntitesDesJoueurs(entrees, g.PlayerEntities)
	encodeVerdictDuFilDesMorts(entrees, g.DeathsFeed)
	ecrireSection(w, sectionEntrees, entrees.b)
	return w.b
}

// remplacerOctet rend une copie de `b` ou l octet `i` est remplace par le varint de `v`.
func remplacerOctet(b []byte, i int, v uint64) []byte {
	out := append([]byte(nil), b[:i]...)
	out = binary.AppendUvarint(out, v)
	return append(out, b[i+1:]...)
}

// decoderSansPanique rend l erreur du decodeur, ou la panique qu il a levee.
func decoderSansPanique(fichier []byte, entry profile.MapQuantEntry) (panique any, err error) {
	defer func() { panique = recover() }()
	_, err = DecodeFilmFactsFile(fichier, entry)
	return nil, err
}

// TestDecodeFilmFactsFile_CompteImpossibleRendUneErreur : un compte impossible rend une erreur,
// jamais une panique.
//
// DEUX MESURES. (1) LE BALAYAGE : chaque octet du fichier temoin (en-tete, cadre, blob) est
// remplace tour a tour par un varint enorme — negatif une fois converti en `int` (2^63), au-dela
// de toute allocation (2^62), puis `compteImpossible` ; aucune panique n est admise. Le blob
// etant fait de zeros, chaque compte du format est vise sans que le test connaisse l ordre des
// sections. (2) LE COMPTE NOMME : 2^40 creations de bipede rendent une ERREUR.
//
// L ORDRE DES VALEURS EST VOULU : sur un decodeur non borne, 2^63 et 2^62 levent une panique
// RATTRAPABLE (« makeslice: cap out of range ») que ce test nomme ; 2^40, lui, demanderait au
// runtime des dizaines de teraoctets avant toute panique.
func TestDecodeFilmFactsFile_CompteImpossibleRendUneErreur(t *testing.T) {
	entry := goldenEntryPourTest(t)
	g := faitsVidesDeLaCarte(entry)
	blob := EncodeFilmFacts(g)
	fichier := fichierAutourDuBlob(t, g, blob)
	if _, err := DecodeFilmFactsFile(fichier, entry); err != nil {
		t.Fatalf("le fichier temoin doit se relire : %v", err)
	}
	for _, v := range []uint64{1 << 63, 1 << 62, compteImpossible} {
		for i := range fichier {
			if p, _ := decoderSansPanique(remplacerOctet(fichier, i, v), entry); p != nil {
				t.Fatalf("valeur %d a l octet %d du fichier : panique %v", v, i, p)
			}
		}
	}
	site := debutDesComptes(g)
	if blob[site] != 0 {
		t.Fatalf("octet %d du blob : %#x, attendu le compte nul des creations de bipede",
			site, blob[site])
	}
	p, err := decoderSansPanique(fichierAutourDuBlob(t, g, remplacerOctet(blob, site, compteImpossible)), entry)
	if p != nil {
		t.Fatalf("compte de %d creations de bipede : panique %v", compteImpossible, p)
	}
	if err == nil {
		t.Fatalf("compte de %d creations de bipede : aucune erreur", compteImpossible)
	}
	t.Logf("compte impossible refuse : %v", err)
}
