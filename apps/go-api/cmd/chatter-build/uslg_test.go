package main

import (
	"encoding/binary"
	"testing"
)

// ucshFab fabrique un sous-fichier `ucsh` à zéro dépendance : l'en-tête de 0x50 octets, la table des trois
// blocs (taille, section, offset), puis le bloc d'index {clé, offset} et le bloc de chaînes.
func ucshFab(textes map[uint32]string) []byte {
	var index, chaines []byte
	for _, cle := range []uint32{0xAAAA0001, 0xAAAA0002} {
		t, ok := textes[cle]
		if !ok {
			continue
		}
		index = append(index, u32le(cle)...)
		index = append(index, u32le(uint32(len(chaines)))...)
		chaines = append(append(chaines, []byte(t)...), 0)
	}
	entete := ucshEnteteFixe + 3*ucshBlocStride
	b := make([]byte, entete)
	copy(b, magicUcsh)
	binary.LittleEndian.PutUint32(b[ucshOffEntete:], uint32(entete))
	poser := func(k, taille, pos int) {
		o := ucshEnteteFixe + k*ucshBlocStride
		binary.LittleEndian.PutUint32(b[o+ucshOffBlocTaill:], uint32(taille))
		binary.LittleEndian.PutUint32(b[o+ucshOffBlocPos:], uint32(pos))
	}
	poser(0, 0, 0)
	poser(1, len(index), 0)
	poser(2, len(chaines), len(index))
	b = append(b, index...)
	b = append(b, chaines...)
	// padding : la taille minimale d'un sous-fichier lisible
	for len(b) < ucshTailleMini {
		b = append(b, 0)
	}
	return b
}

func TestTableUcshLitLesChainesParCle(t *testing.T) {
	got := tableUcsh(ucshFab(map[uint32]string{0xAAAA0001: "Ennemi aperçu !", 0xAAAA0002: "Un Spartan"}))
	if got[0xAAAA0001] != "Ennemi aperçu !" || got[0xAAAA0002] != "Un Spartan" || len(got) != 2 {
		t.Errorf("table = %v", got)
	}
}

func TestTableUcshIllisibleRendUneTableVide(t *testing.T) {
	if got := tableUcsh([]byte("ucsh")); len(got) != 0 {
		t.Errorf("sous-fichier trop court : %v", got)
	}
}

func TestSousFichiersUcshCoupeALaSignature(t *testing.T) {
	a, b := ucshFab(map[uint32]string{0xAAAA0001: "un"}), ucshFab(map[uint32]string{0xAAAA0001: "deux"})
	parts := sousFichiersUcsh(append(append([]byte{}, a...), b...))
	if len(parts) != 2 || tableUcsh(parts[0])[0xAAAA0001] != "un" || tableUcsh(parts[1])[0xAAAA0001] != "deux" {
		t.Fatalf("%d sous-fichier(s), deux attendus, dans l'ordre", len(parts))
	}
}

// sndFab fabrique un tag `snd!` : l'en-tête, deux dépendances (banque, liste de chaînes) puis des mots.
func sndFab(banque, liste uint32, mots ...uint32) []byte {
	b := make([]byte, sndTailleEntete)
	copy(b, magicUcsh)
	binary.LittleEndian.PutUint32(b[sndOffNbDeps:], 2)
	dep := func(groupe string, id uint32) []byte {
		d := make([]byte, ucshDepStride)
		d[0], d[1], d[2], d[3] = groupe[3], groupe[2], groupe[1], groupe[0]
		binary.LittleEndian.PutUint32(d[sndOffDepID:], id)
		return d
	}
	b = append(b, dep(groupeBanque, banque)...)
	b = append(b, dep(groupeListe, liste)...)
	for _, m := range mots {
		b = append(b, u32le(m)...)
	}
	return b
}

func TestDependancesSndEtMots(t *testing.T) {
	d := sndFab(0xB0B0, 0x1157, 1, 2, 3)
	groupes, ids := dependancesSnd(d)
	if len(groupes) != 2 || groupes[0] != groupeBanque || groupes[1] != groupeListe || ids[0] != 0xB0B0 || ids[1] != 0x1157 {
		t.Fatalf("dépendances = %v %v", groupes, ids)
	}
	if mots := motsSnd(d, 2); len(mots) != 3 || mots[2] != 3 {
		t.Errorf("mots = %v", mots)
	}
	if g, _ := dependancesSnd([]byte("trop court")); g != nil {
		t.Error("un tag sans signature n'a pas de dépendance")
	}
}
