package main

// walkers_test.go — LES QUATRE AXES DE DIVERGENCE, SUR DES CHUNKS CONSTRUITS.
//
// Les valeurs sont CALCULABLES a la main (un en-tete de 16 octets, un payload connu) : une
// mesure faite avec des marcheurs qu'on n'a pas verifies ne prouverait rien.

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"slices"
	"testing"
)

// paquetBrut construit un paquet : en-tete de 16 octets puis payload.
func paquetBrut(typ int, ts uint64, payload []byte) []byte {
	tete := make([]byte, entetePaquet)
	binary.LittleEndian.PutUint16(tete[0:], uint16(typ))
	binary.LittleEndian.PutUint32(tete[4:], uint32(len(payload)))
	binary.LittleEndian.PutUint64(tete[8:], ts)
	return append(tete, payload...)
}

func TestLesQuatreGrammairesSAccordentSurUnChunkSain(t *testing.T) {
	chunk := slices.Concat(
		paquetBrut(0, 1000, []byte("aaaa")),
		paquetBrut(2, 2000, []byte("bbbbbbbb")),
	)
	uni := marcheUnifiee(chunk)
	if len(uni) != 2 {
		t.Fatalf("grammaire unifiee : 2 paquets attendus, %d obtenus", len(uni))
	}
	if !slices.Equal(marcheFilmdec(chunk), uni) {
		t.Error("filmdec diverge sur un chunk sain")
	}
	if !slices.Equal(marcheKillsource(chunk), uni) {
		t.Error("killsource diverge sur un chunk sain")
	}
	if !slices.Equal(marcheObjectiveevents(chunk), typeZero(uni)) {
		t.Error("objectiveevents diverge sur un chunk sain")
	}
}

func TestAxeTailleNulle(t *testing.T) {
	// EN-TETE DEGENERE (taille nulle, type autre que 7 — le cas `chunk_00`) : filmdec avale le
	// paquet vide et continue ; killsource et l'unifiee s'arretent net, SANS l'emettre.
	chunk := slices.Concat(
		paquetBrut(0, 1000, []byte("aaaa")),
		paquetBrut(0, 2000, nil),
		paquetBrut(0, 3000, []byte("cccc")),
	)
	uni := marcheUnifiee(chunk)
	if len(uni) != 1 {
		t.Fatalf("unifiee : 1 paquet attendu avant l'en-tete degenere, %d obtenus", len(uni))
	}
	if uni[0].ts != 1000 {
		t.Errorf("unifiee : le paquet emis doit etre le premier (ts 1000), obtenu ts %d", uni[0].ts)
	}
	if n := len(marcheFilmdec(chunk)); n != 3 {
		t.Errorf("filmdec : 3 paquets attendus (il accepte size==0), %d obtenus", n)
	}
	if !slices.Equal(marcheKillsource(chunk), uni) {
		t.Error("killsource devrait s'arreter comme l'unifiee sur size==0")
	}
	if got := axeAuPointDArret(chunk, uni); got != "taille_nulle" {
		t.Errorf("axe attendu taille_nulle, obtenu %q", got)
	}
}

func TestAxeChunkEnd(t *testing.T) {
	// CHUNK_END porte ici un payload : filmdec et killsource le traversent ; l'unifiee L'EMET
	// puis s'arrete (D3 REVISEE) ; objectiveevents avance dessus puis s'arrete sans l'emettre.
	// Le paquet PLACE APRES le terminateur ne doit sortir chez PERSONNE qui s'arrete la.
	chunk := slices.Concat(
		paquetBrut(0, 1000, []byte("aaaa")),
		paquetBrut(typeChunkEnd, 2000, []byte("eeee")),
		paquetBrut(0, 3000, []byte("cccc")),
	)
	uni := marcheUnifiee(chunk)
	if len(uni) != 2 {
		t.Fatalf("unifiee : 2 paquets attendus (le terminateur est EMIS), %d obtenus", len(uni))
	}
	if uni[1].typ != typeChunkEnd || uni[1].ts != 2000 {
		t.Errorf("unifiee : le dernier paquet doit etre le terminateur (type %d, ts 2000), "+
			"obtenu type %d ts %d", typeChunkEnd, uni[1].typ, uni[1].ts)
	}
	for _, p := range uni {
		if p.ts == 3000 {
			t.Error("unifiee : le paquet place APRES le terminateur ne doit pas etre emis")
		}
	}
	if n := len(marcheFilmdec(chunk)); n != 3 {
		t.Errorf("filmdec : 3 paquets attendus (il ignore CHUNK_END), %d obtenus", n)
	}
	if n := len(marcheKillsource(chunk)); n != 3 {
		t.Errorf("killsource : 3 paquets attendus (il ignore CHUNK_END), %d obtenus", n)
	}
	if n := len(marcheObjectiveevents(chunk)); n != 1 {
		t.Errorf("objectiveevents : 1 paquet attendu (arret sur CHUNK_END), %d obtenus", n)
	}
	if got := axeAuPointDArret(chunk, uni); got != "chunk_end" {
		t.Errorf("axe attendu chunk_end, obtenu %q", got)
	}
}

