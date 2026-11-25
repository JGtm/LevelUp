package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Les tests de ce fichier travaillent sur des banques et des paquets FABRIQUES : aucun octet du jeu.

func u32le(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

// objetFab fabrique un objet HIRC : type, taille (identifiant compris), identifiant, données.
func objetFab(typ byte, id uint32, data []byte) []byte {
	b := []byte{typ}
	b = append(b, u32le(uint32(4+len(data)))...)
	b = append(b, u32le(id)...)
	return append(b, data...)
}

func hircFab(objets ...[]byte) []byte {
	b := u32le(uint32(len(objets)))
	for _, o := range objets {
		b = append(b, o...)
	}
	return b
}

func chunkFab(magic string, data []byte) []byte {
	return append(append([]byte(magic), u32le(uint32(len(data)))...), data...)
}

// banqueFab : une banque d'une langue, avec ses objets et ses médias embarqués.
func banqueFab(langue uint32, hirc []byte, medias map[uint32][]byte) []byte {
	bkhd := append(append(u32le(1), u32le(2)...), u32le(langue)...)
	b := chunkFab("BKHD", bkhd)
	b = append(b, chunkFab("HIRC", hirc)...)
	if len(medias) > 0 {
		var didx, data []byte
		for _, id := range []uint32{1, 2, 3} {
			m, ok := medias[id]
			if !ok {
				continue
			}
			didx = append(didx, u32le(id)...)
			didx = append(didx, u32le(uint32(len(data)))...)
			didx = append(didx, u32le(uint32(len(m)))...)
			data = append(data, m...)
		}
		b = append(b, chunkFab("DIDX", didx)...)
		b = append(b, chunkFab("DATA", data)...)
	}
	return b
}

func TestMediasDeLEvenementDescendLaHierarchie(t *testing.T) {
	const ev, action, conteneur, sonA, sonB = 100, 101, 102, 103, 104
	son := func(media uint32) []byte { return append(make([]byte, 5), u32le(media)...) }
	hirc := hircFab(
		objetFab(hircEvenement, ev, append([]byte{1}, u32le(action)...)),
		objetFab(hircAction, action, append(make([]byte, 2), u32le(conteneur)...)),
		// un conteneur : un compte puis les enfants, après un en-tête d'une longueur quelconque
		objetFab(5, conteneur, append(append([]byte{9, 9, 9}, u32le(2)...), append(u32le(sonA), u32le(sonB)...)...)),
		objetFab(hircSon, sonA, son(7000)),
		objetFab(hircSon, sonB, son(6000)),
	)
	bs := separerBanques(banqueFab(fnv1("French(France)"), hirc, nil))
	if len(bs) != 1 {
		t.Fatalf("%d banque(s) lue(s), 1 attendue", len(bs))
	}
	got := mediasDeLEvenement(bs[0].Objets, ev)
	if want := []uint32{6000, 7000}; !reflect.DeepEqual(got, want) {
		t.Errorf("médias = %v, attendu %v (triés)", got, want)
	}
	if got := mediasDeLEvenement(bs[0].Objets, 999); len(got) != 0 {
		t.Errorf("évènement inconnu : %v médias, 0 attendu", got)
	}
}

func TestSeparerBanquesUneParLangue(t *testing.T) {
	hirc := hircFab(objetFab(hircSon, 1, append(make([]byte, 5), u32le(1)...)))
	d := append(banqueFab(fnv1("French(France)"), hirc, nil), banqueFab(fnv1("English(US)"), hirc, nil)...)
	bs := separerBanques(d)
	if len(bs) != 2 || bs[0].Langue != fnv1("French(France)") || bs[1].Langue != fnv1("English(US)") {
		t.Fatalf("banques = %+v, deux attendues, français puis anglais", bs)
	}
}

func TestFnv1IgnoreLaCasse(t *testing.T) {
	if fnv1("French(France)") != fnv1("french(france)") {
		t.Error("l'empreinte d'une langue ne doit pas dépendre de la casse")
	}
	if fnv1("French(France)") == fnv1("English(US)") {
		t.Error("deux langues, une empreinte")
	}
}

func TestMediasEmbarques(t *testing.T) {
	hirc := hircFab(objetFab(hircSon, 1, append(make([]byte, 5), u32le(1)...)))
	bs := separerBanques(banqueFab(1, hirc, map[uint32][]byte{1: []byte("aaa"), 3: []byte("cc")}))
	got := mediasEmbarques(bs[0])
	if len(got) != 2 || got[0].ID != 1 || string(got[0].Octet) != "aaa" || got[1].ID != 3 || string(got[1].Octet) != "cc" {
		t.Errorf("médias embarqués = %+v", got)
	}
}

// pckFab : un paquet à une table de flux de deux entrées (a, b), taille de bloc 1.
func pckFab(a, b []byte) []byte {
	const nEntrees, stride = 2, 20
	tStreams := 4 + nEntrees*stride
	debutDonnees := enteteFixePck + tStreams
	en := func(id uint32, offset, taille int) []byte {
		e := append(u32le(id), u32le(1)...)
		e = append(e, u32le(uint32(taille))...)
		e = append(e, u32le(uint32(offset))...)
		return append(e, u32le(0)...)
	}
	h := []byte(magicAKPK)
	h = append(h, u32le(uint32(debutDonnees))...)
	h = append(h, u32le(1)...)
	h = append(h, u32le(0)...) // langues
	h = append(h, u32le(0)...) // banques
	h = append(h, u32le(uint32(tStreams))...)
	h = append(h, u32le(0)...) // externes
	h = append(h, u32le(nEntrees)...)
	h = append(h, en(0xAAAA, debutDonnees, len(a))...)
	h = append(h, en(0xBBBB, debutDonnees+len(a), len(b))...)
	return append(append(h, a...), b...)
}

func TestLirePckRelitLesOctetsDeChaqueEntree(t *testing.T) {
	p := pckFab([]byte("premier"), []byte("second!"))
	ents, err := lirePck(bytes.NewReader(p), int64(len(p)))
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 2 {
		t.Fatalf("%d entrées, 2 attendues", len(ents))
	}
	for _, e := range ents {
		got := string(p[e.Offset : e.Offset+int64(e.Taille)])
		want := map[uint32]string{0xAAAA: "premier", 0xBBBB: "second!"}[e.ID]
		if got != want {
			t.Errorf("entrée %x : %q, attendu %q", e.ID, got, want)
		}
	}
}

func TestLirePckRefuseUnFichierQuiNEstPasUnPaquet(t *testing.T) {
	if _, err := lirePck(bytes.NewReader(make([]byte, 64)), 64); err == nil {
		t.Error("un fichier sans AKPK doit être refusé")
	}
	p := pckFab([]byte("a"), []byte("b"))
	if _, err := lirePck(bytes.NewReader(p[:30]), 30); err == nil {
		t.Error("un en-tête plus long que le fichier doit être refusé")
	}
}

func TestLirePckFichierEtIndexPaquets(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sb_voix.pck"), pckFab([]byte("x"), []byte("y")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "casse.pck"), []byte("pas un paquet"), 0o644); err != nil {
		t.Fatal(err)
	}
	idx, err := indexPaquets(dir, journalDeTest(t))
	if err != nil {
		t.Fatal(err)
	}
	if idx[0xAAAA] != "sb_voix.pck" || idx[0xBBBB] != "sb_voix.pck" || len(idx) != 2 {
		t.Errorf("index = %v : un paquet illisible doit être écarté, pas arrêter l'index", idx)
	}
}