// TestTerminateurTailleNulleEstEmis — LE CAS REEL DU CACHE (D3 REVISEE) : sur un chunk de
// DONNEES, l'unique paquet de taille 0 est le terminateur, en derniere position, sans un octet
// apres. C'est le cas que l'ancienne regle « arret sur size <= 0 » jetait.
func TestTerminateurTailleNulleEstEmis(t *testing.T) {
	chunk := slices.Concat(
		paquetBrut(0, 1000, []byte("aaaa")),
		paquetBrut(typeChunkEnd, 2000, nil),
	)
	uni := marcheUnifiee(chunk)
	if len(uni) != 2 {
		t.Fatalf("unifiee : 2 paquets attendus (terminateur a taille nulle COMPRIS), %d obtenus", len(uni))
	}
	if uni[1].typ != typeChunkEnd || uni[1].size != 0 {
		t.Errorf("unifiee : dernier paquet attendu type %d taille 0, obtenu type %d taille %d",
			typeChunkEnd, uni[1].typ, uni[1].size)
	}
	// EFFET ATTENDU DE LA REVISION, film par film : filmdec voit EXACTEMENT la meme chose (il
	// accepte size==0 et bute ensuite sur la fin du tampon)...
	if !slices.Equal(marcheFilmdec(chunk), uni) {
		t.Error("filmdec doit voir le meme jeu de paquets que l'unifiee sur un chunk de donnees")
	}
	// ...tandis que killsource, qui s'arrete sur `size <= 0`, voit un paquet de MOINS — celui-la
	// meme, qu'il filtre de toute facon par son type en aval.
	if n := len(marcheKillsource(chunk)); n != 1 {
		t.Errorf("killsource : 1 paquet attendu (il s'arrete sur le terminateur), %d obtenus", n)
	}
	// Et objectiveevents ne rend que le type 0 : le terminateur n'est pas dans sa sortie.
	if !slices.Equal(marcheObjectiveevents(chunk), typeZero(uni)) {
		t.Error("objectiveevents doit voir les memes paquets de type 0 que l'unifiee")
	}
	if got := axeAuPointDArret(chunk, uni); got != "chunk_end" {
		t.Errorf("axe attendu chunk_end, obtenu %q", got)
	}
}

func TestAxeBorneHaute(t *testing.T) {
	// Un second en-tete annonce un payload qui deborde du tampon SANS depasser sa longueur
	// totale : c'est la borne qu'objectiveevents evalue sans l'offset.
	chunk := slices.Concat(paquetBrut(0, 1000, []byte("aaaa")), make([]byte, entetePaquet))
	binary.LittleEndian.PutUint32(chunk[len(chunk)-entetePaquet+4:], 24)
	uni := marcheUnifiee(chunk)
	if len(uni) != 1 {
		t.Fatalf("unifiee : 1 paquet attendu avant le debordement, %d obtenus", len(uni))
	}
	if got := axeAuPointDArret(chunk, uni); got != "borne_haute" {
		t.Errorf("axe attendu borne_haute, obtenu %q", got)
	}
	if !slices.Equal(marcheFilmdec(chunk), uni) || !slices.Equal(marcheKillsource(chunk), uni) {
		t.Error("filmdec et killsource bornent avec l'offset : ils doivent s'arreter la aussi")
	}
	// L'AXE PORTE SUR LA MARCHE, PAS SUR LA SORTIE, et c'est ici qu'on le verrouille :
	// objectiveevents borne par `size > len(data)` SANS l'offset, donc il ne s'arrete PAS devant
	// cet en-tete — il avance de 16+24 et sort du tampon au tour suivant. Sa sortie n'en garde
	// pas moins les memes paquets de type 0 que l'unifiee, parce que son test d'emission, lui,
	// reprend l'offset : le paquet qui deborde n'est emis par personne.
	if !slices.Equal(marcheObjectiveevents(chunk), typeZero(uni)) {
		t.Errorf("objectiveevents doit voir les memes paquets de type 0 que l'unifiee, obtenu %+v",
			marcheObjectiveevents(chunk))
	}
}

// TestTerminateurEnPremierePosition — LE CAS LIMITE DE L'ARRET APRES EMISSION : le terminateur
// OUVRE le chunk. L'unifiee l'emet puis ferme, donc rien de ce qui suit n'est lu — et l'axe se
// lit sur le DERNIER paquet emis, pas sur l'en-tete suivant (c'est la premiere des deux lectures
// d'axeAuPointDArret).
func TestTerminateurEnPremierePosition(t *testing.T) {
	chunk := slices.Concat(
		paquetBrut(typeChunkEnd, 1000, []byte("eeee")),
		paquetBrut(0, 2000, []byte("cccc")),
	)
	uni := marcheUnifiee(chunk)
	if len(uni) != 1 || uni[0].typ != typeChunkEnd || uni[0].ts != 1000 {
		t.Fatalf("unifiee : le seul paquet attendu est le terminateur (type %d, ts 1000), obtenu %+v",
			typeChunkEnd, uni)
	}
	if got := axeAuPointDArret(chunk, uni); got != "chunk_end" {
		t.Errorf("axe attendu chunk_end, obtenu %q", got)
	}
	// objectiveevents avance sur le terminateur puis s'arrete SANS l'emettre : sa sortie est
	// vide, exactement comme le type 0 de l'unifiee.
	if n := len(marcheObjectiveevents(chunk)); n != 0 {
		t.Errorf("objectiveevents : aucun paquet attendu (le terminateur ouvre le chunk), %d obtenus", n)
	}
	if !slices.Equal(marcheObjectiveevents(chunk), typeZero(uni)) {
		t.Error("objectiveevents et l'unifiee doivent voir les memes paquets de type 0")
	}
	// filmdec et killsource ne regardent pas le type : ils traversent le terminateur.
	if n := len(marcheFilmdec(chunk)); n != 2 {
		t.Errorf("filmdec : 2 paquets attendus (il ignore CHUNK_END), %d obtenus", n)
	}
	if n := len(marcheKillsource(chunk)); n != 2 {
		t.Errorf("killsource : 2 paquets attendus (il ignore CHUNK_END), %d obtenus", n)
	}
}

// TestAxeFinDeTampon — UN CHUNK SANS TERMINATEUR : la marche unifiee consomme tout et s'arrete
// faute d'en-tete a lire. L'axe doit nommer la fin du tampon, et non un arret de grammaire —
// confondre les deux ferait chercher une divergence la ou le chunk s'est simplement termine.
func TestAxeFinDeTampon(t *testing.T) {
	chunk := slices.Concat(
		paquetBrut(0, 1000, []byte("aaaa")),
		paquetBrut(2, 2000, []byte("bbbbbbbb")),
	)
	uni := marcheUnifiee(chunk)
	if len(uni) != 2 {
		t.Fatalf("unifiee : 2 paquets attendus, %d obtenus", len(uni))
	}
	if uni[len(uni)-1].typ == typeChunkEnd {
		t.Fatal("ce chunk ne doit PAS porter de terminateur : c'est tout l'objet du cas")
	}
	if got := axeAuPointDArret(chunk, uni); got != "fin_de_tampon" {
		t.Errorf("axe attendu fin_de_tampon, obtenu %q", got)
	}
	// Un tampon trop court pour un seul en-tete rend le meme axe des le premier tour.
	if got := axeAuPointDArret(make([]byte, entetePaquet-1), nil); got != "fin_de_tampon" {
		t.Errorf("tampon plus court qu'un en-tete : axe attendu fin_de_tampon, obtenu %q", got)
	}
}

func TestAxeFluxTronque(t *testing.T) {
	corps := slices.Concat(paquetBrut(0, 1000, bytes.Repeat([]byte("z"), 200)))
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(corps); err != nil {
		t.Fatalf("compression : %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("fermeture : %v", err)
	}
	// On coupe la somme de controle : le flux rend TOUS ses octets, PUIS une erreur.
	tronque := buf.Bytes()[:buf.Len()-4]
	dec, vueObj, estTronque := inflateMesure(tronque)
	if !estTronque {
		t.Fatal("un flux ampute de sa somme de controle doit etre vu comme tronque")
	}
	if !bytes.Equal(dec, corps) {
		t.Errorf("filmdec/killsource doivent garder le PARTIEL decompresse (%d octets sur %d)",
			len(dec), len(corps))
	}
	if !bytes.Equal(vueObj, tronque) {
		t.Error("objectiveevents doit marcher le BRUT COMPRESSE — c'est le quatrieme axe")
	}
	// Et un flux sain ne doit pas etre signale.
	if _, _, t2 := inflateMesure(buf.Bytes()); t2 {
		t.Error("un flux complet ne doit pas etre signale tronque")
	}
}

func TestUnChunkNonCompresseTraverseTelQuel(t *testing.T) {
	brut := paquetBrut(0, 1, []byte("nn"))
	dec, vueObj, tronque := inflateMesure(brut)
	if tronque || !bytes.Equal(dec, brut) || !bytes.Equal(vueObj, brut) {
		t.Errorf("un chunk non compresse doit traverser tel quel (tronque=%v)", tronque)
	}
}

func TestLigneWalkersRendUnTiretSansDivergence(t *testing.T) {
	l := ligneWalkers{film: "000d5950", chunks: 28}
	if got, want := l.String(), "000d5950\t28\t0\t0\t0\t0\t-"; got != want {
		t.Errorf("ligne attendue %q, obtenue %q", want, got)
	}
}
